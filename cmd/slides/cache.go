package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type StaticFile struct {
	content []byte
	ctype   string
}

var knownContentTypes = map[string]string{
	".css": "text/css; charset=utf-8",
	".jpg": "image/jpeg",
	".ico": "image/x-icon",
	".js":  "text/javascript; charset=utf-8",
}

func serveStatic(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		if file, found := staticCache[r.URL.Path]; found {
			w.Header().Add("Content-Length", fmt.Sprint(len(file.content)))
			w.Header().Add("Content-Type", file.ctype)
			w.WriteHeader(http.StatusOK)
			w.Write(file.content)
			return true
		}
	}
	return false
}

func cacheStatic() map[string]StaticFile {
	base := map[string]StaticFile{}
	prefix := path.Clean(config.Path.Static)
	err := filepath.Walk(config.Path.Static, func(filename string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		var file StaticFile
		switch ext := path.Ext(filename); ext {
		case ".css", ".ico", ".jpg", ".js":
			if ctype, found := knownContentTypes[ext]; found {
				file.ctype = ctype
			} else {
				file.ctype = mime.TypeByExtension(ext)
			}

		default:
			return nil
		}

		if buf, err := ioutil.ReadFile(filename); err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("reading")
		} else {
			file.content = buf
		}

		name := strings.TrimPrefix(filename, prefix)
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		base[name] = file
		// log.Debug().Str("filename", filename).Str("name", name).Msg("cached")
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("parting template")
	}
	return base
}

func cacheTemplates() *template.Template {
	var base *template.Template
	err := filepath.Walk(config.Path.Static, func(filename string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if ext := path.Ext(filename); ext != ".tmpl" {
			return nil
		}
		name := path.Base(strings.TrimSuffix(filename, ".tmpl"))

		buf, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("reading")
		}

		var tmpl *template.Template
		if base == nil {
			base = template.New(name)
		}
		if name == base.Name() {
			tmpl = base
		} else {
			tmpl = base.New(name)
		}
		_, err = tmpl.Parse(string(buf))
		if err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("parsing")
		}
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("parting template")
	}
	return base
}
