package config

import (
	"github.com/pubgo/errors"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type _config struct {
	chromes   chan *Ccs
	reChromes chan *Ccs
	count     int64
	Debug     bool
	chromeUrl string
}

func (t *_config) ChromePop(fn func(*Ccs)) {
	defer errors.Handle(func() {})

	for {
		select {
		case c := <-t.chromes:
			errors.ErrHandle(errors.Try(fn, c)(func() {
				t.chromes <- c
			}), func(err *errors.Err) {
				// 放入重试队列
				t.reChromes <- c
				errors.Wrap(err, "chrome 执行失败")
			})
			return
		case <-time.NewTimer(time.Minute).C:
			errors.Panic("获取chrome超时")
		}
	}

}

func (t *_config) Init() {
	defer errors.Handle(func() {})

	if _d, ok := os.LookupEnv("debug"); ok {
		t.Debug = _d == "true" || _d == "1" || _d == "ok"
	}

	if _d, ok := os.LookupEnv("count"); ok {
		a1, err := strconv.Atoi(_d)
		errors.Wrap(err, "parse count error")
		if a1 < 1 {
			a1 = 1
		}
		c.count = int64(a1)
	}

	// 初始化chrome
	go t.initChrome()
}

func (t *_config) Count() int64 {
	return t.count
}

func (t *_config) ChromeCount() int {
	return len(t.chromes)
}

func (t *_config) CheckChrome() {
	errors.ErrHandle(errors.Try(errors.Retry, 3, func() {
		resp, err := http.Get(cnst.ChromeUrl + "/json/version")
		errors.Wrap(err, "http get (%s) error", resp.Request.URL.String())
		errors.T(resp.StatusCode != http.StatusOK, "check code error")
		errors.Panic(resp.Body.Close())
	}), func(err *errors.Err) {
		//	chrome 重启获取服务重启
		err.P()
		t.killChrome()
		t.initChrome()
	})
}

// chrome健康检查
func (t *_config) Check() {
	go func() {
		defer errors.Handle(func() {})
		for {
			select {
			case <-time.NewTimer(time.Second * 5).C:
				errors.ErrHandle(errors.Try(errors.Retry, 3, func() {
					resp, err := http.Get(t.chromeUrl + "/json/version")
					errors.Wrap(err, "http get (%s) error", resp.Request.URL.String())
					errors.T(resp.StatusCode != http.StatusOK, "check code error")
					errors.Panic(resp.Body.Close())
				}), func(err *errors.Err) {
					//	chrome 重启获取服务重启
					t.killChrome()
					t.initChrome()
				})

			case c := <-t.reChromes:
				go func(_c *Ccs) {
					errors.ErrHandle(errors.Try(c.Reconnect)(func() {
						t.chromes <- c
					}), func(err *errors.Err) {
						err.P()
					})
				}(c)
			}
		}
	}()
}

var once sync.Once
var c *_config

func Default() *_config {
	once.Do(func() {
		c = &_config{
			count: 10,
			Debug: true,
		}
	})
	return c
}
