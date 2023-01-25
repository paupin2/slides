package inout

import "unicode"

// FilterLetters returns only the letters on the text
func FilterLetters(s string) string {
	filtered := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsLetter(r) {
			if lower := unicode.ToLower(r); lower != unicode.ReplacementChar {
				filtered = append(filtered, lower)
			} else {
				filtered = append(filtered, r)
			}
		}
	}
	return string(filtered)
}
