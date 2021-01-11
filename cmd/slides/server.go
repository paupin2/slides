package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type (
	D map[string]interface{}

	Server struct {
		lock    sync.RWMutex
		decks   DeckList
		screens map[string][]Screen
	}
)

func newServer() *Server {
	var srv Server
	srv.decks = NewDeckList(&srv)
	srv.screens = map[string][]Screen{}

	err := filepath.Walk(config.Path.Data, func(filename string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if path.Ext(filename) != extension {
			return nil
		}
		return srv.decks.Load(filename)
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to load")
	}

	return &srv
}

// NewDeck creates a new deck on the server
func (s *Server) NewDeck(title string) Deck {
	if !allowedTitle(title) {
		log.Error().Str("title", title).Msg("tried to create deck with bad title")
		return Deck{}
	}

	if existing := s.decks.Get(title); !existing.Empty() {
		// already exists
		return existing
	}

	return s.decks.Add(title, "")
}

// AddClient adds a client to the deck
func (s *Server) AddClient(d Deck, scr Screen) {
	s.lock.Lock()
	defer s.lock.Unlock()
	scr.OnShutdown(func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		scrs := s.screens[d.Title]
		for i, cl := range scrs {
			if cl == scr {
				s.screens[d.Title] = append(scrs[:i], scrs[i+1:]...)
				return
			}
		}
		return
	})
	s.screens[d.Title] = append(s.screens[d.Title], scr)
}

// GetClients returns a list of active clients
func (s *Server) GetClients(d Deck) []Screen {
	s.lock.Lock()
	defer s.lock.Unlock()
	screens := s.screens[d.Title]
	ls := make([]Screen, len(screens))
	for i, s := range screens {
		ls[i] = s
	}
	return ls
}

func handleStatic(w http.ResponseWriter, r *http.Request, filename string) {
	http.ServeFile(w, r, filename)
}

func deckAliases(title string) string {
	const days = time.Hour * 24
	const fmt = "2006-01-02"
	now := time.Now()
	weekday := now.Weekday()
	daysPastLastSunday := time.Duration(-weekday)
	lastSunday := now.Add(daysPastLastSunday * days)
	formatWeekday := func(w time.Weekday) string {
		if weekday >= w {
			// next week
			w += 7
		}
		day := lastSunday.Add(time.Duration(w) * days)
		return day.Format(fmt)
	}

	switch strings.ToLower(title) {
	case "last": // last Sunday
		return lastSunday.Format(fmt)
	case "current", "sunday", "sun": // next Sunday (or today)
		return lastSunday.Add(7 * days).Format(fmt)
	case "monday", "mon":
		return formatWeekday(time.Monday)
	case "tuesday", "tue":
		return formatWeekday(time.Tuesday)
	case "wednesday", "wed":
		return formatWeekday(time.Wednesday)
	case "thursday", "thu":
		return formatWeekday(time.Thursday)
	case "friday", "fri":
		return formatWeekday(time.Friday)
	case "saturday", "sat":
		return formatWeekday(time.Saturday)
	}

	// no special meaning, use title "as is"
	return title
}

// Route returns true if the expected route matches. Each value in `expected`
// can be either a string or a *Deck. In this last case, it must correspond
// to the title of a valid deck, and the pointer is set to the deck.
func (s *Server) Route(r *http.Request, method string, expected ...interface{}) bool {
	if method != r.Method {
		return false
	}

	actual := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(actual) != len(expected) {
		return false
	}

	for idx, exp := range expected {
		switch exp.(type) {
		case string:
			if exp.(string) != actual[idx] {
				return false
			}

		case *Deck:
			title := deckAliases(actual[idx])
			found := s.decks.Get(title)
			if found.Empty() {
				return false
			}
			deckPtr := exp.(*Deck)
			*deckPtr = found

		default:
			log.Error().Str("type", fmt.Sprintf("%T", exp)).Msg("bad route type")
			return false
		}
	}
	return true
}

// Handle requests to the server
func (s *Server) Handle(w http.ResponseWriter, r *http.Request) bool {
	var deck Deck
	if s.Route(r, http.MethodGet, "") {
		sendTemplate(w, r, http.StatusOK, "main.html", D{
			"decks":      s.decks.Titles(),
			"nextSunday": deckAliases("sunday"),
		})

	} else if s.Route(r, http.MethodPost, "decks", "new") {
		if err := r.ParseForm(); err != nil {
			sendTemplate(w, r, http.StatusOK, "error.html", D{
				"error": "Bad form",
			})
			return true
		}

		title := r.Form.Get("title")
		newDeck := s.NewDeck(title)
		if newDeck.Empty() {
			sendTemplate(w, r, http.StatusOK, "error.html", D{
				"error": "Could not create deck",
			})
			return true
		}

		http.Redirect(w, r, newDeck.EditorURL(), http.StatusSeeOther)

	} else if s.Route(r, http.MethodGet, "decks", &deck, "editor") {
		sendTemplate(w, r, http.StatusOK, "editor.html", D{
			"deck": deck,
		})

	} else if s.Route(r, http.MethodGet, "decks", &deck, "load.json") {
		var output struct {
			Text string `json:"text"`
		}
		output.Text = deck.Buffer
		sendJSON(w, r, http.StatusOK, output)

	} else if s.Route(r, http.MethodPut, "decks", &deck, "show.json") {
		var input struct {
			Show string `json:"show"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			sendPayload(w, r, http.StatusBadRequest, nil)
			return true
		}
		deck.Slide = input.Show

		// send update to all screens
		for _, s := range deck.Screens() {
			s.Show(deck.Slide)
		}

		if ok := deck.Save(); !ok {
			sendPayload(w, r, http.StatusInternalServerError, nil)
			return true
		}

		sendPayload(w, r, http.StatusOK, nil)

	} else if s.Route(r, http.MethodPut, "decks", &deck, "save.json") {
		var input struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			sendPayload(w, r, http.StatusBadRequest, nil)
			return true
		}

		deck.Buffer = input.Text
		deck.Save()

	} else if s.Route(r, http.MethodGet, "decks", &deck, "screen") {
		sendTemplate(w, r, http.StatusOK, "screen.html", D{
			"deck": deck,
		})

	} else if s.Route(r, http.MethodGet, "decks", &deck, "screen.socket") {
		handleSocketConnection(s, deck, w, r)

	} else if s.Route(r, http.MethodGet, "decks", &deck, "refresh.json") {
		cksum, _ := strconv.Atoi(r.URL.Query().Get("cksum"))
		if deck.Checksum == uint32(cksum) {
			sendPayload(w, r, http.StatusOK, nil)
			return true
		}

		var output struct {
			Show     string `json:"show"`
			Checksum uint32 `json:"cksum"`
		}
		output.Show = deck.Slide
		output.Checksum = deck.Checksum
		sendJSON(w, r, http.StatusOK, output)

	} else {
		return false
	}

	return true
}
