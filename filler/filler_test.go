package filler_test

import (
	"testing"

	"github.com/cbarker/go-talkback/filler"
)

func TestStrip(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// Hesitation sounds
		{"um at start", "Um, I want to go", "I want to go"},
		{"uh mid-sentence", "I want to, uh, go now", "I want to, go now"},
		{"er at end", "I want to go er", "I want to go"},
		{"hmm standalone", "Hmm. That is interesting.", "That is interesting."},
		{"mhm", "Mhm, sounds good", "Sounds good"},
		{"uh huh", "Uh huh, I agree", "I agree"},

		// Multi-word phrases
		{"you know", "You know, it was great", "It was great"},
		{"you know what I mean", "It was loud, you know what I mean?", "It was loud,"},
		{"I mean", "I mean, we should go", "We should go"},
		{"I guess", "I guess we can try", "We can try"},
		{"I suppose", "I suppose that works", "That works"},
		{"you see", "You see, the problem is timing", "The problem is timing"},
		{"or something", "Get a coffee or something", "Get a coffee"},
		{"believe me", "Believe me, this is hard", "This is hard"},
		{"at the end of the day", "At the end of the day, it matters", "It matters"},

		// Case variants
		{"uppercase UM", "UM, hello", "Hello"},
		{"mixed case Uh", "Uh, hello", "Hello"},

		// Trailing punctuation consumed
		{"um with comma", "um, hello", "Hello"},
		{"uh with period", "uh. hello", "Hello"},
		{"um with exclamation", "um! hello", "Hello"},

		// Multiple fillers
		{"multiple fillers", "Um, I mean, you know, let's go", "Let's go"},

		// No fillers — unchanged (modulo capitalization)
		{"no fillers", "Let's go to the store", "Let's go to the store"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},

		// All fillers → empty
		{"all fillers", "um uh er", ""},

		// Re-capitalization after leading filler removed
		{"recapitalize", "um the dog barked", "The dog barked"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filler.Strip(tc.input)
			if got != tc.want {
				t.Errorf("Strip(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
