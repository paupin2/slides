package planningcenter

import "testing"

func TestCleanSectionName(t *testing.T) {
	check := func(expected string, lines ...string) {
		t.Helper()
		for _, line := range lines {
			if actual := cleanSectionName(line); actual != expected {
				t.Errorf(`while cleaning "%s" expected "%s" but got "%s"`, line, expected, actual)
			}
		}
	}

	check("intro", "INTRO", "INTRO:", "intro:", "intro", "Intro")
	check("intro 2", "INTRO 2", "INTRO 2:", "intro 2:", "intro 2", "Intro 2")
	check("chorus", "CHORUS", "CHORUS:", "chorus:", "chorus", "Chorus")
	check("chorus 1", "CHORUS 1", "CHORUS 1:", "chorus 1:", "chorus 1", "Chorus 1")
	check("verse 1", "VERSE 1", "VERSE 1:", "verse 1:", "verse 1", "Verse 1")
	check("verse 2", "VERSE 2", "VERSE 2:", "verse 2:", "verse 2", "Verse 2")
	check("lole", "lole:", "LOLE:", "Lole:")

	// not section names
	const INVALID = ""
	check(INVALID, "potato", "v1l1", "A", "B", "C", "D", "E", "F", "G", "Am")
}

func TestParseText(t *testing.T) {
	expect := func(actual, expected string) bool {
		t.Helper()
		if actual != expected {
			t.Errorf("expected -------\n%s\nbut got -------\n%s", expected, actual)
			return false
		}
		return true
	}

	check := func(source, expected string, sequence ...string) {
		t.Helper()
		if expect(parseText(source, "", sequence), expected) {
			expect(parseText("", source, sequence), expected)
		}
	}

	check(
		"INTRO:\r\n\nIntro text\nVERSE 1:\nV1L1\nV1L2\nCHORUS:\nCH1L1\nCH1L2\nVERSE 2:\nV2L1",
		`INTRO:

Intro text
VERSE 1:
V1L1
V1L2
CHORUS:
CH1L1
CH1L2
VERSE 2:
V2L1`,
	)

	check(
		`INTRO:

Intro[Bm] te[G]xt
VERSE 1:
V1L1
CHORUS
CH1[|Bm / / /] [|G / / /] [|D / / /] [|A/C# / / /]L1
CH1[A/C#]L2
Verse 2
V2L1`,
		`# Intro

Intro text

# Verse 1
V1L1

# Chorus
CH1   L1
CH1L2

# verse 2
V2L1

# chorus
CH1   L1
CH1L2`,
		"Intro", "Verse 1", "Chorus", "verse 2", "chorus",
	)
}
