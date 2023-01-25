package decks

import (
	"net/http"
	"time"

	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/paupin2/slides/pkg/data"
)

func HandleList(req *inout.Request) *inout.Reply {
	req.IsAjax()
	text := req.Str("text").Def("").Get()
	if req.Failed() {
		return nil
	}

	return inout.JSON(data.ListDecks(text))
}

type DeckReply struct {
	Title    string     `json:"title"`
	Text     string     `json:"text"`
	Created  *time.Time `json:"created,omitempty"`
	Modified *time.Time `json:"modified,omitempty"`
}

func formatDeck(in data.Deck) DeckReply {
	return DeckReply{
		Title:    in.Title,
		Text:     in.Text,
		Created:  &in.Created,
		Modified: &in.Modified,
	}
}

func HandleGet(req *inout.Request) *inout.Reply {
	req.IsAjax()
	title := req.Str("title").Get()
	if req.Failed() {
		return nil
	}

	if deck, found := data.LoadDeck(title); found {
		return inout.JSON(formatDeck(deck))
	}
	return inout.Error(http.StatusNotFound, "not found")
}

func HandlePut(req *inout.Request) *inout.Reply {
	req.IsAjax()
	var dr DeckReply
	if err := req.Read(&dr); err != nil {
		return inout.Error(http.StatusBadRequest, "could not read data")
	}

	if err := data.CheckTitle(dr.Title); err != nil {
		return inout.Error(http.StatusBadRequest, "bad title")
	}

	deck, found := data.LoadDeck(dr.Title)
	if !found {
		deck.Title = dr.Title
	}
	deck.Text = dr.Text

	if err := deck.Save(); err != nil {
		return inout.Error(http.StatusBadRequest, "error: %v", err)
	}

	return inout.OK()
}

func HandleDelete(req *inout.Request) *inout.Reply {
	req.IsAjax()
	title := req.Str("title").Get()
	if req.Failed() {
		return nil
	}

	deck, found := data.LoadDeck(title)
	if !found {
		return inout.Error(http.StatusNotFound, "not found")
	}
	if err := deck.Delete(); err != nil {
		return inout.Error(http.StatusInternalServerError, "error deleting")
	}

	return inout.OK()
}
