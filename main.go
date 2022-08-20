package main

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CAFxX/httpcompression"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var (
	ErrNotFound = fs.ErrNotExist
)

//go:embed _template/*
var templateFiles embed.FS

var (
	errorPageTemplate = template.Must(template.ParseFS(templateFiles, "_template/error.html"))
)

type ErrorData struct {
	Message string
	Code    int
}

type MarkdownConfig struct {
	Script     []string `yaml:"script"`
	ScriptCode string   `yaml:"script_code"`
	Style      []string `yaml:"style"`
	StyleCode  string   `yaml:"style_code"`
}

type Config struct {
	Index     []string       `yaml:"index"`
	AutoIndex bool           `yaml:"autoindex"`
	Markdown  MarkdownConfig `yaml:"markdown"`
}

func (c *Config) SetDefault() {
	if len(c.Index) == 0 {
		c.Index = []string{"index.html", "index.md", "index.txt"}
	}
}

type MiniWeb struct {
	Logger    zerolog.Logger
	Dir       fs.FS
	FileTypes map[string]FileType
	Config    Config
}

func (mw MiniWeb) ErrorPage(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(code)
	errorPageTemplate.Execute(w, ErrorData{
		Message: message,
		Code:    code,
	})
}

func (mw MiniWeb) NotFound(w http.ResponseWriter) {
	mw.ErrorPage(w, "not found", http.StatusNotFound)
}

func (mw MiniWeb) Forbidden(w http.ResponseWriter) {
	mw.ErrorPage(w, "forbidden", http.StatusForbidden)
}

func (mw MiniWeb) MethodNotAllowed(w http.ResponseWriter) {
	mw.ErrorPage(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (mw MiniWeb) InternalServerError(w http.ResponseWriter) {
	mw.ErrorPage(w, "internal server error", http.StatusInternalServerError)
}

func (mw MiniWeb) handleError(c *Context, err error) {
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid):
		mw.NotFound(c)
	case errors.Is(err, fs.ErrPermission):
		mw.Forbidden(c)
	default:
		mw.Logger.Warn().Err(err).Msg("internal server error")
		mw.InternalServerError(c)
	}
}

func (mw MiniWeb) serveFile(c *Context, file fs.File, info fs.FileInfo) {
	s, ok := mw.FileTypes[filepath.Ext(info.Name())]
	if !ok {
		s = mw.FileTypes[""]
	}

	if err := s.Serve(c, file); err != nil {
		mw.handleError(c, err)
	}
}

func (mw MiniWeb) serveDir(c *Context, dir fs.File, info fs.FileInfo) {
	for _, name := range mw.Config.Index {
		f, err := mw.Dir.Open(filepath.Join(info.Name(), name))
		if err == nil {
			mw.serve(c, f)
			return
		}
	}

	if !mw.Config.AutoIndex {
		mw.NotFound(c)
		return
	}

	// TODO: implement directory listing
	mw.NotFound(c)
}

func (mw MiniWeb) serve(c *Context, f fs.File) {
	info, err := f.Stat()
	if err != nil {
		mw.handleError(c, err)
		return
	}

	if info.IsDir() {
		mw.serveDir(c, f, info)
	} else {
		mw.serveFile(c, f, info)
	}
}

func (mw MiniWeb) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := NewContext(mw.Config, w, r, mw.Dir)

	if r.Method != http.MethodGet {
		mw.MethodNotAllowed(c)
	} else if filepath.Base(r.URL.Path)[0] == '.' {
		mw.NotFound(c)
	} else {
		if f, err := mw.Dir.Open(filepath.Clean("." + r.URL.Path)); err != nil {
			mw.handleError(c, err)
		} else {
			mw.serve(c, f)
		}
	}

	mw.Logger.Info().
		Timestamp().
		Str("remote", r.RemoteAddr).
		Str("method", r.Method).
		Str("url", r.URL.Path).
		Str("referer", r.Referer()).
		Str("ua", r.UserAgent()).
		Int("status", c.statusCode).
		Int64("sent", c.sentBytes).
		Send()
}

func main() {
	mw := MiniWeb{
		Logger:    zerolog.New(os.Stdout),
		Dir:       os.DirFS("./example-site"),
		FileTypes: StandardFileTypes,
	}

	conf, err := os.ReadFile("./example-site/.miniweb.yaml")
	if err != nil {
		mw.Logger.Fatal().Err(err).Msg("failed to read config")
	}
	if err = yaml.Unmarshal(conf, &mw.Config); err != nil {
		mw.Logger.Fatal().Err(err).Msg("failed to parse config")
	}
	mw.Config.SetDefault()

	mw.Logger.Info().
		Timestamp().
		Str("address", "0.0.0.0:8000").
		Msg("start MiniWeb server")

	compress, _ := httpcompression.DefaultAdapter()

	http.ListenAndServe(":8000", compress(mw))
}
