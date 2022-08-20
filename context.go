package main

import (
	"io/fs"
	"net/http"
)

type Context struct {
	Config  Config
	Request *http.Request

	resp       http.ResponseWriter
	dir        fs.FS
	statusCode int
	sentBytes  int64
}

func NewContext(c Config, w http.ResponseWriter, r *http.Request, f fs.FS) *Context {
	return &Context{
		Config:  c,
		Request: r,

		resp:       w,
		dir:        f,
		statusCode: http.StatusOK,
	}
}

func (c *Context) Header() http.Header {
	return c.resp.Header()
}

func (c *Context) Write(b []byte) (int, error) {
	n, err := c.resp.Write(b)
	c.sentBytes += int64(n)
	return n, err
}

func (c *Context) WriteHeader(statusCode int) {
	c.statusCode = statusCode
	c.resp.WriteHeader(statusCode)
}

func (c *Context) ContentType(mimeType string) {
	if mimeType != "" {
		c.Header().Set("Content-Type", mimeType)
	}
}

func (c *Context) Open(name string) (fs.File, error) {
	return c.dir.Open(name)
}
