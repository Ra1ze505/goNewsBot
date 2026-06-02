package telegramutil

import "unicode"

const MaxMessageLength = 4096

// SplitMessage splits text into Telegram-safe parts not exceeding MaxMessageLength runes.
// It prefers splitting on paragraph breaks, line breaks, and spaces before hard cuts.
func SplitMessage(text string) []string {
	runes := []rune(text)
	if len(runes) <= MaxMessageLength {
		return []string{text}
	}

	var parts []string
	for len(runes) > 0 {
		if len(runes) <= MaxMessageLength {
			parts = append(parts, string(runes))
			break
		}

		window := runes[:MaxMessageLength]
		splitAt := findSplitPoint(window)
		if splitAt <= 0 {
			splitAt = MaxMessageLength
		}

		parts = append(parts, string(runes[:splitAt]))
		runes = trimLeadingSpace(runes[splitAt:])
	}

	return parts
}

func findSplitPoint(runes []rune) int {
	for i := len(runes) - 1; i >= 1; i-- {
		if runes[i-1] == '\n' && runes[i] == '\n' {
			return i + 1
		}
	}

	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '\n' {
			return i + 1
		}
	}

	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsSpace(runes[i]) {
			return i + 1
		}
	}

	return len(runes)
}

func trimLeadingSpace(runes []rune) []rune {
	for len(runes) > 0 && unicode.IsSpace(runes[0]) {
		runes = runes[1:]
	}
	return runes
}
