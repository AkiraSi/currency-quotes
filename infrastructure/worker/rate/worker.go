package rate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"currency-quotes/common/currencies"
	commonLogger "currency-quotes/common/logger"
	commonRate "currency-quotes/common/rate"
	updateStatus "currency-quotes/common/update_status"
	"currency-quotes/infrastructure/worker/common"
	"currency-quotes/infrastructure/worker/rate/config"
	"currency-quotes/pkg/clients/cbr"
)

const (
	updateInterval      = time.Minute
	updateQueueCapacity = 50
)

var (
	errMalformedPair       = errors.New("malformed currency pair")
	errUnsupportedCurrency = errors.New("unsupported currency")
	errRateUnavailable     = errors.New("rate unavailable")
)

type worker struct {
	config         *config.Config
	logger         commonLogger.Logger
	client         *cbr.Client
	updateInterval time.Duration

	ratesMu sync.RWMutex
	rates   map[currencies.Currency]commonRate.Rate
	queue   chan updateTask

	updatesMu sync.RWMutex
	updates   map[string]updateResult

	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
}

type latestRateResponse struct {
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

type pushResponse struct {
	UpdateID string `json:"update_id"`
}

type updateResult struct {
	Pair      string
	Status    updateStatus.UpdateStatus
	Price     float64
	UpdatedAt time.Time
	Error     string
}

func NewWorker(cfg *config.Config, lgr commonLogger.Logger) common.Worker {
	return &worker{
		config:         cfg,
		logger:         lgr,
		client:         cbr.NewClient(),
		updateInterval: updateInterval,
		queue:          make(chan updateTask, updateQueueCapacity),
		updates:        make(map[string]updateResult),
	}
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

	pair := strings.TrimPrefix(path, "/rates/")
	price, updatedAt, err := w.latestRate(pair)
	if err != nil {
		switch {
		case errors.Is(err, errMalformedPair):
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

	common.WriteJSON(ctx, fasthttp.StatusOK, latestRateResponse{
		Price:     price,
		UpdatedAt: updatedAt,
	})
}

func (w *worker) handlePush(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.Response.Header.Set("Allow", fasthttp.MethodPost)
		common.WriteError(ctx, fasthttp.StatusMethodNotAllowed, "method not allowed")

		return
	}

	job := newJob(w.client)
	if err := job.processMessage(ctx.PostBody()); err != nil {
		common.WriteError(ctx, fasthttp.StatusBadRequest, "invalid request body")

		return
	}

	if job.msg.ID() == "" {
		common.WriteError(ctx, fasthttp.StatusBadRequest, "message id is required")

		return
	}

	base, quote, err := parsePair(job.msg.Pair)
	if err != nil {
		if errors.Is(err, errUnsupportedCurrency) {
			common.WriteError(ctx, fasthttp.StatusNotFound, err.Error())
		} else {
			common.WriteError(ctx, fasthttp.StatusBadRequest, err.Error())
		}

		return
	}

	job.msg.Pair = base.String() + "/" + quote.String()

	w.updatesMu.Lock()
	w.updates[job.msg.ID()] = updateResult{
		Pair:   job.msg.Pair,
		Status: updateStatus.UpdateStatusAccepted,
	}
	w.updatesMu.Unlock()

	if err := job.CreateTasks(w.queue); err != nil {
		w.updatesMu.Lock()
		delete(w.updates, job.msg.ID())
		w.updatesMu.Unlock()

		if errors.Is(err, errUpdateQueueFull) {
			common.WriteError(ctx, fasthttp.StatusServiceUnavailable, err.Error())
		} else {
			common.WriteError(ctx, fasthttp.StatusInternalServerError, "create update task")
		}

		return
	}

	common.WriteJSON(ctx, fasthttp.StatusAccepted, pushResponse{UpdateID: job.msg.ID()})
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

	if err := w.refreshRatesFrom(task.client); err != nil {
		w.setUpdateFailed(updateID, err)
		w.logger.Error("process rate update", zap.String("update_id", updateID), zap.Error(err))

		return
	}

	price, updatedAt, err := w.latestRate(task.msg.Pair)
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
		result.Error = ""
		w.updates[updateID] = result
	}
	w.updatesMu.Unlock()
}

func (w *worker) setUpdateRunning(updateID string) {
	w.updatesMu.Lock()
	result, ok := w.updates[updateID]
	if ok {
		result.Status = updateStatus.UpdateStatusRunning
		result.Error = ""
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
	return w.refreshRatesFrom(w.client)
}

func (w *worker) refreshRatesFrom(client *cbr.Client) error {
	actualCurrencies, err := client.GetCurrencies()
	if err != nil {
		return err
	}
	if actualCurrencies == nil {
		return errors.New("CBR client returned an empty response")
	}
	if actualCurrencies.Date.IsZero() {
		return errors.New("CBR client returned an empty timestamp")
	}

	rates := map[currencies.Currency]commonRate.Rate{
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
		if actualCurrency.Nominal <= 0 || actualCurrency.Value <= 0 {
			return fmt.Errorf("invalid %s rate", code.String())
		}

		rates[code] = commonRate.Rate{
			Code:      code,
			Nominal:   actualCurrency.Nominal,
			Value:     actualCurrency.Value,
			UpdatedAt: actualCurrencies.Date,
		}
	}
	for _, code := range []currencies.Currency{currencies.CodeEur, currencies.CodeUSD} {
		if _, ok := rates[code]; !ok {
			return fmt.Errorf("%s rate is missing", code.String())
		}
	}

	w.ratesMu.Lock()
	w.rates = rates
	w.ratesMu.Unlock()

	return nil
}

func (w *worker) latestRate(pair string) (float64, time.Time, error) {
	base, quote, err := parsePair(pair)
	if err != nil {
		return 0, time.Time{}, err
	}

	w.ratesMu.RLock()
	baseRate, baseOK := w.rates[base]
	quoteRate, quoteOK := w.rates[quote]
	w.ratesMu.RUnlock()

	if !baseOK || !quoteOK || baseRate.Nominal <= 0 || quoteRate.Nominal <= 0 || baseRate.Value <= 0 || quoteRate.Value <= 0 {
		return 0, time.Time{}, errRateUnavailable
	}

	baseRubValue := float64(baseRate.Value) / float64(baseRate.Nominal)
	quoteRubValue := float64(quoteRate.Value) / float64(quoteRate.Nominal)
	updatedAt := baseRate.UpdatedAt
	if quoteRate.UpdatedAt.Before(updatedAt) {
		updatedAt = quoteRate.UpdatedAt
	}

	return baseRubValue / quoteRubValue, updatedAt, nil
}

func parsePair(pair string) (currencies.Currency, currencies.Currency, error) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(pair)), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return currencies.CodeErr, currencies.CodeErr, errMalformedPair
	}

	base := currencies.CodeByCurrency(parts[0])
	quote := currencies.CodeByCurrency(parts[1])
	if base == currencies.CodeErr || quote == currencies.CodeErr {
		return currencies.CodeErr, currencies.CodeErr, errUnsupportedCurrency
	}

	return base, quote, nil
}
