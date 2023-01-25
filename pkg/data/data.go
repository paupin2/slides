package data

import (
	"database/sql"
	_ "embed"
	"errors"
	"flag"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
	"github.com/paupin2/slides/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	SystemUserID = "system"
)

var (
	db *sql.DB

	//go:embed "create.sql"
	sqlCreate string

	systemUser = User{ID: SystemUserID, Name: "System"}
)

func runQuery(query string, args ...interface{}) (*sql.Rows, error) {
	Connect()
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Err(err).Str("sql", query).Msg("query failed")
	}
	return rows, err
}

func execQuery(query string, args ...interface{}) (sql.Result, error) {
	Connect()
	res, err := db.Exec(query, args...)
	if err != nil {
		log.Err(err).Str("sql", query).Msg("query failed")
	}
	return res, err
}

func Connect() {
	if db != nil {
		return
	}

	path := config.Config.Path.Db
	if path == "" {
		flag.Usage()
		os.Exit(1)

	} else if sqlCreate == "" {
		log.Fatal().Msg("no/bad creation SQL")
	}

	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal().Err(err).Msg("opening db")
	}

	if _, err = execQuery(sqlCreate); err != nil {
		log.Fatal().Msg("could not create db")
	}

	log.Info().Str("db", path).Msg("connected")
}

const (
	letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// randomString returns a random string of the specified length
func randomString(length int) string {
	// from: http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
	buf := make([]byte, length)

	// a src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := length-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			buf[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(buf)
}

func titleBefore(a, b string) bool {
	const format = "2006-01-02"
	t1, e1 := time.Parse(format, a)
	t2, e2 := time.Parse(format, b)

	if e1 == nil && e2 == nil {
		return t1.After(t2) // both are dates
	} else if e1 == nil {
		return true // only first is a date
	} else if e2 == nil {
		return false // only second is a date
	} else {
		return a < b // neither is a date
	}
}

var (
	reservedTitles = func() map[string]bool {
		m := map[string]bool{}
		ts := []string{
			"new", "deck", "decks",
			"current", "next", "last",
			"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday",
			"sun", "mon", "tue", "wed", "thu", "fri", "sat",
		}
		for _, t := range ts {
			m[strings.ToLower(t)] = true
		}
		return m
	}()

	errTooShort = errors.New("title is too short")
	errTooLong  = errors.New("title is too long")
	errMustTrim = errors.New("title has leading/trailing spaces")
	errReserved = errors.New("title is reserved")
	errBadChars = errors.New("title has invalid characters")
)

const (
	minTitleLength = 4
	maxTitleLength = 32
)

func CheckTitle(title string) error {
	if len(title) < minTitleLength {
		return errTooShort
	} else if len(title) > maxTitleLength {
		return errTooLong
	} else if title != strings.TrimSpace(title) {
		return errMustTrim
	} else if reservedTitles[strings.ToLower(title)] {
		return errReserved
	}

	const validPunct = " -._"
	for _, c := range title {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			// alphanumeric
		} else if strings.ContainsRune(validPunct, c) {
			// punctuation
		} else {
			return errBadChars
		}
	}

	return nil
}

func SystemUser() User {
	return systemUser
}

func internalLoadUsers(includeSystem bool) (map[string]User, error) {
	rows, err := runQuery(`select id, name from users`)
	if err != nil {
		return nil, err
	}

	list := map[string]User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, err
		}
		if u.ID == SystemUserID && !includeSystem {
			continue
		}
		list[u.ID] = u
	}
	return list, nil
}

func LoadUsers() (map[string]User, error) {
	return internalLoadUsers(false)
}
