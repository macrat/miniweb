package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"io"
	"io/fs"

	chroma "github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	markdownTemplate = template.Must(template.ParseFS(templateFiles, "_template/markdown.html"))
)

type MarkdownConfig struct {
	Script     []string     `yaml:"script"`
	ScriptCode template.JS  `yaml:"script_code"`
	Style      []string     `yaml:"style"`
	StyleCode  template.CSS `yaml:"style_code"`
}

type MarkdownType struct{}

func (mt MarkdownType) Serve(c *Context, f fs.File, i fs.FileInfo) error {
	c.ContentType("text/html")

	gfm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(chroma.WithLineNumbers(true)),
			),
			emoji.Emoji,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	raw, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err = gfm.Convert(raw, &buf); err != nil {
		return err
	}

	data := struct {
		Body   template.HTML
		Config MarkdownConfig
	}{
		Body:   template.HTML(buf.String()),
		Config: c.Config.Markdown,
	}

	buf.Reset()
	if err = markdownTemplate.Execute(&buf, data); err != nil {
		return err
	}

	c.ServeContent(i.Name(), i.ModTime(), int64(buf.Len()), &buf)

	return nil
}
