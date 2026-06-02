package telegramutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitMessageShortTextUnchanged(t *testing.T) {
	text := "Последние новости:\nКороткая сводка"
	parts := SplitMessage(text)

	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	if parts[0] != text {
		t.Fatalf("expected unchanged text, got %q", parts[0])
	}
}

func TestSplitMessageLongASCIIWithinLimit(t *testing.T) {
	text := strings.Repeat("a", MaxMessageLength+500)
	parts := SplitMessage(text)

	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}

	rejoined := strings.Join(parts, "")
	if rejoined != text {
		t.Fatalf("rejoined text does not match original")
	}

	for i, part := range parts {
		if len([]rune(part)) > MaxMessageLength {
			t.Fatalf("part %d exceeds limit: %d runes", i, len([]rune(part)))
		}
	}
}

func TestSplitMessageLongRussianTextUsesRuneLength(t *testing.T) {
	text := strings.Repeat("ж", MaxMessageLength+100)
	parts := SplitMessage(text)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}

	if len([]rune(parts[0])) != MaxMessageLength {
		t.Fatalf("first part length = %d, want %d", len([]rune(parts[0])), MaxMessageLength)
	}
	if len([]rune(parts[1])) != 100 {
		t.Fatalf("second part length = %d, want 100", len([]rune(parts[1])))
	}

	if !utf8.ValidString(parts[0]) || !utf8.ValidString(parts[1]) {
		t.Fatal("split produced invalid UTF-8")
	}
}

func TestSplitMessagePrefersParagraphBreak(t *testing.T) {
	prefix := strings.Repeat("a", MaxMessageLength-10)
	text := prefix + "\n\n" + strings.Repeat("b", 200)
	parts := SplitMessage(text)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if !strings.HasSuffix(parts[0], "\n\n") {
		t.Fatalf("expected first part to end with paragraph break")
	}
	if !strings.HasPrefix(parts[1], strings.Repeat("b", 10)) {
		t.Fatalf("expected second part to start with paragraph content, got %q", parts[1])
	}
}

func TestSplitMessagePrefersLineBreak(t *testing.T) {
	prefix := strings.Repeat("a", MaxMessageLength-5)
	text := prefix + "\n" + strings.Repeat("b", 200)
	parts := SplitMessage(text)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if !strings.HasSuffix(parts[0], "\n") {
		t.Fatalf("expected first part to end with line break")
	}
}

func TestSplitMessagePrefersSpace(t *testing.T) {
	prefix := strings.Repeat("a", MaxMessageLength-5)
	text := prefix + " " + strings.Repeat("b", 200)
	parts := SplitMessage(text)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if !strings.HasSuffix(parts[0], " ") {
		t.Fatalf("expected first part to end with split space")
	}
}

func TestSplitMessageHardCutWithoutDelimiters(t *testing.T) {
	text := strings.Repeat("x", MaxMessageLength+1)
	parts := SplitMessage(text)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if len([]rune(parts[0])) != MaxMessageLength {
		t.Fatalf("first part length = %d, want %d", len([]rune(parts[0])), MaxMessageLength)
	}
	if parts[1] != "x" {
		t.Fatalf("second part = %q, want %q", parts[1], "x")
	}
}
