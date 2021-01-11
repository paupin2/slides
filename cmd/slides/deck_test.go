package main

import (
	"sort"
	"testing"
)

func TestDeckListSort(t *testing.T) {
	titles := []string{"2020-11-22", "draft", "2020-12-20", "x-slides", "2020-12-06", "2020-11-29", "another", "2020-12-13"}
	expected := []string{"2020-12-20", "2020-12-13", "2020-12-06", "2020-11-29", "2020-11-22", "another", "draft", "x-slides"}

	var dl DeckList
	dl.order = make([]Deck, len(titles))
	for i, t := range titles {
		dl.order[i] = Deck{Title: t}
	}
	sort.Sort(&dl)

	actual := make([]string, len(titles))
	for i, d := range dl.order {
		actual[i] = d.Title
	}
	for i, d := range dl.order {
		if d.Title != expected[i] {
			t.Errorf("error\nexpected: %+v\nbut got:  %+v", expected, actual)
			return
		}
	}
}
