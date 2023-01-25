package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/paupin2/slides/cmd/slides/pkg/static"
	"github.com/paupin2/slides/pkg/config"
	"github.com/paupin2/slides/pkg/data"
	"github.com/paupin2/slides/pkg/planningcenter"
	"github.com/rs/zerolog/log"
)

var (
	loadDecksPath = flag.String("load-decks", "", "Path where we should load decks from")
)

func runServer() {
	srv := newServer()
	notFound := inout.Status(http.StatusNotFound)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		req := inout.NewRequest(w, r)
		reply := static.Handle(req)
		if reply == nil {
			reply = srv.Handle(req)
		}
		if reply == nil {
			reply = notFound
		}
		req.Send(reply)
	})

	log.Info().Str("address", config.Config.BaseURL).Msg("serving")
	err := http.ListenAndServe(config.Config.Address, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("serving")
	}
}

func updateSongs() {
	if err := planningcenter.Update(); err != nil {
		log.Fatal().Err(err).Msg("update failed")
	}
	log.Info().Msg("update ok")
}

func loadDecks() {
	data.ImportDecks(*loadDecksPath)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [-option] <action>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  action is one of:\n")
	fmt.Fprintf(os.Stderr, "  \trun: run the server\n")
	fmt.Fprintf(os.Stderr, "  \tupdate: update the songs from planning center\n")
	fmt.Fprintf(os.Stderr, "  \tload: load the decks from files into the database\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		usage()
	}

	var action func()
	switch args[0] {
	case "run":
		action = runServer
	case "update":
		action = updateSongs
	case "load":
		action = loadDecks
	default:
		usage()
	}

	config.Load()
	data.Connect()
	action()
}
