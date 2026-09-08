package rate

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"currency-quotes/common"
	"currency-quotes/common/currencies"
	commonLogger "currency-quotes/common/logger"
	"currency-quotes/common/messages"
	rateModels "currency-quotes/common/models/rate"
	updateStatus "currency-quotes/common/update_status"
	workerCommon "currency-quotes/infrastructure/worker/common"
	"currency-quotes/infrastructure/worker/rate/config"
	"currency-quotes/pkg/clients/cbr"
)

var (
	errMalformedCurrency   = errors.New("malformed currency code")
	errUnsupportedCurrency = errors.New("unsupported currency")
	errRateUnavailable     = errors.New("rate unavailable")
	errUpdateIDRequired    = errors.New("update id is required")
)

type worker struct {
	config         *config.Config
	logger         commonLogger.Logger
	client         *cbr.Client
	updateInterval time.Duration

	ratesMu sync.RWMutex
	rates   map[currencies.Currency]rateModels.Rate
	queue   chan updateTask

	updatesMu sync.RWMutex
	updates   map[string]updateResult

	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
}

type updateResult struct {
	Code      string
	Status    updateStatus.UpdateStatus
	Price     float64
	UpdatedAt time.Time
	Error     string
}

func NewWorker(cfg *config.Config, lgr commonLogger.Logger) workerCommon.Worker {
	w := &worker{
		config:         cfg,
		logger:         lgr,
		client:         cbr.NewClient(),
		updates:        make(map[string]updateResult),
		updateInterval: cfg.UpdateInterval,
		queue:          make(chan updateTask, cfg.UpdateQueueCapacity),
	}

	return w
}

func (w *worker) Init(ctx context.Context) error {
	if err := w.refreshRates(); err != nil {
		return err
	}

	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	if w.cancel != nil {
		return errors.New("rate worker is already initialized")
	}

	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})

	go w.run(runCtx, w.done)

	return nil
}

func (w *worker) Stop(ctx context.Context) {
	w.lifecycleMu.Lock()
	cancel, done := w.cancel, w.done
	w.lifecycleMu.Unlock()

	if cancel == nil || done == nil {
		return
	}

	cancel()

	select {
	case <-done:
	case <-ctx.Done():
		w.logger.Warn("rate worker shutdown timed out", zap.Error(ctx.Err()))
	}
}

func (w *worker) CreateHTTPHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if recovered := recover(); recovered != nil {
				w.logger.Error("panic in rate HTTP handler", zap.Any("panic", recovered))

				common.WriteError(ctx, fasthttp.StatusInternalServerError, "internal server error")
			}
		}()

		w.handleHTTP(ctx)
	}
}

func (w *worker) handleHTTP(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	switch {
	case path == "/push":
		w.handlePush(ctx)
	case strings.HasPrefix(path, "/updates/"):
		w.handleUpdate(ctx, path)
	case strings.HasPrefix(path, "/rates/"):
		w.handleLatestRate(ctx, path)
	default:
		common.WriteError(ctx, fasthttp.StatusNotFound, "not found")
	}
}

func (w *worker) handleLatestRate(ctx *fasthttp.RequestCtx, path string) {
	if !ctx.IsGet() {
		ctx.Response.Header.Set("Allow", fasthttp.MethodGet)
		common.WriteError(ctx, fasthttp.StatusMethodNotAllowed, "method not allowed")

		return
	}

	code := strings.TrimPrefix(path, "/rates/")
	price, updatedAt, err := w.latestRate(code)
	if err != nil {
		switch {
		case errors.Is(err, errMalformedCurrency):
			common.WriteError(ctx, fasthttp.StatusBadRequest, err.Error())
		case errors.Is(err, errUnsupportedCurrency):
			common.WriteError(ctx, fasthttp.StatusNotFound, err.Error())
		case errors.Is(err, errRateUnavailable):
			common.WriteError(ctx, fasthttp.StatusServiceUnavailable, err.Error())
		default:
			common.WriteError(ctx, fasthttp.StatusInternalServerError, "internal server error")
		}

		return
	}

	common.WriteJSON(ctx, fasthttp.StatusOK, rateModels.LatestResponse{
		Price:     price,
		UpdatedAt: updatedAt,
	})
}

func (w *worker) handleUpdate(ctx *fasthttp.RequestCtx, path string) {
	if !ctx.IsGet() {
		common.WriteError(ctx, fasthttp.StatusMethodNotAllowed, "method not allowed")

		return
	}

	updateID := strings.TrimPrefix(path, "/updates/")
	if updateID == "" {
		common.WriteError(ctx, fasthttp.StatusBadRequest, errUpdateIDRequired.Error())

		return
	}

	w.updatesMu.RLock()
	result, ok := w.updates[updateID]
	w.updatesMu.RUnlock()
	if !ok {
		common.WriteError(ctx, fasthttp.StatusNotFound, "update not found")

		return
	}

	response := rateModels.UpdateResponse{Status: result.Status.String()}
	switch result.Status {
	case updateStatus.UpdateStatusSucceeded:
		response.Status = ""
		response.Price = &result.Price
		response.UpdatedAt = &result.UpdatedAt
	case updateStatus.UpdateStatusFailed:
		response.Error = result.Error
	}

	common.WriteJSON(ctx, fasthttp.StatusOK, response)
}

