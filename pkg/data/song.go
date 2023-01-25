package data

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/rs/zerolog/log"
)

type Song struct {
	RowID      int
	ExternalID string
	Title      string
	Author     string
	CCLI       string
	Content    string
	Created    time.Time
	Modified   time.Time
}

var (
	errBadTitle   = errors.New("title is empty")
	errBadContent = errors.New("content is empty")
)

func (s Song) Check() error {
	if strings.TrimSpace(s.Title) == "" {
		return errBadTitle
	} else if strings.TrimSpace(s.Content) == "" {
		return errBadContent
	}
	return nil
}

func (s *Song) String() string {
	id := "unsaved"
	if s.RowID > 0 {
		id = fmt.Sprint(s.RowID)
	}
	if s.ExternalID != "" {
		id += "/" + s.ExternalID
	}
	return id
}

func (s *Song) Delete() error {
	var count int64
	resp, err := execQuery(`delete from songs where rowid = ?`, s.RowID)
	if err == nil {
		count, err = resp.RowsAffected()
	}
	if err != nil {
		log.Debug().Int("id", s.RowID).Err(err).Msg("could not delete song")
		return err
	}
	if count < 1 {
		return errors.New("not found")
	}

	log.Info().Int("id", s.RowID).Msg("deleted song")
	return nil
}

func scanSong(rows *sql.Rows) (Song, error) {
	var (
		s          Song
		externalID *string
		title      *string
		author     *string
		ccli       *string
		content    *string
	)
	err := rows.Scan(
		&s.RowID,
		&externalID,
		&title,
		&author,
		&ccli,
		&content,
		&s.Created,
		&s.Modified,
	)

	set := func(src, dest *string) {
		if src != nil {
			*dest = *src
		}
	}
	set(externalID, &s.ExternalID)
	set(title, &s.Title)
	set(author, &s.Author)
	set(ccli, &s.CCLI)
	set(content, &s.Content)
	return s, err
}

func querySongs(limit int, whereetc string, args ...interface{}) []*Song {
	query := `
	select rowid, external_id, title, author, ccli, content, created, modified
	from songs
	` + whereetc

	rows, err := runQuery(query, args...)
	if err != nil {
		log.Err(err).Str("sql", query).Interface("args", args).Msg("querying song")
		return nil
	}

	defer rows.Close()
	var found []*Song
	for rows.Next() {
		song, err := scanSong(rows)
		if err != nil {
			log.Err(err).Str("sql", query).Interface("args", args).Msg("scanning song")
			return nil
		}

		found = append(found, &song)
		if limit != 0 && len(found) >= limit {
			break
		}
	}

	return found
}

func querySong(query string, args ...interface{}) *Song {
	if ss := querySongs(1, query, args...); len(ss) > 0 {
		return ss[0]
	}
	return nil
}

func AllSongs(offset, limit int) []*Song {
	if offset == 0 && limit == 0 {
		return querySongs(0, `order by title`)
	}

	return querySongs(limit, `
		order by title
		limit $1
		offset $2
	`, limit, offset)
}

func AllSongsContaining(text string, offset, limit int) []*Song {
	if text = inout.FilterLetters(text); text == "" {
		return AllSongs(offset, limit)
	}

	return querySongs(limit, `
		where title like $1
		order by title
		limit $2
		offset $3
	`, "%"+text+"%", limit, offset)
}

func SongByExternalID(id string) *Song {
	if id == "" {
		return nil
	}
	return querySong(`where external_id = ?`, id)
}

func SongByID(id int) *Song {
	if id == 0 {
		return nil
	}
	return querySong(`where rowid = ?`, id)
}

// Save inserts or updates the song, updating the RowID id the song was
// inserted.
func (s *Song) Save() bool {
	if err := s.Check(); err != nil && s.ExternalID == "" {
		return false
	}

	if s.RowID == 0 && s.ExternalID != "" {
		// is there an external id? is it on the DB? get its rowid
		if existing := SongByExternalID(s.ExternalID); existing != nil {
			s.RowID = existing.RowID
		}
	}

	p := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	if s.RowID == 0 {
		// insert
		res, err := execQuery(`
			insert into songs (external_id, title, author, ccli, content)
			values (?, ?, ?, ?, ?);
		`, p(s.ExternalID), p(s.Title), p(s.Author), p(s.CCLI), p(s.Content))
		if err == nil {
			var id int64
			id, err = res.LastInsertId()
			s.RowID = int(id)
		}
		if err == nil {
			log.Info().Str("song", s.String()).Msg("inserted")
		} else {
			log.Err(err).Str("song", s.String()).Msg("inserting")
		}
		return err == nil
	}

	// update
	rows, err := execQuery(`
		update songs set
			external_id = ?,
			title = ?,
			author = ?,
			ccli = ?,
			content = ?,
			modified = current_timestamp
		where rowid = ?;
	`, p(s.ExternalID), p(s.Title), p(s.Author), p(s.CCLI), p(s.Content),
		s.RowID,
	)
	if err == nil {
		var n int64
		if n, err = rows.RowsAffected(); err == nil && n == 0 {
			return false
		}
	}
	if err != nil {
		log.Err(err).Str("song", s.String()).Err(err).Msg("updating")
		return false
	}

	log.Info().Str("song", s.String()).Err(err).Msg("updated")
	s.Modified = time.Now()
	return true
}

func LastModified(idPrefix string) time.Time {
	def := time.Time{}
	sql := `select max(modified) from songs`
	if idPrefix != "" {
		sql += fmt.Sprintf(` where external_id like '%s%%'`, idPrefix)
	}
	rows, err := runQuery(sql)
	if err != nil {
		log.Err(err).Msg("fetching lastmod")
		return def
	}
	defer rows.Close()
	if rows.Next() {
		var timeStr *string
		if err := rows.Scan(&timeStr); err != nil {
			log.Fatal().Err(err).Msg("scanning lastmod")
		}

		if timeStr != nil {
			t, err := time.Parse("2006-01-02 15:04:05", *timeStr)
			if err != nil {
				log.Err(err).Str("lastmod", *timeStr).Msg("bad lastmod")
			}
			return t
		}
	}
	return def
}
