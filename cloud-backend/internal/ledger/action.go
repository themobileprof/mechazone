package ledger

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const mergeJaccard = 0.72
const skillJaccard = 0.5

var phraseFold = []struct{ from, to string }{
	{"back-probe", "backprobe"},
	{"back probe", "backprobe"},
	{"from the rear of the connector", "backprobe"},
	{"from the back of the connector", "backprobe"},
	{"still plugged", "backprobe"},
	{"obd-ii", "dlc"},
	{"obd ii", "dlc"},
	{"obd-2", "dlc"},
	{"obd2", "dlc"},
	{"diagnostic link connector", "dlc"},
	{"diagnostic link", "dlc"},
	{"16-pin", "dlc"},
	{"16 pin", "dlc"},
	{"j1962", "dlc"},
	{"dc voltage", "volt"},
	{"dc volts", "volt"},
	{"clear codes", "cleardtc"},
	{"clear dtcs", "cleardtc"},
	{"clear dtc", "cleardtc"},
	{"wiggle test", "wiggle"},
	{"wiggle the loom", "wiggle"},
	{"open circuit", "continuity"},
	{"open wire", "continuity"},
	{"beep test", "continuity"},
	{"diode mode", "continuity"},
}

var tokenFold = map[string]string{
	"ohms": "ohm", "ohm": "ohm", "resistance": "ohm", "resist": "ohm",
	"kohm": "ohm", "megohm": "ohm", "volts": "volt", "voltage": "volt",
	"volt": "volt", "vdc": "volt", "continuity": "continuity", "beep": "continuity",
	"dlc": "dlc", "obd": "dlc", "backprobe": "backprobe", "cleardtc": "cleardtc",
	"wiggle": "wiggle", "measure": "test", "check": "test", "prove": "test",
	"verify": "test", "confirm": "test", "read": "test", "test": "test",
	"connector": "connector", "cavity": "connector", "housing": "connector",
	"battery": "battery", "batt": "battery", "ground": "ground", "earth": "ground",
	"chassis": "ground", "pin": "pin", "code": "code", "codes": "code", "dtc": "code", "dtcs": "code",
}

var actionStop = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "of": {}, "to": {}, "for": {}, "in": {}, "on": {},
	"at": {}, "by": {}, "with": {}, "from": {}, "into": {}, "over": {}, "this": {},
	"that": {}, "those": {}, "these": {}, "is": {}, "are": {}, "be": {}, "been": {},
	"being": {}, "do": {}, "does": {}, "did": {}, "and": {}, "or": {}, "not": {},
	"no": {}, "if": {}, "then": {}, "vs": {}, "versus": {}, "using": {}, "use": {},
	"you": {}, "your": {}, "we": {}, "it": {}, "its": {}, "as": {}, "off": {},
	"per": {}, "via": {}, "than": {}, "so": {}, "too": {}, "just": {}, "also": {},
	"about": {}, "after": {}, "before": {}, "while": {}, "when": {}, "where": {},
	"how": {}, "what": {}, "which": {}, "can": {}, "may": {}, "must": {}, "should": {},
	"would": {}, "will": {}, "through": {}, "between": {}, "without": {}, "within": {},
	"two": {}, "both": {}, "each": {}, "any": {}, "all": {}, "same": {}, "next": {},
}

var (
	spaceRun   = regexp.MustCompile(`\s+`)
	omegaRunes = strings.NewReplacer("Ω", " ohm ", "ω", " ohm ", "kΩ", " ohm ", "µ", " ")
	dtcTok     = regexp.MustCompile(`(?i)^p[0-9a-z]{4}$`)
	hexTok     = regexp.MustCompile(`(?i)^[0-9a-f]{3,4}$`)
)

func normalizeActionKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	if k == "" {
		return "test"
	}
	return k
}

func foldActionText(s string) string {
	s = strings.ToLower(strings.TrimSpace(omegaRunes.Replace(s)))
	for _, p := range phraseFold {
		s = strings.ReplaceAll(s, p.from, " "+p.to+" ")
	}
	return spaceRun.ReplaceAllString(s, " ")
}

