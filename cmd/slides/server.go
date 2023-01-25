package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/paupin2/slides/cmd/slides/pkg/decks"
	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/paupin2/slides/cmd/slides/pkg/songs"
	"github.com/paupin2/slides/pkg/data"
	"github.com/rs/zerolog/log"
)

type (
	handler func(*inout.Request) *inout.Reply

	Server struct {
		lock    sync.RWMutex
		routes  map[string]map[string]handler
		screens map[string][]*Screen
		content map[string]string
	}
)

func (srv *Server) get(title string) string {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	return srv.content[title]
}

func (srv *Server) set(title, content string) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.content == nil {
		srv.content = make(map[string]string)
	}

	srv.content[title] = content
}

func (srv *Server) HandleShow(req *inout.Request) *inout.Reply {
	var data struct {
		Title string `json:"title"`
		Show  string `json:"show"`
	}

	if err := req.Read(&data); err != nil || data.Title == "" {
		return inout.Error(http.StatusBadRequest, "bad deck")
	}

	// set the content, send it to all screens
	srv.set(data.Title, data.Show)
	srv.Broadcast(data.Title, Content{data.Show})

	return inout.OK()
}

var version = ""

func handleGetVersion(req *inout.Request) *inout.Reply {
	return inout.JSON(version)
}

func newServer() *Server {
	srv := &Server{
		screens: map[string][]*Screen{},
		routes: map[string]map[string]handler{
			http.MethodGet: {
				"/version": handleGetVersion,

				"/song":  songs.HandleGet,
				"/songs": songs.HandleList,

				"/deck":  decks.HandleGet,
				"/decks": decks.HandleList,
			},
			http.MethodPost: {
				"/song": songs.HandlePost,
			},
			http.MethodPut: {
				"/song": songs.HandlePut,
				"/deck": decks.HandlePut,
			},
			http.MethodDelete: {
				"/deck": decks.HandleDelete,
				"/song": songs.HandleDelete,
			},
		},
	}
	srv.routes[http.MethodPost]["/show"] = srv.HandleShow
	srv.routes[http.MethodGet]["/screen"] = srv.HandleScreen

	return srv
}

// Route returns true if the expected route matches. Each value in `expected`
// can be either a string or a *Deck. In this last case, it must correspond
// to the title of a valid deck, and the pointer is set to the deck.
func (s *Server) Route(r *http.Request, method string, expected ...any) bool {
	if method != r.Method {
		return false
	}

	actual := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(actual) != len(expected) {
		return false
	}

	for idx, exp := range expected {
		switch exp := exp.(type) {
		case string:
			if exp != "*" && exp != actual[idx] {
				return false
			}

		case *data.Deck:
			deck, found := data.LoadDeck(actual[idx])
			if !found {
				return false
			}
			*exp = deck

		default:
			log.Error().Str("type", fmt.Sprintf("%T", exp)).Msg("bad route type")
			return false
		}
	}
	return true
}

// Handle requests to the server
func (s *Server) Handle(req *inout.Request) *inout.Reply {
	path := req.Path()
	if handler := s.routes[req.Method()][path]; handler != nil {
		return handler(req)
	}
	return nil
}
