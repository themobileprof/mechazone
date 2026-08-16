package ai

import (
	"strings"
	"unicode"
)

var langWords = map[string][]string{
	"en": {"the", "and", "with", "from", "this", "that"},
	"de": {"und", "der", "die", "das", "nicht", "eine"},
	"fr": {"les", "une", "dans", "pour", "avec", "des"},
	"es": {"los", "las", "para", "con", "una", "por"},
	"pt": {"não", "uma", "para", "com", "os", "as"},
	"nl": {"het", "van", "een", "niet", "voor", "op"},
	"it": {"che", "non", "per", "una", "con", "dei"},
	"pl": {"nie", "się", "jest", "na", "do", "jak"},
}

func DetectLanguage(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "und"
	}
	var han, hira, cyr, arab, latin int
	for _, r := range text {
		switch {
		case unicode.In(r, unicode.Han):
			han++
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			hira++
		case unicode.In(r, unicode.Cyrillic):
			cyr++
		case unicode.In(r, unicode.Arabic):
			arab++
		case unicode.IsLetter(r) && r < 0x250:
			latin++
		}
	}
	switch {
	case hira > 20:
		return "ja"
	case han > 20 && hira == 0:
		return "zh"
	case cyr > 20:
		return "ru"
	case arab > 20:
		return "ar"
	}

	lower := " " + strings.ToLower(text) + " "
	best, bestN := "und", 0
	for code, words := range langWords {
		n := 0
		for _, w := range words {
			n += strings.Count(lower, " "+w+" ")
		}
		if n > bestN {
			best, bestN = code, n
		}
	}
	if bestN >= 3 {
		return best
	}
	if latin > 20 {
		return "en"
	}
	return "und"
}
