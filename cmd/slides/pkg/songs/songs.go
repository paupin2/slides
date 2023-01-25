package songs

import (
	"net/http"
	"time"

	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/paupin2/slides/pkg/data"
)

type ListItem struct {
	ID       int       `json:"id,omitempty"`
	Title    string    `json:"title,omitempty"`
	Author   string    `json:"author,omitempty"`
	CCLI     string    `json:"ccli,omitempty"`
	Imported bool      `json:"imported,omitempty"`
	Text     string    `json:"text,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

func (li *ListItem) Song() (song *data.Song) {
	if li.ID != 0 {
		song = data.SongByID(li.ID)
	} else {
		song = &data.Song{}
	}

	if song != nil {
		song.Title = li.Title
		song.Author = li.Author
		song.CCLI = li.CCLI
		song.Content = li.Text
	}
	return song
}

func newListItem(s *data.Song) ListItem {
	return ListItem{
		ID:       s.RowID,
		Title:    s.Title,
		Author:   s.Author,
		CCLI:     s.CCLI,
		Imported: s.ExternalID != "",
		Modified: s.Modified,
		Text:     s.Content,
	}
}

const (
	searchLimit = 25
)

func HandleList(req *inout.Request) *inout.Reply {
	req.IsAjax()
	name := req.Str("name").Def("").Get()
	if req.Failed() {
		return inout.Status(http.StatusBadRequest)
	}

	var found []*data.Song
	if name != "" {
		found = data.AllSongsContaining(name, 0, searchLimit)
	} else {
		found = data.AllSongs(0, 0)
	}

	result := []ListItem{}
	for _, s := range found {
		result = append(result, newListItem(s))
	}

	return inout.JSON(result)
}

func HandleGet(req *inout.Request) *inout.Reply {
	req.IsAjax()
	id := req.Int("song_id").Get()
	if req.Failed() {
		return nil
	}

	if found := data.SongByID(id); found != nil {
		return inout.JSON(newListItem(found))
	}

	return inout.Error(http.StatusNotFound, "not found")
}

func HandlePut(req *inout.Request) *inout.Reply {
	req.IsAjax()
	var input ListItem
	if err := req.Read(&input); err != nil {
		return inout.Error(http.StatusBadRequest, "bad input")
	}

	song := input.Song()
	if song == nil {
		return inout.Error(http.StatusNotFound, "not found")
	}

	if err := song.Check(); err != nil {
		return inout.Error(http.StatusBadRequest, "bad data")
	}

	if !song.Save() {
		return inout.Error(http.StatusInternalServerError, "error saving")
	}

	return inout.JSON(newListItem(song))
}

func HandlePost(req *inout.Request) *inout.Reply {
	req.IsAjax()
	var input ListItem
	if err := req.Read(&input); err != nil {
		return inout.Error(http.StatusBadRequest, "bad input")
	}

	if input.ID != 0 {
		return inout.Error(http.StatusBadRequest, "unexpected id")
	}

	song := input.Song()
	if err := song.Check(); err != nil {
		return inout.Error(http.StatusBadRequest, "bad data")
	}

	if !song.Save() {
		return inout.Error(http.StatusInternalServerError, "error saving")
	}

	return inout.JSON(newListItem(song))
}

func HandleDelete(req *inout.Request) *inout.Reply {
	req.IsAjax()
	id := req.Int("song_id").Get()
	if req.Failed() {
		return nil
	}

	song := data.SongByID(id)
	if song == nil {
		return inout.Error(http.StatusNotFound, "not found")
	}

	if err := song.Delete(); err != nil {
		return inout.Error(http.StatusInternalServerError, "could not delete")
	}

	return inout.OK()
}
