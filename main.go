package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pubgo/dhtml/internal/config"
	"github.com/pubgo/errors"
	"math/big"
	"net/http"
	"time"
)

func main() {
	cfg := config.Default()
	cfg.Init()

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger())

	r.GET("/", func(ctx *gin.Context) {
		url, _ := ctx.GetQuery("url")
		timeOut, _ := ctx.GetQuery("time_out")

		tm := 2
		if timeOut != "" {
			a1, bool := big.NewInt(0).SetString(timeOut, 10)
			if !bool {
				ctx.String(http.StatusBadRequest, fmt.Sprintf("time out parse error: %s", timeOut))
				return
			}
			tm = int(a1.Int64())
		}

		for _, c := range cfg.Chromes {
			if c.Url == "" && c.C != nil {
				errors.ErrHandle(errors.Try(func() {
					c.Response(url, time.Duration(tm), func(s string) {
						ctx.String(http.StatusOK, s)
					})
				}), func(err *errors.Err) {
					go c.Close()
					go c.Reconnect()
					ctx.String(http.StatusBadRequest, err.Error())
				})
			}
			c.Url = ""
			return
		}

		ctx.String(http.StatusBadRequest, "超过最大并发量, 请等待")
		return
	})

	errors.Wrap(r.Run(), "服务器未知错误, 重启")
}
