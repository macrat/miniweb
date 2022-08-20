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

type MarkdownType struct{}

func (mt MarkdownType) Serve(c *Context, f fs.File) error {
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
	if err := gfm.Convert(raw, &buf); err != nil {
		return err
	}

	data := struct {
		Body template.HTML
	}{
		Body: template.HTML(buf.String()),
	}

	return markdownTemplate.Execute(c, data)
}
