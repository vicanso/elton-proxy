// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/vicanso/hes"

	"github.com/vicanso/cod"
)

const (
	// ErrCategory proxy error category
	ErrCategory = "cod-proxy"
)

var (
	// errTargetIsNil target is nil
	errTargetIsNil = &hes.Error{
		Exception:  true,
		Message:    "target can not be nil",
		StatusCode: http.StatusBadRequest,
		Category:   ErrCategory,
	}
)

type (
	// Done http proxy done callback
	Done func(*cod.Context)
	// TargetPicker target picker function
	TargetPicker func(c *cod.Context) (*url.URL, Done, error)
	// Config proxy config
	Config struct {
		// Done proxy done callback
		Done         Done
		Target       *url.URL
		Rewrites     []string
		Host         string
		Transport    *http.Transport
		TargetPicker TargetPicker
		Skipper      cod.Skipper
	}
)

func captureTokens(pattern *regexp.Regexp, input string) *strings.Replacer {
	groups := pattern.FindAllStringSubmatch(input, -1)
	if groups == nil {
		return nil
	}
	values := groups[0][1:]
	replace := make([]string, 2*len(values))
	for i, v := range values {
		j := 2 * i
		replace[j] = "$" + strconv.Itoa(i+1)
		replace[j+1] = v
	}
	return strings.NewReplacer(replace...)
}

func rewrite(rewriteRegexp map[*regexp.Regexp]string, req *http.Request) {
	for k, v := range rewriteRegexp {
		replacer := captureTokens(k, req.URL.Path)
		if replacer != nil {
			req.URL.Path = replacer.Replace(v)
		}
	}
}

// generateRewrites generate rewrites
func generateRewrites(rewrites []string) (m map[*regexp.Regexp]string, err error) {
	if len(rewrites) == 0 {
		return
	}
	m = make(map[*regexp.Regexp]string)

	for _, value := range rewrites {
		arr := strings.Split(value, ":")
		if len(arr) != 2 {
			continue
		}
		k := arr[0]
		v := arr[1]
		k = strings.Replace(k, "*", "(\\S*)", -1)
		reg, e := regexp.Compile(k)
		if e != nil {
			err = e
			break
		}
		m[reg] = v
	}
	return
}

// New create a proxy middleware
func New(config Config) cod.Handler {
	if config.Target == nil && config.TargetPicker == nil {
		panic("require target or targer picker")
	}
	regs, err := generateRewrites(config.Rewrites)
	if err != nil {
		panic(err)
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = cod.DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		target := config.Target
		var done Done
		if target == nil {
			target, done, err = config.TargetPicker(c)
			if err != nil {
				return
			}
		}
		// 如果无target，则抛错
		if target == nil {
			err = errTargetIsNil
			return
		}
		p := httputil.NewSingleHostReverseProxy(target)
		if config.Transport != nil {
			p.Transport = config.Transport
		}
		req := c.Request
		var originalPath, originalHost string
		if regs != nil {
			originalPath = req.URL.Path
			rewrite(regs, req)
		}
		if config.Host != "" {
			originalHost = req.Host
			req.Host = config.Host
		}
		p.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, e error) {
			he := hes.NewWithError(e)
			he.Category = ErrCategory
			he.Exception = true
			err = he
		}
		p.ServeHTTP(c, req)
		if config.Done != nil {
			config.Done(c)
		}
		if done != nil {
			done(c)
		}
		if err != nil {
			return
		}
		if originalPath != "" {
			req.URL.Path = originalPath
		}
		if originalHost != "" {
			req.Host = originalHost
		}
		return c.Next()
	}
}
