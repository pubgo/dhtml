package config

import (
	"math/big"
	"sync"
)

type _config struct {
	Chromes []*Ccs
	Count   int64
}

func (t *_config) Init() {
	go t.InitChrome()

	for i := 0; i < int(t.Count); i++ {
		c := &Ccs{tx: &sync.Mutex{}}
		c.Loop()

		c.Reconnect()

		t.Chromes = append(t.Chromes, c)
	}
}

var once sync.Once
var c *_config

func Default() *_config {
	once.Do(func() {
		c = &_config{
			Count: 10,
		}
		c.Chromes = []*Ccs{}

		if e := env("count"); e != "" {
			a1, _ := big.NewInt(0).SetString(e, 10)
			c.Count = a1.Int64()
			if c.Count < 1 {
				c.Count = 1
			}
		}
	})
	return c
}