func (w *worker) handlePush(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		common.WriteError(ctx, fasthttp.StatusMethodNotAllowed, "method not allowed")

		return
	}

	var request rateModels.PushRequest
	if err := json.Unmarshal(ctx.PostBody(), &request); err != nil {
		common.WriteError(ctx, fasthttp.StatusBadRequest, "invalid request body")

		return
	}

	code := currencies.CodeByCurrency(request.Code)
	if code == currencies.CodeErr {
		common.WriteError(ctx, fasthttp.StatusBadRequest, errUnsupportedCurrency.Error())

		return
	}

	updateID := uuid.NewString()

	job := newJob(messages.RateMsg{
		MsgID: updateID,
		Code:  code.String(),
	}, w.client)

	w.updatesMu.Lock()
	w.updates[updateID] = updateResult{
		Code:   job.msg.Code,
		Status: updateStatus.UpdateStatusAccepted,
	}
	w.updatesMu.Unlock()

	if err := job.CreateTasks(w.queue); err != nil {
		w.updatesMu.Lock()
		delete(w.updates, updateID)
		w.updatesMu.Unlock()

		if errors.Is(err, errUpdateQueueFull) {
			common.WriteError(ctx, fasthttp.StatusServiceUnavailable, err.Error())
		} else {
			common.WriteError(ctx, fasthttp.StatusInternalServerError, "create update task")
		}

		return
	}

	common.WriteJSON(ctx, fasthttp.StatusAccepted, rateModels.PushResponse{UpdateID: updateID})
}

func (w *worker) run(ctx context.Context, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(w.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-w.queue:
			w.processUpdate(task)
		case <-ticker.C:
			if err := w.refreshRates(); err != nil {
				w.logger.Error("refresh currency rates", zap.Error(err))
			}
		}
	}
}

func (w *worker) processUpdate(task updateTask) {
	updateID := task.msg.ID()
	w.setUpdateRunning(updateID)

	if err := w.refreshRates(); err != nil {
		w.setUpdateFailed(updateID, err)
		w.logger.Error("process rate update", zap.String("update_id", updateID), zap.Error(err))

		return
	}

	price, updatedAt, err := w.latestRate(task.msg.Code)
	if err != nil {
		w.setUpdateFailed(updateID, err)
		w.logger.Error("calculate updated rate", zap.String("update_id", updateID), zap.Error(err))

		return
	}

	w.updatesMu.Lock()
	result, ok := w.updates[updateID]
	if ok {
		result.Status = updateStatus.UpdateStatusSucceeded
		result.Price = price
		result.UpdatedAt = updatedAt
		w.updates[updateID] = result
	}
	w.updatesMu.Unlock()
}

func (w *worker) setUpdateRunning(updateID string) {
	w.updatesMu.Lock()
	result, ok := w.updates[updateID]
	if ok {
		result.Status = updateStatus.UpdateStatusRunning
		w.updates[updateID] = result
	}
	w.updatesMu.Unlock()
}

func (w *worker) setUpdateFailed(updateID string, updateErr error) {
	w.updatesMu.Lock()
	result, ok := w.updates[updateID]
	if ok {
		result.Status = updateStatus.UpdateStatusFailed
		result.Error = updateErr.Error()
		w.updates[updateID] = result
	}
	w.updatesMu.Unlock()
}

func (w *worker) refreshRates() error {
	w.logger.Info("refresh rates")

	actualCurrencies, err := w.client.GetCurrencies()
	if err != nil {
		return err
	}

	rates := map[currencies.Currency]rateModels.Rate{
		currencies.CodeRub: {
			Code:      currencies.CodeRub,
			Nominal:   1,
			Value:     currencies.RateValueScale,
			UpdatedAt: actualCurrencies.Date,
		},
	}

	for _, actualCurrency := range actualCurrencies.Valute {
		code := currencies.Currency(actualCurrency.NumCode)
		if code != currencies.CodeEur && code != currencies.CodeUSD {
			continue
		}

		rates[code] = rateModels.Rate{
			Code:      code,
			Nominal:   actualCurrency.Nominal,
			Value:     actualCurrency.Value,
			UpdatedAt: actualCurrencies.Date,
		}
	}

	w.ratesMu.Lock()
	w.rates = rates
	w.ratesMu.Unlock()

	return nil
}

func (w *worker) latestRate(currencyCode string) (float64, time.Time, error) {
	code := currencies.CodeByCurrency(currencyCode)
	if code == currencies.CodeErr {
		return 0, time.Time{}, errUnsupportedCurrency
	}

	w.ratesMu.RLock()
	rate, ok := w.rates[code]
	w.ratesMu.RUnlock()

	if !ok {
		return 0, time.Time{}, errRateUnavailable
	}

	price := float64(rate.Value) / float64(rate.Nominal) / currencies.RateValueScale

	return price, rate.UpdatedAt, nil
}
