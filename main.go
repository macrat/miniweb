package main

import (
	"embed"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

//go:embed _template/*
var templateFiles embed.FS

var (
	errorPageTemplate = template.Must(template.ParseFS(templateFiles, "_template/error.html"))
	listingTemplate   = template.Must(template.ParseFS(templateFiles, "_template/listing.html"))
)

type ErrorData struct {
	Message string
	Code    int
}

type Config struct {
	Index     []string       `yaml:"index"`
	AutoIndex bool           `yaml:"autoindex"`
	Markdown  MarkdownConfig `yaml:"markdown"`
}

type ConfigReader struct {
	cache Config
	mtime time.Time
}

func (c *Config) SetDefault() {
	if len(c.Index) == 0 {
		c.Index = []string{"index.html", "index.md", "index.txt"}
	}
}

func NewConfigReader() *ConfigReader {
	reader := &ConfigReader{}
	reader.cache.SetDefault()
	return reader
}

func (cr *ConfigReader) Read(dir fs.FS, logger zerolog.Logger) Config {
	f, err := dir.Open(".miniweb.yaml")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Warn().Err(err).Msg("failed to open config")
		return cr.cache
	}

	info, err := f.Stat()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to check mtime of config")
	} else if info.ModTime().Equal(cr.mtime) {
		return cr.cache
	}
	cr.mtime = info.ModTime()

	raw, err := io.ReadAll(f)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read config")
		return cr.cache
	}

	if err = yaml.Unmarshal(raw, &cr.cache); err != nil {
		logger.Error().Err(err).Msg("failed to parse config")
		cr.cache = Config{}
	}

	cr.cache.SetDefault()
	return cr.cache
}

type MiniWeb struct {
	Logger       zerolog.Logger
	Dir          fs.FS
	FileTypes    map[string]FileType
	ConfigReader *ConfigReader
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
		c.ServeContent(info.Name(), info.ModTime(), info.Size(), file)
		return
	}

	if err := s.Serve(c, file, info); err != nil {
		mw.handleError(c, err)
	}
}

func (mw MiniWeb) serveDir(c *Context, dir fs.File, info fs.FileInfo) {
	if !strings.HasSuffix(c.Request.URL.Path, "/") {
		c.Redirect(c.Request.URL.Path + "/")
		return
	}

	for _, name := range mw.Config().Index {
		f, err := mw.Dir.Open(filepath.Join("."+c.Request.URL.Path, name))
		if err == nil {
			mw.serve(c, f)
			return
		}
	}

	if !mw.Config().AutoIndex {
		mw.Forbidden(c)
		return
	}

	files, err := fs.ReadDir(mw.Dir, filepath.Clean("."+c.Request.URL.Path))
	if err != nil {
		mw.handleError(c, err)
		return
	}

	data := struct {
		Path  string
		Files []fs.DirEntry
	}{
		Path: c.Request.URL.Path,
	}
	for _, f := range files {
		if f.Name()[0] != '.' {
			data.Files = append(data.Files, f)
		}
	}

	listingTemplate.Execute(c, data)
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
	c := NewContext(mw.Config(), w, r, mw.Dir)

	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	} else if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		mw.MethodNotAllowed(c)
	} else if filepath.Base(r.URL.Path)[0] == '.' {
		mw.NotFound(c)
	} else {
		for _, p := range filepath.SplitList(r.URL.Path) {
			if p[0] == '.' {
				mw.NotFound(c)
				return
			}
		}

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
		Str("useragent", r.UserAgent()).
		Int("status", c.statusCode).
		Int64("sent", c.sentBytes).
		Send()
}

func (mw MiniWeb) Config() Config {
	return mw.ConfigReader.Read(mw.Dir, mw.Logger)
}

func main() {
	logger := zerolog.New(os.Stdout)

	dir, err := os.Getwd()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to get the current directory")
	} else if len(os.Args) >= 2 {
		dir = os.Args[1]
	}

	mw := MiniWeb{
		Logger:       logger,
		Dir:          os.DirFS(dir),
		ConfigReader: NewConfigReader(),
	}

	mw.FileTypes = map[string]FileType{
		".md": MarkdownType{},
	}

	mw.Logger.Info().
		Timestamp().
		Str("address", "0.0.0.0:8000").
		Msg("start MiniWeb server")

	compress, _ := httpcompression.DefaultAdapter()

	http.ListenAndServe(":8000", compress(mw))
}
