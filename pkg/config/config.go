package config

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
)

// Struct contains the data structure read from config.yaml
type Struct struct {
	Address string
	BaseURL string
	Path    struct {
		Db      string
		Log     string
		logfile io.ReadCloser
	}
	PlanningCenter struct {
		AppID  string
		Secret string
	}
}

var Config Struct

func abort(format string, a ...interface{}) {
	msg := fmt.Sprintf("aborting: "+format, a...)
	os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func Load() {
	if configPath == nil || *configPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(*configPath)
	if err != nil {
		abort("reading %s: %v\n", *configPath, err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&Config); err != nil {
		abort("parsing %s: %v\n", *configPath, err)
	}

	if lfn := Config.Path.Log; lfn != "" {
		wc, err := os.OpenFile(lfn, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
		if err != nil {
			abort("parsing %s: %v\n", lfn, err)
		}
		log.Logger = log.Output(wc)
		Config.Path.logfile = wc

	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05"})
	}
	log.Info().Str("path", *configPath).Msg("loaded config")
}
