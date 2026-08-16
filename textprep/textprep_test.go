package textprep

import (
	"reflect"
	"strings"
	"testing"
)

func TestPrepareBehavior(t *testing.T) {
	t.Run("paragraphs", func(t *testing.T) {
		cfg := Config{LongformThreshold: 1}
		input := "First paragraph without punctuation\nstill first paragraph\n\nSecond paragraph here."

		got, err := Prepare(input, cfg)
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}

		want := []Chunk{{
			Index: 0,
			Text:  "First paragraph without punctuation still first paragraph. Second paragraph here.",
			Words: 10,
			Chars: runeLen("First paragraph without punctuation still first paragraph. Second paragraph here."),
			BOT:   true,
			EOT:   true,
		}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v want %#v", got, want)
		}
	})

	t.Run("line breaks", func(t *testing.T) {
		got, err := Prepare("Line one\nline two\nline three.", Config{})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 || got[0].Text != "Line one line two line three." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("bullets", func(t *testing.T) {
		cfg := Config{LongformThreshold: 1}
		input := "• First bullet\n- Second bullet\n✓ Third bullet"
		got, err := Prepare(input, cfg)
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 chunk, got %#v", got)
		}
		want := "First bullet. Second bullet. Third bullet"
		if got[0].Text != want {
			t.Fatalf("got %q want %q", got[0].Text, want)
		}
	})

	t.Run("whitespace and control", func(t *testing.T) {
		input := "A\tB\u00a0C\fD\x00E\vF"
		got, err := Prepare(input, Config{})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 || got[0].Text != "A B C DE F" {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("abbreviations", func(t *testing.T) {
		got, err := Prepare("Dr. Smith went home. He slept.", Config{LongformThreshold: 1})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 || got[0].Text != "Dr. Smith went home. He slept." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("decimals", func(t *testing.T) {
		chunks, err := Prepare("Pi is 3.14. The end.", Config{
			MaxWordsPerChunk:  3,
			MaxCharsPerChunk:  100,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(chunks) != 2 || chunks[0].Text != "Pi is 3.14." || chunks[1].Text != "The end." {
			t.Fatalf("unexpected chunks: %#v", chunks)
		}
	})

	t.Run("initials", func(t *testing.T) {
		got, err := Prepare("J. K. Rowling wrote books. They sold well.", Config{
			MaxWordsPerChunk:  5,
			MaxCharsPerChunk:  200,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 2 || got[0].Text != "J. K. Rowling wrote books." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("urls", func(t *testing.T) {
		got, err := Prepare("Visit https://example.com for details. Thanks.", Config{
			MaxWordsPerChunk:  4,
			MaxCharsPerChunk:  200,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 2 || got[0].Text != "Visit https://example.com for details." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("version numbers", func(t *testing.T) {
		got, err := Prepare("Use v1.2.3 for best results. Upgrade later.", Config{
			MaxWordsPerChunk:  5,
			MaxCharsPerChunk:  200,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 2 || got[0].Text != "Use v1.2.3 for best results." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("punctuation preservation", func(t *testing.T) {
		got, err := Prepare("Wait! Really? Yes!", Config{
			MaxWordsPerChunk:  1,
			MaxCharsPerChunk:  20,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 3 || got[0].Text != "Wait!" || got[1].Text != "Really?" || got[2].Text != "Yes!" {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("sentence packing", func(t *testing.T) {
		got, err := Prepare("One. Two. Three.", Config{
			MaxWordsPerChunk:  10,
			MaxCharsPerChunk:  100,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 || got[0].Text != "One. Two. Three." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("both limits enforced", func(t *testing.T) {
		got, err := Prepare("Alpha beta. Gamma delta.", Config{
			MaxWordsPerChunk:  10,
			MaxCharsPerChunk:  12,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 chunks, got %#v", got)
		}
	})

	t.Run("overlong sentence fallback", func(t *testing.T) {
		text := "alpha beta gamma, delta epsilon zeta, eta theta iota, kappa lambda mu."
		got, err := Prepare(text, Config{
			MaxWordsPerChunk:  3,
			MaxCharsPerChunk:  20,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) < 3 {
			t.Fatalf("expected multiple chunks, got %#v", got)
		}
		for _, chunk := range got {
			if chunk.Words > 3 || chunk.Chars > 20 {
				t.Fatalf("chunk exceeds limit: %#v", chunk)
			}
		}
	})

	t.Run("huge token", func(t *testing.T) {
		text := "supercalifragilisticexpialidocious"
		got, err := Prepare(text, Config{
			MaxWordsPerChunk:  10,
			MaxCharsPerChunk:  5,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) < 2 {
			t.Fatalf("expected token split, got %#v", got)
		}
		var rebuilt strings.Builder
		for _, chunk := range got {
			if chunk.Chars > 5 {
				t.Fatalf("chunk exceeds char limit: %#v", chunk)
			}
			rebuilt.WriteString(chunk.Text)
		}
		if rebuilt.String() != text {
			t.Fatalf("rebuilt token %q want %q", rebuilt.String(), text)
		}
	})

	t.Run("unicode rune counts", func(t *testing.T) {
		text := "こんにちは世界。"
		got, err := Prepare(text, Config{
			MaxWordsPerChunk:  10,
			MaxCharsPerChunk:  7,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 chunks, got %#v", got)
		}
		if got[0].Chars != 7 {
			t.Fatalf("expected rune count 7, got %#v", got[0])
		}
	})

	t.Run("deterministic metadata", func(t *testing.T) {
		got, err := Prepare("One. Two.", Config{
			MaxWordsPerChunk:  1,
			MaxCharsPerChunk:  10,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		want := []Chunk{
			{Index: 0, Text: "One.", Words: 1, Chars: 4, BOT: true, EOT: false},
			{Index: 1, Text: "Two.", Words: 1, Chars: 4, BOT: false, EOT: true},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v want %#v", got, want)
		}
	})

	t.Run("optional normalization", func(t *testing.T) {
		got, err := Prepare("I have 2 apples.", Config{}, func(s string) string {
			return strings.ReplaceAll(s, "2", "two")
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 || got[0].Text != "I have two apples." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("threshold behavior", func(t *testing.T) {
		got, err := Prepare("One. Two. Three.", Config{
			MaxWordsPerChunk:  10,
			MaxCharsPerChunk:  100,
			LongformThreshold: 100,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 || got[0].Text != "One. Two. Three." {
			t.Fatalf("unexpected chunks: %#v", got)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got, err := Prepare(" \n\t\f ", Config{})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil chunks, got %#v", got)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		if _, err := Prepare("hello", Config{MaxWordsPerChunk: -1}); err == nil {
			t.Fatal("expected error for invalid config")
		}
		if _, err := Prepare("hello", Config{MaxCharsPerChunk: -1}); err == nil {
			t.Fatal("expected error for invalid config")
		}
	})

	// --- sentence-splitting and long-text chunk tests ---

	t.Run("short text is one chunk", func(t *testing.T) {
		got, err := Prepare("Hello from Magpie.", Config{})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 chunk, got %d: %#v", len(got), got)
		}
	})

	t.Run("multi-sentence becomes multiple chunks when forced", func(t *testing.T) {
		// Use tight limits to force each sentence into its own chunk.
		got, err := Prepare("First sentence. Second sentence. Third sentence.", Config{
			MaxWordsPerChunk:  2,
			MaxCharsPerChunk:  20,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) < 2 {
			t.Fatalf("expected multiple chunks, got %d: %#v", len(got), got)
		}
		// Chunks must be in order.
		for i, c := range got {
			if c.Index != i {
				t.Fatalf("chunk %d has wrong index %d", i, c.Index)
			}
		}
	})

	t.Run("chunks are non-empty", func(t *testing.T) {
		got, err := Prepare("Alpha. Beta. Gamma.", Config{
			MaxWordsPerChunk:  1,
			MaxCharsPerChunk:  10,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		for _, c := range got {
			if strings.TrimSpace(c.Text) == "" {
				t.Fatalf("empty chunk found: %#v", c)
			}
		}
	})

	t.Run("content is fully preserved across chunks", func(t *testing.T) {
		input := "The quick brown fox. Jumped over the lazy dog. And ran away fast."
		got, err := Prepare(input, Config{
			MaxWordsPerChunk:  4,
			MaxCharsPerChunk:  30,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		var joined strings.Builder
		for _, c := range got {
			if joined.Len() > 0 {
				joined.WriteString(" ")
			}
			joined.WriteString(c.Text)
		}
		// Every word from the original must appear in the reassembled text.
		for _, word := range strings.Fields(input) {
			word = strings.Trim(word, ".?,!")
			if !strings.Contains(joined.String(), word) {
				t.Fatalf("word %q missing from reassembled text %q", word, joined.String())
			}
		}
	})

	t.Run("no chunk exceeds configured limits", func(t *testing.T) {
		const maxWords = 5
		const maxChars = 40
		input := "One two three four five six seven. Eight nine ten eleven twelve thirteen fourteen fifteen."
		got, err := Prepare(input, Config{
			MaxWordsPerChunk:  maxWords,
			MaxCharsPerChunk:  maxChars,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		for _, c := range got {
			if c.Words > maxWords {
				t.Fatalf("chunk %d has %d words > limit %d: %q", c.Index, c.Words, maxWords, c.Text)
			}
			if c.Chars > maxChars {
				t.Fatalf("chunk %d has %d chars > limit %d: %q", c.Index, c.Chars, maxChars, c.Text)
			}
		}
	})

	t.Run("oversized sentence is split into smaller parts", func(t *testing.T) {
		// A single sentence that is far too long for one chunk.
		input := "This is a very long sentence that goes on and on and on without any stopping point whatsoever even though it should have stopped long ago."
		got, err := Prepare(input, Config{
			MaxWordsPerChunk:  10,
			MaxCharsPerChunk:  60,
			LongformThreshold: 1,
		})
		if err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if len(got) < 2 {
			t.Fatalf("expected oversized sentence to be split, got %d chunk(s)", len(got))
		}
		for _, c := range got {
			if c.Words > 10 || c.Chars > 60 {
				t.Fatalf("chunk exceeds limit: words=%d chars=%d text=%q", c.Words, c.Chars, c.Text)
			}
		}
	})

	t.Run("default config limits are sensible for 20-second window", func(t *testing.T) {
		cfg := DefaultConfig()
		// Default limits should be non-zero and reasonable.
		if cfg.MaxWordsPerChunk <= 0 || cfg.MaxWordsPerChunk > 200 {
			t.Fatalf("unexpected MaxWordsPerChunk: %d", cfg.MaxWordsPerChunk)
		}
		if cfg.MaxCharsPerChunk <= 0 || cfg.MaxCharsPerChunk > 1000 {
			t.Fatalf("unexpected MaxCharsPerChunk: %d", cfg.MaxCharsPerChunk)
		}
	})
}
