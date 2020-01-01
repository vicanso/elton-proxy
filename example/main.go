package main

import (
	"net/url"

	"github.com/vicanso/elton"

	proxy "github.com/vicanso/elton-proxy"
)

func main() {
	e := elton.New()

	target, _ := url.Parse("https://www.baidu.com")

	e.GET("/*url", proxy.New(proxy.Config{
		// proxy done will call this function
		Done: func(c *elton.Context) {

		},
		// http request url rewrite
		Rewrites: []string{
			"/api/*:/$1",
		},
		Target: target,
		// change the request host
		Host: "www.baidu.com",
	}))

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
