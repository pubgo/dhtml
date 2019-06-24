package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pubgo/dhtml/internal/config"
	"github.com/pubgo/errors"
	"net/http"
	nurl "net/url"
	"strconv"
	"time"
)

func main() {
	cfg := config.Default()
	cfg.Init()
	go cfg.Check()

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.GET("/", func(ctx *gin.Context) {
		defer errors.Resp(func(err *errors.Err) {
			ctx.JSON(http.StatusBadRequest, err.StackTrace())
		})

		url, _ := ctx.GetQuery("url")

		errors.T(url == "", "url(%s) is null", url)

		_, err := nurl.Parse(url)
		errors.Wrap(err, "url(%s) parse error", url)

		timeOut, _ := ctx.GetQuery("time_out")

		tm := 2
		if timeOut != "" {
			a1, err := strconv.Atoi(timeOut)
			errors.Wrap(err, "time out parse error: %s", timeOut)
			tm = a1
		}

		errors.T(cfg.ChromeCount() == 0, "超过最大并发量(%d), 请等待", cfg.Count())

		cfg.ChromePop(func(c *config.Ccs) {
			c.Response(url, time.Duration(tm), func(resp *config.HeadlessResponse) {
				ctx.JSON(http.StatusOK, resp)
			})
		})
	})

	errors.Wrap(r.Run(), "服务器未知错误")
}
