package data

import (
	"sort"
	"testing"
)

func TestDeckTitlesSort(t *testing.T) {
	list := func(titles ...string) (out DeckTitles) {
		for _, t := range titles {
			out = append(out, DeckItem{Title: t})
		}
		return
	}

	titles := list("2020-11-22", "draft", "2020-12-20", "x-slides", "2020-12-06", "2020-11-29", "another", "2020-12-13")
	expected := list("2020-12-20", "2020-12-13", "2020-12-06", "2020-11-29", "2020-11-22", "another", "draft", "x-slides")

	sort.Sort(titles)
	for i := range expected {
		if a, e := titles[i].Title, expected[i].Title; a != e {
			t.Errorf("error\nexpected: %+v\nbut got:  %+v", expected, titles)
			break
		}
	}
}
