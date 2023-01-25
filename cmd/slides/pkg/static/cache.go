package static

import (
	"bytes"
	"compress/gzip"
	"embed"
	_ "embed"
	"flag"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/rs/zerolog/log"
)

const (
	mainPage = "main.html"
)

var (
	// *.jpg
	//go:embed *.html *.js *.css *.svg *.ico *.png
	static embed.FS

	staticPath  = flag.String("static-path", "", "Reload static assets from this path before each request")
	staticCache = makeCache(loadEmbededFiles())

	knownContentTypes = map[string]string{
		".css":  "text/css; charset=UTF-8",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".ico":  "image/x-icon",
		".svg":  "image/svg+xml",
		".js":   "text/javascript; charset=UTF-8",
		".html": "text/html; charset=UTF-8",
	}
)

type StaticFile struct {
	content    []byte
	compressed []byte
	ctype      string
}

type cachedFile struct {
	path    string
	content []byte
}

func loadEmbededFiles() []cachedFile {
	var files []cachedFile
	var read func(string)
	read = func(dirname string) {
		content, err := static.ReadDir(dirname)
		if err != nil {
			panic(err)
		}
		for _, f := range content {
			filename := f.Name()
			fullName := path.Join(dirname, filename)
			if f.IsDir() {
				read(fullName)
				continue
			}

			buf, err := static.ReadFile(fullName)
			if err != nil {
				panic(err)
			}
			files = append(files, cachedFile{filename, buf})
		}
	}
	read(".")
	return files
}

// this clean version and minification suffixes from filenames,
// making the filename more opaque to the client
var reCleanNames = regexp.MustCompile(`(\.v[0-9.]+)?\.min\b`)

func makeCache(files []cachedFile) map[string]StaticFile {
	static := map[string]StaticFile{}
	for _, f := range files {
		name := reCleanNames.ReplaceAllString(f.path, "")
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}

		ext := path.Ext(name)
		ctype, allowed := knownContentTypes[ext]
		if !allowed {
			// don't show this file
			continue
		}

		file := StaticFile{content: f.content, ctype: ctype}

		// compress content
		var buf bytes.Buffer
		gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if _, err := gz.Write(f.content); err == nil {
			if err = gz.Close(); err == nil {
				file.compressed = buf.Bytes()
			}
		}

		static[name] = file
		if name == "/"+mainPage {
			static["/"] = file
		}
	}
	return static
}

// Refresh the cache from the disk
func Refresh(base string) {
	prefix := path.Clean(base)
	var files []cachedFile
	err := filepath.Walk(base, func(name string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		filename := name
		buf, err := os.ReadFile(filename)
		if err != nil {
			log.Error().Err(err).Str("filename", filename).Msg("reading")
			return nil
		}

		filename = strings.TrimPrefix(filename, prefix)
		files = append(files, cachedFile{filename, buf})
		return nil
	})
	if err != nil {
		log.Error().Err(err).Str("path", base).Msg("loading files from disk")
	}

	staticCache = makeCache(files)
}

func Handle(req *inout.Request) *inout.Reply {
	if *staticPath != "" {
		// development mode
		// reload the cache before every pageview
		Refresh(*staticPath)
	}

	path := req.Path()
	if path == "" {
		path = mainPage
	}

	if file, found := staticCache[path]; found {
		if req.AcceptsGzip() {
			resp := inout.Static(file.ctype, file.compressed)
			resp.Header("Content-Encoding", "gzip")
			return resp
		}

		return inout.Static(file.ctype, file.content)
	}

	return nil
}
