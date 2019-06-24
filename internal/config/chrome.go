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
	"github.com/pubgo/dhtml/internal/cnst"
	"github.com/pubgo/errors"
	"io/ioutil"
	"os/exec"
	"sync"
	"time"
)

type HeadlessResponse struct {
	Status    int               `json:"status,omitempty"`
	Content   string            `json:"content,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Latency   float64           `json:"latency,omitempty"`
	StartTime int64             `json:"start_time,omitempty"`
}

type Ccs struct {
	tx  *sync.Mutex
	tgt *devtool.Target
	c   *cdp.Client
}

func (t *Ccs) ResponseImage(url string, tmt time.Duration, fn func(string)) {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	_, err := t.c.Page.Navigate(ctx, page.NewNavigateArgs(url))
	errors.Wrap(err, "mth Response Page.Navigate")

	domContent, err := t.c.Page.DOMContentEventFired(ctx)
	errors.Wrap(err, "mth Response Page.DOMContentEventFired")
	defer domContent.Close()

	_, err = domContent.Recv()
	errors.Panic(err)

	// 等待
	time.Sleep(time.Second * tmt)

	screenshotName := "screenshot.jpg"
	screenshotArgs := page.NewCaptureScreenshotArgs().SetFormat("jpeg").SetQuality(80)
	screenshot, err := t.c.Page.CaptureScreenshot(ctx, screenshotArgs)
	errors.Panic(err)

	errors.Panic(ioutil.WriteFile(screenshotName, screenshot.Data, 0644))

	//if err = t.C.Page.StartScreencast(ctx, page.NewStartScreencastArgs().SetEveryNthFrame(1).SetFormat("png")); err != nil {
	//	return err
	//}

	// Random delay for our screencast.

	//err = t.C.Page.StopScreencast(ctx)
	//if err != nil {
	//	return err
	//}

	//fn(string(dd))
}

func (t *Ccs) Response(url string, tmt time.Duration, fn func(resp *HeadlessResponse)) {
	defer errors.Handle(func() {})

	t.tx.Lock()
	defer t.tx.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	timeStart := time.Now()

	_, err := t.c.Page.Navigate(ctx, page.NewNavigateArgs(url))
	errors.Wrap(err, "mth Response Page.Navigate")

	networkResponse, err := t.c.Network.ResponseReceived(ctx)
	errors.Wrap(err, "mth Response Network.ResponseReceived")

	responseReply, err := networkResponse.Recv()
	errors.Wrap(err, "mth Response networkResponse.Recv")

	domContent, err := t.c.Page.DOMContentEventFired(ctx)
	errors.Wrap(err, "mth Response Page.DOMContentEventFired")
	defer domContent.Close()

	// 等待
	time.Sleep(time.Second * tmt)

	_, err = domContent.Recv()
	errors.Wrap(err, "mth Response domContent.Recv")

	doc, err := t.c.DOM.GetDocument(ctx, nil)
	errors.Wrap(err, "mth Response DOM.GetDocument")

	result, err := t.c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
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

func (t *Ccs) Reconnect() {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	devt := devtool.New(cnst.ChromeUrl)

	// 关闭连接
	errors.Wrap(devtool.New(cnst.ChromeUrl).Close(ctx, t.tgt), "chrome关闭连接失败")

	pt, err := devt.Create(ctx)
	errors.Wrap(err, "mth Reconnect devt.Create")

	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	errors.Wrap(err, "mth Reconnect rpcc.DialContext")

	t.tgt = pt
	t.c = cdp.NewClient(conn)

	domContent, err := t.c.Page.DOMContentEventFired(ctx)
	errors.Panic(err)
	defer domContent.Close()

	errors.Wrap(t.c.Page.Enable(ctx), "mth Reconnect Page.Enable")

	errors.Wrap(t.c.Network.Enable(ctx, nil), "mth Reconnect Network.Enable")

	headers := make(map[string]string)
	headersStr, err := json.Marshal(headers)
	errors.Wrap(err, "mth Reconnect json.Marshal")

	errors.Wrap(t.c.Network.SetExtraHTTPHeaders(ctx, network.NewSetExtraHTTPHeadersArgs(headersStr)), "mth Reconnect Network.SetExtraHTTPHeaders")
}

func (t *_config) killChrome() {
	defer errors.Handle(func() {})

	errors.Wrap(t.chrome.Process.Kill(), "chrome kill error")
}

func (t *_config) initChrome() {
	errors.Handle(func() {})

	t.chrome = exec.Command(
		"/tini",
		"--",
		"google-chrome",
		"--headless",
		"--no-sandbox",
		"--verbose=error",
		"--disable-setuid-sandbox",
		"--disable-new-tab-first-run",
		"--disable-translate",
		"--no-first-run",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		"--remote-debugging-address=0.0.0.0",
		"--remote-debugging-port=9222",
		"--disable-remote-fonts",
		"--user-data-dir=/tmp",
		"--crash-dumps-dir=/tmp",
	)

	//cmd.Stdout = os.Stderr
	//cmd.Stderr = os.Stderr

	errors.Wrap(t.chrome.Run(), "run chrome error")
}
