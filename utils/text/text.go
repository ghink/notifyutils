package text

import (
	"strings"
	"unicode/utf8"

	"go.gh.ink/notifyutils/model"
)

// Subst replaces every ${key} occurrence in template with the matching var value.
// Placeholders with no matching var are left untouched, which keeps a typo visible
// in the delivered message instead of silently erasing it.
func Subst(template string, vars model.Vars) string {
	for _, v := range vars {
		if v == nil {
			continue
		}
		template = strings.ReplaceAll(template, "${"+v.Key+"}", v.Value)
	}
	return template
}

// Chunk splits s into pieces of at most max runes, so a message longer than a
// channel's per-message ceiling spans several sends without losing characters.
// It breaks on the last whitespace that still fits in a piece, and only cuts
// mid-word when a single run of text leaves no choice.
func Chunk(s string, max int) []string {
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return []string{s}
	}

	var chunks []string
	for len(runes) > max {
		cut := breakAt(runes[:max])
		if piece := strings.TrimRight(string(runes[:cut]), " \t\r\n"); piece != "" {
			chunks = append(chunks, piece)
		}
		runes = trimLeading(runes[cut:])
	}
	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}
	return chunks
}

// breakAt reports where a full window should end: just past the last whitespace it
// contains, else the whole window. It never returns 0, which keeps Chunk progressing.
func breakAt(window []rune) int {
	for i := len(window) - 1; i > 0; i-- {
		if window[i] == '\n' || window[i] == ' ' || window[i] == '\t' {
			return i + 1
		}
	}
	return len(window)
}

func trimLeading(runes []rune) []rune {
	i := 0
	for i < len(runes) && (runes[i] == '\n' || runes[i] == ' ' || runes[i] == '\r') {
		i++
	}
	return runes[i:]
}

// Runes counts characters, the unit channels such as Telegram and Discord state
// their limits in.
func Runes(s string) int {
	return utf8.RuneCountInString(s)
}
