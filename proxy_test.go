package proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/cod"
)

func TestProxy(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://github.com")
		config := Config{
			Target:    target,
			Host:      "github.com",
			Transport: &http.Transport{},
			Rewrites: []string{
				"/api/*:/$1",
			},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/api/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		originalPath := req.URL.Path
		originalHost := req.Host
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn(c)
		assert.Equal(c.GetHeader("Content-Encoding"), "gzip")
		assert.Equal(c.Request.URL.Path, originalPath)
		assert.Equal(req.Host, originalHost)
		assert.True(done)
		assert.Equal(c.StatusCode, http.StatusOK)
	})

	t.Run("target picker", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://www.baidu.com")
		config := Config{
			TargetPicker: func(c *cod.Context) (*url.URL, Done, error) {
				return target, nil, nil
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn(c)
		assert.True(done)
		assert.Equal(c.StatusCode, http.StatusOK)
	})

	t.Run("target picker error", func(t *testing.T) {
		assert := assert.New(t)
		config := Config{
			TargetPicker: func(c *cod.Context) (*url.URL, Done, error) {
				return nil, nil, errors.New("abcd")
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		err := fn(c)
		assert.Equal(err.Error(), "abcd")
	})

	t.Run("no target", func(t *testing.T) {
		assert := assert.New(t)
		config := Config{
			TargetPicker: func(c *cod.Context) (*url.URL, Done, error) {
				return nil, nil, nil
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		err := fn(c)
		assert.Equal(err.Error(), "category=cod-proxy, message=target can not be nil")
	})

	t.Run("proxy error", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://a")
		config := Config{
			TargetPicker: func(c *cod.Context) (*url.URL, Done, error) {
				return target, nil, nil
			},
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.NotNil(err)
	})

	t.Run("proxy done", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://www.baidu.com")
		done := false
		config := Config{
			Target:    target,
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
			Done: func(_ *cod.Context) {
				done = true
			},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		fn(c)
		assert.True(done)
	})
}
