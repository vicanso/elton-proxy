# elton-proxy

[![Build Status](https://img.shields.io/travis/vicanso/elton-proxy.svg?label=linux+build)](https://travis-ci.org/vicanso/elton-proxy)

Proxy middleware for elton, it can proxy http request to other host.

```go
package main

import (
	"net/url"

	"github.com/vicanso/elton"

	proxy "github.com/vicanso/elton-proxy"
)

func main() {
	d := elton.New()

	target, _ := url.Parse("https://www.baidu.com")

	d.GET("/*url", proxy.New(proxy.Config{
		Target: target,
		Host:   "www.baidu.com",
	}))

	d.ListenAndServe(":7001")
}
```