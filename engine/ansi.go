package engine

import (
	"regexp"
	"strings"
)

var tokRe = regexp.MustCompile(`⟦(\w)\|([^⟧]*)⟧`)

// ANSI wandelt das Markup in Terminal-Farben um (für den SSH-Server).
func ANSI(s string) string {
	s = tokRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := tokRe.FindStringSubmatch(m)
		farbe := map[string]string{"d": "1;34", "f": "0", "x": "1;32", "c": "4;36"}[sub[1]]
		return "\x1b[" + farbe + "m" + Label(sub[1], sub[2]) + "\x1b[0m"
	})
	return strings.NewReplacer(
		"«", "\x1b[1;33m", "»", "\x1b[0m",
		"⟪", "\x1b[1;31m", "⟫", "\x1b[0m",
		"‹", "\x1b[90m", "›", "\x1b[0m",
	).Replace(s)
}

// Plain entfernt das Markup komplett.
func Plain(s string) string {
	s = tokRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := tokRe.FindStringSubmatch(m)
		return Label(sub[1], sub[2])
	})
	return strings.NewReplacer("«", "", "»", "", "⟪", "", "⟫", "", "‹", "", "›", "").Replace(s)
}
