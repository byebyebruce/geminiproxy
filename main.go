package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/byebyebruce/geminiproxy/gemproxy"

	"github.com/gin-gonic/gin"
	"github.com/lmittmann/tint"
)

var (
	addr         = flag.String("addr", ":8085", "server address")
	httpProxyURL = flag.String("proxy", "", "proxy url. eg. http://127.0.0.1:1080")
)

func init() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelDebug,
			AddSource:  true,
			TimeFormat: "2006/01/02 15:04:05.000",
		}),
	))
}

func main() {
	flag.Parse()

	var httpProxy *http.Transport
	if *httpProxyURL != "" {
		proxyURL, err := url.Parse(*httpProxyURL)
		if err != nil {
			log.Fatal(err)
		}
		httpProxy = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	geminiProxy := gemproxy.NewGeminiHandler(httpProxy)
	openaiProxy := gemproxy.NewOpenHandler(httpProxy)

	router := gin.Default()

	api := router.Group("/").Use(func(ctx *gin.Context) {
		slog.Info("request begin", "ip", ctx.ClientIP(), "path", ctx.Request.URL.Path)
		ctx.Next()
	})
	api.Any("/*path", func(ctx *gin.Context) {
		if gemproxy.IsOpenAI(ctx.Request.URL.Path) {
			openaiProxy.ServeHTTP(ctx.Writer, ctx.Request)
		} else {
			geminiProxy.ServeHTTP(ctx.Writer, ctx.Request)
		}
	})

	if err := router.Run(*addr); err != nil {
		log.Fatal(err)
	}
}
