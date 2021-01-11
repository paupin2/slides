package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

const (
	extension = ".txt"
)

var (
	developmentMode bool
	templateCache   *template.Template
	staticCache     map[string]StaticFile
)

func getFilename(name string) string {
	return path.Join(config.Path.Data, name+extension)
}

func sendTemplate(w http.ResponseWriter, r *http.Request, s int, name string, data D) {
	var buf bytes.Buffer
	if data == nil {
		data = D{}
	}
	data["baseurl"] = config.BaseURL

	if err := templateCache.ExecuteTemplate(&buf, name, data); err != nil {
		log.Error().Err(err).Str("template", name).Msg("template error")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	w.WriteHeader(s)
	buf.WriteTo(w)
}

func sendPayload(w http.ResponseWriter, r *http.Request, s int, b []byte) {
	w.WriteHeader(s)
	if b != nil {
		w.Write(b)
	}
}

func sendJSON(w http.ResponseWriter, r *http.Request, s int, data interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		log.Error().Err(err).Msg("encoding error")
		sendPayload(w, r, http.StatusInternalServerError, nil)
	} else {
		sendPayload(w, r, s, buf.Bytes())
	}
}

func sendMessage(w http.ResponseWriter, r *http.Request, s int, m Message) {
	b, err := json.Marshal(m)
	if err != nil {
		sendPayload(w, r, http.StatusInternalServerError, nil)
		log.Info().Str("path", r.URL.Path).Int("status", s).Err(err).Msg("request")
		return
	}
	sendPayload(w, r, s, b)
}

func load() {
	isDir := func(s string) bool {
		fi, err := os.Stat(s)
		return err == nil && fi.IsDir()
	}

	if !isDir(config.Path.Data) {
		log.Fatal().Str("dir", config.Path.Data).Msg("bad data dir")
	}
	if !isDir(config.Path.Static) {
		log.Fatal().Str("dir", config.Path.Static).Msg("bad static dir")
	}

	templateCache = cacheTemplates()
	staticCache = cacheStatic()
}

// Config is used in config.yaml
type Config struct {
	Address string
	BaseURL string
	Path    struct {
		Static  string
		Data    string
		Log     string
		logfile io.ReadCloser
	}
}

var config Config

func loadConfig(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", filename, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		fmt.Fprintf(os.Stderr, "parsing %s: %v\n", filename, err)
	}

	if lfn := config.Path.Log; lfn != "" {
		wc, err := os.OpenFile(lfn, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parsing %s: %v\n", lfn, err)
		}
		log.Logger = log.Output(wc)
		config.Path.logfile = wc

	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05"})
	}

	for _, d := range []string{config.Path.Data, config.Path.Static} {
		if s, err := os.Stat(d); err != nil {
			fmt.Fprintf(os.Stderr, "bad dir %s: %v\n", d, err)
		} else if !s.IsDir() {
			fmt.Fprintf(os.Stderr, "not dir %s\n", d)
		}
	}
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.BoolVar(&developmentMode, "dev", false, "use development mode")
	flag.Parse()
	loadConfig(*configPath)
	load()
	fmt.Printf("%+v\n", config)

	srv := newServer()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if developmentMode {
			templateCache = cacheTemplates()
			staticCache = cacheStatic()
		}

		if serveStatic(w, r) || srv.Handle(w, r) {
			return
		}

		sendTemplate(w, r, http.StatusNotFound, "404.html", nil)
	})

	log.Info().Str("address", config.BaseURL).Msg("serving")
	err := http.ListenAndServe(config.Address, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("serving")
	}
}
