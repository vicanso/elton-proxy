package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
	"gopkg.in/h2non/gock.v1"
)

func TestNoTargetPanic(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.Equal(r.(error), errNoTargetFunction)
	}()
	New(Config{})
}

func TestInvalidRewrite(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.NotNil(r.(error))
	}()
	target, _ := url.Parse("https://github.com")
	New(Config{
		Target: target,
		Rewrites: []string{
			"/(d/:a",
		},
	})
}

func TestGenerateRewrites(t *testing.T) {
	assert := assert.New(t)
	regs, err := generateRewrites([]string{
		"a:b:c",
	})
	assert.Nil(err)
	assert.Equal(len(regs), 0, "rewrite regexp map should be 0")

	regs, err = generateRewrites([]string{
		"/(d/:a",
	})
	assert.NotNil(err)
	assert.Equal(len(regs), 0, "regexp map should be 0 when error occur")
}

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
		c := elton.NewContext(resp, req)
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
		callBackDone := false
		config := Config{
			TargetPicker: func(c *elton.Context) (*url.URL, Done, error) {
				return target, func(_ *elton.Context) {
					callBackDone = true
				}, nil
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn(c)
		assert.True(done)
		assert.True(callBackDone)
		assert.Equal(c.StatusCode, http.StatusOK)
	})

	t.Run("target picker error", func(t *testing.T) {
		assert := assert.New(t)
		config := Config{
			TargetPicker: func(c *elton.Context) (*url.URL, Done, error) {
				return nil, nil, errors.New("abcd")
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		err := fn(c)
		assert.Equal(err.Error(), "abcd")
	})

	t.Run("no target", func(t *testing.T) {
		assert := assert.New(t)
		config := Config{
			TargetPicker: func(c *elton.Context) (*url.URL, Done, error) {
				return nil, nil, nil
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		err := fn(c)
		assert.Equal(err.Error(), "category=elton-proxy, message=target can not be nil")
	})

	t.Run("proxy error", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://a")
		config := Config{
			TargetPicker: func(c *elton.Context) (*url.URL, Done, error) {
				return target, nil, nil
			},
			Transport: &http.Transport{},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
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
			Done: func(_ *elton.Context) {
				done = true
			},
		}
		fn := New(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		fn(c)
		assert.True(done)
	})
}

func BenchmarkProxy(b *testing.B) {
	b.ReportAllocs()
	target, _ := url.Parse("https://www.baidu.com")
	config := Config{
		Target: target,
		Host:   "www.baidu.com",
		Rewrites: []string{
			"/api/*:/$1",
		},
	}
	fn := New(config)
	defer gock.Off() // Flush pending mocks after test execution

	gock.New("https://www.baidu.com").
		Get("/").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		fn(c)
	}
}

// https://stackoverflow.com/questions/50120427/fail-unit-tests-if-coverage-is-below-certain-percentage
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	rc := m.Run()

	// rc 0 means we've passed,
	// and CoverMode will be non empty if run with -cover
	if rc == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < 0.9 {
			fmt.Println("Tests passed but coverage failed at", c)
			rc = -1
		}
	}
	os.Exit(rc)
}