func keepActionToken(tok string) bool {
	if tok == "" || len(tok) == 1 {
		return false
	}
	if _, stop := actionStop[tok]; stop {
		return false
	}
	if isAllDigits(tok) {
		return false
	}
	if dtcTok.MatchString(tok) || hexTok.MatchString(tok) {
		return false
	}
	return true
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// ActionTokens is the morph identity for one AI playbook title (kind included in the fingerprint).
func ActionTokens(kind, title string) []string {
	kind = normalizeActionKind(kind)
	folded := foldActionText(title)
	raw := strings.FieldsFunc(folded, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw)+4)
	add := func(tok string) {
		if tok == "" {
			return
		}
		if syn, ok := tokenFold[tok]; ok {
			tok = syn
		}
		if !keepActionToken(tok) && !strings.HasPrefix(tok, "skill:") {
			return
		}
		if _, ok := seen[tok]; ok {
			return
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	for _, tok := range raw {
		add(tok)
	}
	if _, hasClear := seen["clear"]; hasClear {
		if _, hasCode := seen["code"]; hasCode {
			add("cleardtc")
		}
	}
	for _, skill := range []string{"ohm", "volt", "continuity", "dlc", "backprobe", "cleardtc", "wiggle"} {
		if _, ok := seen[skill]; ok {
			add("skill:" + skill)
		}
	}
	if len(out) == 0 {
		add(kind)
	}
	sort.Strings(out)
	return out
}

func ActionFingerprint(kind string, tokens []string) string {
	kind = normalizeActionKind(kind)
	return kind + "|" + strings.Join(tokens, " ")
}

func tokenSet(tokens []string) map[string]struct{} {
	s := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		s[t] = struct{}{}
	}
	return s
}

func Jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	as, bs := tokenSet(a), tokenSet(b)
	inter := 0
	for t := range as {
		if _, ok := bs[t]; ok {
			inter++
		}
	}
	union := len(as) + len(bs) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func sharedSkill(a, b []string) bool {
	as, bs := tokenSet(a), tokenSet(b)
	for t := range as {
		if strings.HasPrefix(t, "skill:") {
			if _, ok := bs[t]; ok {
				return true
			}
		}
	}
	return false
}

func nonSkillOverlap(a, b []string) int {
	as, bs := tokenSet(a), tokenSet(b)
	n := 0
	for t := range as {
		if strings.HasPrefix(t, "skill:") {
			continue
		}
		if _, ok := bs[t]; ok {
			n++
		}
	}
	return n
}

func isSubset(small, big []string) bool {
	b := tokenSet(big)
	for _, t := range small {
		if _, ok := b[t]; !ok {
			return false
		}
	}
	return len(small) > 0
}

func sharedShopSkill(a, b []string) bool {
	as, bs := tokenSet(a), tokenSet(b)
	for _, sk := range []string{"skill:cleardtc", "skill:backprobe", "skill:continuity", "skill:wiggle"} {
		_, aok := as[sk]
		_, bok := bs[sk]
		if aok && bok {
			return true
		}
	}
	return false
}

// ShouldMerge reports whether two same-kind token sets are one action.
func ShouldMerge(kindA, kindB string, a, b []string) bool {
	if normalizeActionKind(kindA) != normalizeActionKind(kindB) {
		return false
	}
	if ActionFingerprint(kindA, a) == ActionFingerprint(kindB, b) {
		return true
	}
	if sharedShopSkill(a, b) {
		return true
	}
	j := Jaccard(a, b)
	if j >= mergeJaccard {
		return true
	}
	if sharedSkill(a, b) && j >= skillJaccard && nonSkillOverlap(a, b) >= 1 {
		return true
	}
	if len(a) >= 3 && len(b) >= 3 && absInt(len(a)-len(b)) <= 2 && (isSubset(a, b) || isSubset(b, a)) {
		return true
	}
	return false
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
