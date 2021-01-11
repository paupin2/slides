package main

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Deck struct {
	list     *DeckList
	Title    string
	Buffer   string
	Slide    string
	Checksum uint32
}

func (deck *Deck) Before(other *Deck) bool {
	const format = "2006-01-02"
	t1, e1 := time.Parse(format, deck.Title)
	t2, e2 := time.Parse(format, other.Title)

	if e1 == nil && e2 == nil {
		return t1.Before(t2) // both are dates
	} else if e1 == nil {
		return false // only first is a date
	} else if e2 == nil {
		return true // only second is a date
	} else {
		return deck.Title > other.Title // neither is a date
	}
}

func (deck *Deck) Filename() string {
	return path.Join(config.Path.Data, deck.Title+".txt")
}

func (deck *Deck) FileExists() bool {
	s, err := os.Stat(deck.Filename())
	return err == nil && !s.IsDir()
}

func (deck *Deck) EditorURL() string {
	return fmt.Sprintf("/decks/%s/editor", deck.Title)
}

func (deck *Deck) Save() bool {
	if deck.list == nil {
		return false
	}
	return deck.list.Save(deck)
}

func (deck *Deck) WriteToDisk() error {
	// create a temporary file
	f, err := ioutil.TempFile(config.Path.Data, fmt.Sprintf("%s-*.tmp", deck.Title))
	if err != nil {
		log.Error().Err(err).Msg("error creating temp file")
		return err
	}
	defer func() {
		if f != nil {
			os.Remove(f.Name())
		}
	}()

	// write content to it
	if _, err = f.WriteString(deck.Buffer); err != nil {
		log.Error().Err(err).Msg("error writing content")
		return err
	}

	// rename the temporary file
	f.Close()
	if err = os.Rename(f.Name(), deck.Filename()); err == nil {
		f = nil
		log.Info().Str("filename", deck.Filename()).Msg("saved")
	} else {
		log.Error().Err(err).Msg("error renaming")
	}
	return err
}

func (deck *Deck) Empty() bool {
	return deck == nil || *deck == Deck{}
}

func (deck *Deck) Screens() []Screen {
	return deck.list.server.GetClients(*deck)
}

type DeckList struct {
	server *Server
	lock   sync.RWMutex
	titles map[string]*Deck
	order  []Deck
}

func (dl *DeckList) Len() int           { return len(dl.order) }
func (dl *DeckList) Less(i, j int) bool { return !dl.order[i].Before(&dl.order[j]) }
func (dl *DeckList) Swap(i, j int)      { dl.order[i], dl.order[j] = dl.order[j], dl.order[i] }

// Add a deck
func (dl *DeckList) Add(title, buffer string) Deck {
	deck := Deck{
		list:   dl,
		Title:  title,
		Buffer: buffer,
	}
	deck.Save()
	return deck
}

// Load a deck from disk
func (dl *DeckList) Load(filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	title := strings.TrimSuffix(path.Base(filename), extension)
	deck := Deck{
		list:     dl,
		Title:    title,
		Checksum: crc32.ChecksumIEEE([]byte("")),
		Buffer:   string(buf),
	}

	// save
	dl.lock.Lock()
	defer dl.lock.Unlock()
	dl.order = append(dl.order, deck)
	dl.titles[deck.Title] = &deck
	sort.Sort(dl)

	log.Info().Str("filename", filename).Msg("loaded")
	return nil
}

// Titles returns the list of titles
func (dl *DeckList) Titles() []string {
	dl.lock.RLock()
	defer dl.lock.RUnlock()
	titles := make([]string, len(dl.order))
	for i, d := range dl.order {
		titles[i] = d.Title
	}
	return titles
}

// Get returns the deck with the specified title
func (dl *DeckList) Get(title string) Deck {
	dl.lock.RLock()
	defer dl.lock.RUnlock()
	if d, found := dl.titles[title]; found {
		return *d
	}
	return Deck{}
}

const (
	minTitleLength = 4
	maxTitleLength = 32
)

var (
	reValidTitle   = regexp.MustCompile(`^(?i)[-\w0-9_.][-\w0-9_. ]+[-\w0-9_.]$`)
	reservedTitles = []string{
		"new", "deck", "decks",
		"current", "next", "last",
		"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday",
		"sun", "mon", "tue", "wed", "thu", "fri", "sat",
	}
)

func allowedTitle(title string) bool {
	if len(title) < minTitleLength || len(title) > maxTitleLength {
		return false
	}

	lower := strings.ToLower(title)
	for _, t := range reservedTitles {
		if lower == t {
			return false
		}
	}

	if !reValidTitle.MatchString(title) {
		return false
	}
	return true
}

// Save the (existing) deck
func (dl *DeckList) Save(deck *Deck) bool {
	if dl.server == nil || deck == nil {
		return false
	}

	// check the title before saving
	if !allowedTitle(deck.Title) {
		return false
	}

	saved, found := dl.titles[deck.Title]
	needSave := saved == nil || deck.Buffer != saved.Buffer
	needUpdate := needSave || deck.Slide != saved.Slide
	if deck.list == nil {
		deck.list = dl
		needUpdate = true
	}

	if !(needSave || needUpdate) {
		return true
	}

	dl.lock.Lock()
	defer dl.lock.Unlock()

	if needUpdate {
		deck.Checksum = crc32.ChecksumIEEE([]byte(deck.Slide))
		if found {
			// first remove
			for idx, d := range dl.order {
				if d.Title == deck.Title {
					dl.order = append(dl.order[:idx], dl.order[idx+1:]...)
					break
				}
			}
		}
		copy := *deck
		dl.order = append(dl.order, copy)
		dl.titles[deck.Title] = &copy
		sort.Sort(dl)
	}

	if needSave {
		return deck.WriteToDisk() == nil
	}

	return true
}

func NewDeckList(srv *Server) DeckList {
	return DeckList{
		server: srv,
		titles: map[string]*Deck{},
	}
}
