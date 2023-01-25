package data

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/rs/zerolog/log"
)

type Deck struct {
	Title    string    `json:"title"`
	Text     string    `json:"text"`
	Creator  User      `json:"creator"`
	LastMod  User      `json:"last_mod"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

func (deck *Deck) Before(other *Deck) bool {
	return titleBefore(deck.Title, other.Title)
}

var errCouldNotSave = errors.New("could not save, try again later")

func (d Deck) Valid() bool {
	return CheckTitle(d.Title) == nil
}

func (d Deck) Save() error {
	if err := CheckTitle(d.Title); err != nil {
		return err
	}

	now := time.Now()
	if d.Created.IsZero() {
		d.Created = now
	}
	d.Modified = now

	if !d.Creator.Valid() {
		d.Creator = SystemUser()
	}
	if !d.LastMod.Valid() {
		d.LastMod = SystemUser()
	}

	_, err := execQuery(`
		insert into decks (title, text, creator, lastmod, created, modified)
		values (?, ?, ?, ?, ?, ?)
		on conflict(title)
		do update set
			text = excluded.text,
			creator = excluded.creator,
			lastmod = excluded.lastmod,
			created = excluded.created,
			modified = excluded.modified;
	`,
		d.Title, d.Text,
		d.Creator.ID, d.LastMod.ID,
		d.Created, d.Modified,
	)
	if err != nil {
		log.Debug().Str("title", d.Title).Err(err).Msg("could not save")
		return errCouldNotSave
	}
	log.Debug().Str("title", d.Title).Msg("saved")
	return nil
}

func (d Deck) Delete() error {
	var count int64
	resp, err := execQuery(`delete from decks where title = ?`, d.Title)
	if err == nil {
		count, err = resp.RowsAffected()
	}
	if err != nil {
		log.Debug().Str("title", d.Title).Err(err).Msg("could not delete")
		return errCouldNotSave
	}
	if count < 1 {
		return errors.New("not found")
	}

	log.Debug().Str("title", d.Title).Msg("deleted")
	return nil
}

func ResolveAliases(title string) string {
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

// LoadDeck loads a deck by its id
func LoadDeck(title string) (Deck, bool) {
	rows, err := runQuery(`
		select
			D.title, D.text,
			D.created, D.modified,
			coalesce(UC.username, "system"), coalesce(UC.name, "System"),
			coalesce(UM.username, "system"), coalesce(UM.name, "System")
		from decks D
		left join users UC on (UC.username = D.creator)
		left join users UM on (UM.username = D.lastmod)
		where D.title = ?
		limit 1;
	`, ResolveAliases(title))
	if err != nil {
		log.Fatal().Msg("could not load decks")
	}
	defer rows.Close()

	var d Deck
	if !rows.Next() {
		return d, false
	}
	err = rows.Scan(
		&d.Title, &d.Text,
		&d.Created, &d.Modified,
		&d.Creator.ID, &d.Creator.Name,
		&d.LastMod.ID, &d.LastMod.Name,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not scan from decks")
	}

	return d, true
}

type Decks []Deck

func (ds Decks) Empty() bool {
	return len(ds) == 0
}

func (ds Decks) Len() int           { return len(ds) }
func (ds Decks) Less(i, j int) bool { return ds[i].Before(&ds[j]) }
func (ds Decks) Swap(i, j int)      { ds[i], ds[j] = ds[j], ds[i] }

type DeckItem struct {
	Title string `json:"title"`
	Songs []int  `json:"songs,omitempty"`
}

type DeckTitles []DeckItem

func (ds DeckTitles) Len() int           { return len(ds) }
func (ds DeckTitles) Less(i, j int) bool { return titleBefore(ds[i].Title, ds[j].Title) }
func (ds DeckTitles) Swap(i, j int)      { ds[i], ds[j] = ds[j], ds[i] }

// LoadDecks returns a sorted list of decks on the database
func LoadDecks() Decks {
	rows, err := runQuery(`
		select
			D.title, D.text,
			D.created, D.modified,
			coalesce(UC.username, "system"), coalesce(UC.name, "System"),
			coalesce(UM.username, "system"), coalesce(UM.name, "System")
		from decks D
		left join users UC on (UC.username = D.creator)
		left join users UM on (UM.username = D.lastmod)
	`)
	if err != nil {
		log.Fatal().Msg("could not load decks")
	}

	var decks Decks
	for rows.Next() {
		var d Deck
		err = rows.Scan(
			&d.Title, &d.Text,
			&d.Created, &d.Modified,
			&d.Creator.ID, &d.Creator.Name,
			&d.LastMod.ID, &d.LastMod.Name,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("could not scan from decks")
		}
		decks = append(decks, d)
	}
	sort.Sort(decks)
	return decks
}

var reLabelSongId = regexp.MustCompile(`^\s*#.*\(@([0-9]+)\)`)

// ListDecks returns a sorted list of decks
func ListDecks(text string) DeckTitles {
	var args []any
	query := `select title, text from decks`
	if text = inout.FilterLetters(text); text != "" {
		query += ` where text like $1 limit 25`
		args = append(args, text)
	}

	var ls DeckTitles
	rows, err := runQuery(query, args...)
	if err != nil {
		log.Err(err).Msg("querying decks")
	}
	defer rows.Close()

	for rows.Next() {
		var title, text *string
		if err := rows.Scan(&title, &text); err == nil && title != nil {
			item := DeckItem{Title: *title}
			if text != nil {
				for _, line := range strings.Split(*text, "\n") {
					if m := reLabelSongId.FindStringSubmatch(line); len(m) > 1 {
						if id, err := strconv.Atoi(m[1]); err == nil {
							item.Songs = append(item.Songs, id)
						}
					}
				}
			}

			ls = append(ls, item)
		}
	}

	sort.Sort(ls)
	return ls
}

func ImportDecks(base string) {
	log.Info().Str("path", base).Msg("loading decks")

	err := filepath.Walk(base, func(name string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		ext := path.Ext(name)
		if ext != ".txt" {
			return nil
		}
		title := path.Base(strings.TrimSuffix(name, ext))

		buf, err := ioutil.ReadFile(name)
		if err != nil {
			log.Fatal().Err(err).Str("filename", name).Msg("reading")
			return nil
		}

		d := Deck{
			Title:    title,
			Text:     string(buf),
			Modified: info.ModTime(),
		}

		if err := d.Save(); err != nil {
			log.Fatal().Str("filename", name).Err(err).Msg("Could not save")
		}
		log.Info().Str("filename", name).Msg("imported")
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Str("path", base).Msg("loading files from disk")
	}
}
