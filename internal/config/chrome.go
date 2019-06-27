package config

import (
	"context"
	"encoding/json"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
	"github.com/pubgo/errors"
	"time"
)

type HeadlessResponse struct {
	Status    int               `json:"status,omitempty"`
	Content   string            `json:"content,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Latency   float64           `json:"latency,omitempty"`
	StartTime int64             `json:"start_time,omitempty"`
}

var devt *devtool.DevTools

func Response(url string, tmt time.Duration, fn func(resp *HeadlessResponse)) {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cfg := Default()

	if devt != nil {
		_, err := devt.Version(ctx)
		errors.Panic(err)
	} else {
		devt = devtool.New(cfg.chromeUrl)
		errors.Retry(10, func() {
			_v, err := devt.Version(ctx)
			errors.Panic(err)
			errors.P(_v)
		})
	}

	pt, err := devt.Get(ctx, devtool.Page)
	if err != nil {
		pt, err = devt.Create(ctx)
		errors.Panic(err)
	}

	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	errors.Wrap(err, "mth Reconnect rpcc.DialContext")

	c := cdp.NewClient(conn)

	errors.Wrap(c.Page.Enable(ctx), "mth Reconnect Page.Enable")
	errors.Wrap(c.Network.Enable(ctx, nil), "mth Reconnect Network.Enable")

	headers := make(map[string]string)
	headersStr, err := json.Marshal(headers)
	errors.Wrap(err, "mth Reconnect json.Marshal")

	errors.Wrap(c.Network.SetExtraHTTPHeaders(ctx, network.NewSetExtraHTTPHeadersArgs(headersStr)), "mth Reconnect Network.SetExtraHTTPHeaders")

	timeStart := time.Now()

	_, err = c.Page.Navigate(ctx, page.NewNavigateArgs(url))
	errors.Wrap(err, "mth Response Page.Navigate")

	networkResponse, err := c.Network.ResponseReceived(ctx)
	errors.Wrap(err, "mth Response Network.ResponseReceived")

	responseReply, err := networkResponse.Recv()
	errors.Wrap(err, "mth Response networkResponse.Recv")

	domContent, err := c.Page.DOMContentEventFired(ctx)
	errors.Wrap(err, "mth Response Page.DOMContentEventFired")
	defer domContent.Close()

	// 等待
	time.Sleep(time.Second * tmt)

	_, err = domContent.Recv()
	errors.Wrap(err, "mth Response domContent.Recv")

	doc, err := c.DOM.GetDocument(ctx, nil)
	errors.Wrap(err, "mth Response DOM.GetDocument")

	result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	errors.Wrap(err, "GetOuterHTML error")

	elapsed := float64(time.Since(timeStart)) / float64(time.Duration(1*time.Millisecond))

	responseHeaders := make(map[string]string)
	errors.Wrap(json.Unmarshal(responseReply.Response.Headers, &responseHeaders), "mth Response json.Unmarshal")
	fn(&HeadlessResponse{
		Content:   result.OuterHTML,
		Status:    responseReply.Response.Status,
		Headers:   responseHeaders,
		Latency:   elapsed,
		StartTime: timeStart.Unix(),
	})
}
