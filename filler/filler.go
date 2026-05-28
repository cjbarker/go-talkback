// Package filler removes common English speech filler words and phrases from
// transcribed text before it is typed at the cursor.
package filler

import (
	"regexp"
	"strings"
	"unicode"
)

// words lists unambiguous filler words and phrases, longest first so that
// multi-word phrases are matched before their component words.
var words = []string{
	"you know what I mean",
	"at the end of the day",
	"you know",
	"you see",
	"I mean",
	"I guess",
	"I suppose",
	"or something",
	"believe me",
	"uh huh",
	"mhm",
	"hmm",
	"um",
	"uh",
	"er",
	"basically",
	"actually",
	"okay",
	"so",
}

var patterns []*regexp.Regexp

func init() {
	patterns = make([]*regexp.Regexp, len(words))
	for i, w := range words {
		patterns[i] = regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b[,!?.]?`)
	}
}

// Strip removes filler words and phrases from s, then normalizes whitespace
// and re-capitalizes the first letter.
func Strip(s string) string {
	for _, re := range patterns {
		s = re.ReplaceAllString(s, "")
	}
	s = strings.TrimSpace(strings.Join(strings.Fields(s), " "))
	return capitalizeFirst(s)
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
