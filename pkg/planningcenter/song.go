package planningcenter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/paupin2/slides/pkg/data"
)

const (
	TimeFormat = time.RFC3339
	IDPrefix   = "planningcenter:"
)

type Song struct {
	ID         string `json:"id"`
	Attributes struct {
		Admin                   string `json:"admin"`                      // "EMI Christian Music Publishing",
		Author                  string `json:"author"`                     // "Jonas Myrin and Matt Redman",
		CcliNumber              int    `json:"ccli_number"`                // 6016351,
		Copyright               string `json:"copyright"`                  // "2011 Thankyou Music, Said And Done Music, sixsteps Music, and SHOUT! Publishing",
		CreatedAt               string `json:"created_at"`                 // "2014-03-06T09:11:55Z",
		Hidden                  bool   `json:"hidden"`                     // false,
		LastScheduledAt         string `json:"last_scheduled_at"`          // "2019-09-01T15:00:00Z",
		LastScheduledShortDates string `json:"last_scheduled_short_dates"` // "Sept 1, 2019",
		Notes                   string `json:"notes"`                      // null,
		Themes                  string `json:"themes"`                     // ", Adoration, Blessing, Christian Life, Praise",
		Title                   string `json:"title"`                      // "10,000 Reasons (Bless The Lord)",
		UpdatedAt               string `json:"updated_at"`                 // "2015-07-14T19:45:23Z"
	} `json:"attributes"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}

func (s Song) CreatedAt() time.Time {
	t, _ := time.Parse(TimeFormat, s.Attributes.CreatedAt)
	return t
}

func (s Song) LastScheduledAt() time.Time {
	t, _ := time.Parse(TimeFormat, s.Attributes.LastScheduledAt)
	return t
}

func (s Song) UpdatedAt() time.Time {
	t, _ := time.Parse(TimeFormat, s.Attributes.UpdatedAt)
	return t
}

func dumpJSON(v interface{}) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "\t")
	_ = enc.Encode(v)
	return buf.String()
}

func (s Song) String() string {
	return dumpJSON(s)
}

func (s Song) Fetch() (data.Song, error) {
	ds := data.Song{
		RowID:      0,
		ExternalID: IDPrefix + s.ID,
		Title:      s.Attributes.Title,
		Author:     s.Attributes.Author,
		Created:    s.CreatedAt(),
		Modified:   s.UpdatedAt(),
	}

	if s.Attributes.CcliNumber > 0 {
		ds.CCLI = fmt.Sprint(s.Attributes.CcliNumber)
	}

	var reply struct {
		Data []struct {
			Attributes struct {
				Chords   string   `json:"chord_chart"`
				Lyrics   string   `json:"lyrics"`
				Sequence []string `json:"sequence"`
			} `json:"attributes"`
		} `json:"data"`
	}

	err := Call(fmt.Sprintf("/services/v2/songs/%s/arrangements", s.ID), nil, &reply)
	if err == nil {
		for _, data := range reply.Data {
			parsed := parseText(
				data.Attributes.Chords,
				data.Attributes.Lyrics,
				data.Attributes.Sequence,
			)
			if parsed != "" {
				ds.Content = parsed
				break
			}
		}
	}

	return ds, err
}

var (
	reSectionNames = []*regexp.Regexp{
		regexp.MustCompile(`^\s*([A-Z]{2,}\s*\d*)\s*$`),
		regexp.MustCompile(`^\s*((?i)(verse|chorus|intro)(\s*\d+)?)\s*$`),
		regexp.MustCompile(`^\s*((?i)([a-z]+\s*\d*)):\s*$`),
	}
	reCleanReplacements = []struct {
		from *regexp.Regexp
		to   string
	}{
		{regexp.MustCompile(`\r`), ""},
		{regexp.MustCompile(`\[[A-G#m|/ ]+\]`), ""},
	}
)

func cleanSectionName(line string) string {
	for _, re := range reSectionNames {
		if match := re.FindStringSubmatch(line); match != nil {
			return strings.TrimSpace(strings.ToLower(match[1]))
		}
	}
	return ""
}

func parseText(chords, lyrics string, sequence []string) string {
	source := chords
	if source == "" {
		source = lyrics
	}
	lines := strings.Split(source, "\n")
	if len(lines) == 0 {
		return ""
	}

	cleanLine := func(s string) string {
		for _, repl := range reCleanReplacements {
			s = repl.from.ReplaceAllString(s, repl.to)
		}
		return strings.TrimSpace(s)
	}

	useSequence := len(sequence) > 0
	var result []string
	var currentKey string
	sectionContent := map[string][]string{}
	for _, line := range lines {
		line = cleanLine(line)
		if !useSequence {
			result = append(result, line)

		} else if key := cleanSectionName(line); key != "" {
			currentKey = key

		} else if currentKey == "" {
			// text before first key: append directly to result
			result = append(result, line)

		} else {
			sectionContent[currentKey] = append(sectionContent[currentKey], line)
		}
	}

	if useSequence {
		for _, s := range sequence {
			key := cleanSectionName(s)
			if ls := sectionContent[key]; len(ls) > 0 {
				if len(result) > 0 {
					result = append(result, "")
				}
				result = append(result, fmt.Sprintf("# %s", s))
				result = append(result, ls...)
			}
		}
	}
	return strings.Join(result, "\n")
}
