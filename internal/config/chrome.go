package config

import (
	"context"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
	"github.com/pubgo/dhtml/internal/cnst"
	"github.com/pubgo/errors"
	"io/ioutil"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type HeadlessResponse struct {
	Status    int               `json:"status"`
	Content   string            `json:"content"`
	Headers   map[string]string `json:"headers"`
	Latency   float64           `json:"latency"`
	Url       string            `json:"url"`
	StartTime int64             `json:"start_time"`
}

type Ccs struct {
	tx  *sync.Mutex
	tgt *devtool.Target
	Url string
	C   *cdp.Client
}

func checkHeadless(arg string) {
	defer errors.Handle(func() {})

	errors.Retry(3, func() {
		resp, err := http.Get(arg + "/json/version")
		errors.Wrap(err, "http get (%s) error", resp.Request.URL.String())
		errors.T(resp.StatusCode != http.StatusOK, "check code error")
		errors.Panic(resp.Body.Close())
	})

}

func (t *Ccs) Loop() {
	go errors.Ticker(func(dur time.Time) time.Duration {
		checkHeadless(cnst.ChromeUrl)
		return time.Second * 10
	})

}

func (t *Ccs) ResponseImage(url string, tmt time.Duration, fn func(string)) {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	_, err := t.C.Page.Navigate(ctx, page.NewNavigateArgs(url))
	errors.Wrap(err, "mth Response Page.Navigate")

	domContent, err := t.C.Page.DOMContentEventFired(ctx)
	errors.Wrap(err, "mth Response Page.DOMContentEventFired")
	defer domContent.Close()

	_, err = domContent.Recv()
	errors.Panic(err)

	// 等待
	time.Sleep(time.Second * tmt)

	screenshotName := "screenshot.jpg"
	screenshotArgs := page.NewCaptureScreenshotArgs().SetFormat("jpeg").SetQuality(80)
	screenshot, err := t.C.Page.CaptureScreenshot(ctx, screenshotArgs)
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

func (t *Ccs) Response(url string, tmt time.Duration, fn func(string)) {
	defer errors.Handle(func() {})

	t.tx.Lock()
	defer t.tx.Unlock()
	t.Url = url

	t.response(url, tmt, fn)
}

func (t *Ccs) response(url string, tmt time.Duration, fn func(string)) {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	timeStart := time.Now()

	_, err := t.C.Page.Navigate(ctx, page.NewNavigateArgs(url))
	errors.Wrap(err, "mth Response Page.Navigate")

	networkResponse, err := t.C.Network.ResponseReceived(ctx)
	errors.Wrap(err, "mth Response Network.ResponseReceived")

	responseReply, err := networkResponse.Recv()
	errors.Wrap(err, "mth Response networkResponse.Recv")

	domContent, err := t.C.Page.DOMContentEventFired(ctx)
	errors.Wrap(err, "mth Response Page.DOMContentEventFired")
	defer domContent.Close()

	// 等待
	time.Sleep(time.Second * tmt)

	_, err = domContent.Recv()
	errors.Wrap(err, "mth Response domContent.Recv")

	doc, err := t.C.DOM.GetDocument(ctx, nil)
	errors.Wrap(err, "mth Response DOM.GetDocument")

	result, err := t.C.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	errors.Wrap(err, "GetOuterHTML error")

	elapsed := float64(time.Since(timeStart)) / float64(time.Duration(1*time.Millisecond))

	responseHeaders := make(map[string]string)
	errors.Wrap(json.Unmarshal(responseReply.Response.Headers, &responseHeaders), "mth Response json.Unmarshal")

	dd, err := json.Marshal(&HeadlessResponse{
		Content:   result.OuterHTML,
		Status:    responseReply.Response.Status,
		Headers:   responseHeaders,
		Latency:   elapsed,
		Url:       url,
		StartTime: timeStart.Unix(),
	})
	errors.Wrap(err, "mth Response json.Marshal")

	fn(string(dd))
}

func (t *Ccs) Close() {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	t.C = nil
	errors.Panic(devtool.New(cnst.ChromeUrl).Close(ctx, t.tgt))
}

func (t *Ccs) Reconnect() {
	defer errors.Handle(func() {})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	devt := devtool.New(cnst.ChromeUrl)
	pt, err := devt.Create(ctx)
	errors.Wrap(err, "mth Reconnect devt.Create")

	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	errors.Wrap(err, "mth Reconnect rpcc.DialContext")

	t.tgt = pt
	t.C = cdp.NewClient(conn)
	t.Url = ""

	domContent, err := t.C.Page.DOMContentEventFired(ctx)
	errors.Panic(err)
	defer domContent.Close()

	errors.Wrap(t.C.Page.Enable(ctx), "mth Reconnect Page.Enable")

	errors.Wrap(t.C.Network.Enable(ctx, nil), "mth Reconnect Network.Enable")

	headers := make(map[string]string)
	headersStr, err := json.Marshal(headers)
	errors.Wrap(err, "mth Reconnect json.Marshal")

	errors.Wrap(t.C.Network.SetExtraHTTPHeaders(ctx, network.NewSetExtraHTTPHeadersArgs(headersStr)), "mth Reconnect Network.SetExtraHTTPHeaders")
}

func (t *_config) InitChrome() {
	errors.Handle(func() {})

	cmd := exec.Command(
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

	errors.Wrap(cmd.Run(), "run chrome error")
}
