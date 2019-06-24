package config

import (
	"errors"
	"github.com/json-iterator/go"
	"math/rand"
	"os"
	"time"
)

var env = os.Getenv
var json = jsoniter.ConfigCompatibleWithStandardLibrary

func init() {
	rand.Seed(time.Now().Unix())
}

type M map[string]interface{}

func (t M) String() error {
	d, _ := json.Marshal(t)
	return errors.New(string(d))
}
