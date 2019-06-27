package config

import (
	"github.com/pubgo/errors"
	"os"
	"sync"
)

type _config struct {
	count     int64
	Debug     bool
	chromeUrl string
}

func (t *_config) Init() {
	defer errors.Handle(func() {})

	if _d, ok := os.LookupEnv("debug"); ok {
		t.Debug = _d == "true" || _d == "1" || _d == "ok"
	}

	if _d, ok := os.LookupEnv("chrome_url"); ok {
		t.chromeUrl = _d
	}

}

func (t *_config) Count() int64 {
	return t.count
}

var once sync.Once
var c *_config

func Default() *_config {
	once.Do(func() {
		c = &_config{
			count:     10,
			Debug:     true,
			chromeUrl: "http://localhost:9222",
		}
	})
	return c
}
