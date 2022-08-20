package main

import (
	"io"
	"io/fs"
)

type FileType interface {
	Serve(c *Context, f fs.File) error
}

type AsIsFile string

func (af AsIsFile) Serve(c *Context, f fs.File) error {
	c.ContentType(string(af))

	_, err := io.Copy(c, f)
	return err
}

var (
	StandardFileTypes = map[string]FileType{
		".3g2":    AsIsFile("video/3gpp2"),
		".3gp2":   AsIsFile("video/3gpp2"),
		".mkv":    AsIsFile("video/x-matroska"),
		".3gp":    AsIsFile("video/3gpp"),
		".3gpp":   AsIsFile("video/3gpp"),
		".7z":     AsIsFile("application/x-7z-compressed"),
		".aac":    AsIsFile("audio/aac"),
		".avi":    AsIsFile("video/x-msvideo"),
		".bash":   AsIsFile("text/plain"),
		".bmp":    AsIsFile("image/bmp"),
		".bz":     AsIsFile("application/x-bzip"),
		".bz2":    AsIsFile("application/x-bzip2"),
		".csh":    AsIsFile("text/plain"),
		".css":    AsIsFile("text/css"),
		".csv":    AsIsFile("text/csv"),
		".epub":   AsIsFile("application/epub+zip"),
		".gif":    AsIsFile("image/gif"),
		".gz":     AsIsFile("application/gzip"),
		".htm":    AsIsFile("text/html"),
		".html":   AsIsFile("text/html"),
		".ico":    AsIsFile("image/vnd.microsoft.icon"),
		".ics":    AsIsFile("text/calendar"),
		".jar":    AsIsFile("application/java-archive"),
		".jpeg":   AsIsFile("image/jpeg"),
		".jpg":    AsIsFile("image/jpeg"),
		".js":     AsIsFile("text/javascript"),
		".json":   AsIsFile("application/json"),
		".jsonl":  AsIsFile("application/json"),
		".jsonld": AsIsFile("application/ld+json"),
		".mid":    AsIsFile("audio/midi"),
		".midi":   AsIsFile("audio/midi"),
		".mjs":    AsIsFile("text/javascript"),
		".mp3":    AsIsFile("audio/mpeg"),
		".mpeg":   AsIsFile("video/mpeg"),
		".oga":    AsIsFile("audio/ogg"),
		".ogv":    AsIsFile("video/ogg"),
		".opus":   AsIsFile("audio/opus"),
		".otf":    AsIsFile("font/otf"),
		".pdf":    AsIsFile("application/pdf"),
		".png":    AsIsFile("image/png"),
		".sh":     AsIsFile("text/plain"),
		".svg":    AsIsFile("image/svg+xml"),
		".tar":    AsIsFile("application/x-tar"),
		".tif":    AsIsFile("image/tiff"),
		".tiff":   AsIsFile("image/tiff"),
		".ts":     AsIsFile("video/mp2t"),
		".ttf":    AsIsFile("font/ttf"),
		".txt":    AsIsFile("text/plain"),
		".wav":    AsIsFile("audio/wav"),
		".weba":   AsIsFile("audio/webm"),
		".webm":   AsIsFile("video/webm"),
		".webp":   AsIsFile("image/webm"),
		".woff":   AsIsFile("font/woff"),
		".woff2":  AsIsFile("font/woff2"),
		".xhtml":  AsIsFile("application/xhtml+xml"),
		".xml":    AsIsFile("text/xml"),
		".zip":    AsIsFile("application/zip"),
		".md":     MarkdownType{},
		"":        AsIsFile(""),
	}
)
