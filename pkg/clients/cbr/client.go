package cbr

import (
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

type Client struct {
	client *fasthttp.Client
}

func NewClient() *Client {
	return &Client{
		client: &fasthttp.Client{},
	}
}

func (c *Client) GetCurrencies() (*Response, error) {
	respBody, err := c.getCurrencies()
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := resp.unmarshalJSON(respBody); err != nil {
		return nil, err
	}

	return &resp, nil
}

const (
	cbrUrl            = "https://www.cbr-xml-daily.ru/daily_json.js"
	cbrRequestTimeout = 5 * time.Second
)

func (c *Client) getCurrencies() ([]byte, error) {
	httpReq := fasthttp.AcquireRequest()
	httpResp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseResponse(httpResp)
		fasthttp.ReleaseRequest(httpReq)
	}()

	httpReq.Header.SetMethod(fasthttp.MethodGet)
	httpReq.SetRequestURI(cbrUrl)

	if err := c.client.DoTimeout(httpReq, httpResp, cbrRequestTimeout); err != nil {
		return nil, err
	}

	if httpResp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("unexpected response status code: %d", httpResp.StatusCode())
	}

	respBody := append([]byte{}, httpResp.Body()...)

	return respBody, nil
}
