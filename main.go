package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pubgo/dhtml/internal/config"
	"github.com/pubgo/dhtml/version"
	"github.com/pubgo/errors"
	"net/http"
	nurl "net/url"
	"strconv"
	"sync"
	"time"
)

func main() {
	cfg := config.Default()
	cfg.Init()

	r := gin.New()
	r.GET("/version", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"version": version.Version,
			"build":   version.BuildVersion,
			"commit":  version.GitCommit,
		})
	})

	tx := &sync.Mutex{}
	r.GET("/", func(ctx *gin.Context) {
		tx.Lock()
		defer tx.Unlock()

		defer errors.Resp(func(err *errors.Err) {
			ctx.JSON(http.StatusBadRequest, err.StackTrace())
			errors.Panic(err)
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

		config.Response(url, time.Duration(tm), func(resp *config.HeadlessResponse) {
			ctx.JSON(http.StatusOK, resp)
		})
	})

	errors.Wrap(r.Run(), "服务器未知错误")
}
