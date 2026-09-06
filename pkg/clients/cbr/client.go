package cbr

import (
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
	cbrUrl = "https://www.cbr-xml-daily.ru/daily_json.js"
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

	if err := c.client.Do(httpReq, httpResp); err != nil {
		return nil, err
	}

	respBody := append([]byte{}, httpResp.Body()...)

	return respBody, nil
}
