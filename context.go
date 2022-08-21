package main

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"time"
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

func (c *Context) ServeContent(name string, modtime time.Time, size int64, content io.Reader) {
	http.ServeContent(c, c.Request, name, modtime, dummySeeker{content, size})
}

func (c *Context) Redirect(path string) {
	http.Redirect(c, c.Request, path, http.StatusMovedPermanently)
}

func (c *Context) ContentType(mimeType string) {
	if mimeType != "" {
		c.Header().Set("Content-Type", mimeType)
	}
}

func (c *Context) Open(name string) (fs.File, error) {
	return c.dir.Open(name)
}

type dummySeeker struct {
	Reader io.Reader
	Size   int64
}

func (d dummySeeker) Read(p []byte) (int, error) {
	return d.Reader.Read(p)
}

func (d dummySeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart && offset == 0 {
		return 0, nil
	} else if whence == io.SeekEnd && offset == 0 {
		return d.Size, nil
	} else {
		return 0, errors.New("not supported operation")
	}
}
