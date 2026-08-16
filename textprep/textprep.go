package textprep

import (
	"errors"
	"strings"
	"unicode"
)

const (
	defaultMaxWordsPerChunk  = 80
	defaultMaxCharsPerChunk  = 500
	defaultLongformThreshold = 200
)

var errInvalidConfig = errors.New("invalid config")

// Config holds pipeline configuration.
type Config struct {
	MaxWordsPerChunk  int // default 80, min 1
	MaxCharsPerChunk  int // default 500, min 1
	LongformThreshold int // default 200 chars; text at or above this triggers longform multi-chunk; below returns single chunk regardless (but never violates hard limits)
}

// Chunk is a single prepared text chunk.
type Chunk struct {
	Index int
	Text  string
	Words int
	Chars int
	BOT   bool // beginning of text
	EOT   bool // end of text
}

// Normalizer is an optional hook for text normalization (e.g. number expansion).
// It receives sanitized text and returns normalized text.
type Normalizer func(string) string

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxWordsPerChunk:  defaultMaxWordsPerChunk,
		MaxCharsPerChunk:  defaultMaxCharsPerChunk,
		LongformThreshold: defaultLongformThreshold,
	}
}

// Prepare is the main entry point. It runs the full pipeline and returns
// []Chunk. Returns (nil, nil) for empty input. Returns an error for invalid config.
// If len(normalizer) > 0, the first element is called after sanitization.
func Prepare(text string, cfg Config, normalizer ...Normalizer) ([]Chunk, error) {
	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	text = sanitize(text)
	if text == "" {
		return nil, nil
	}

	if len(normalizer) > 0 && normalizer[0] != nil {
		text = strings.TrimSpace(normalizer[0](text))
		if text == "" {
			return nil, nil
		}
	}

	if runeLen(text) < cfg.LongformThreshold && withinLimits(text, cfg) {
		return assignMetadata([]string{text}), nil
	}

	sentences := segment(text)
	if len(sentences) == 0 {
		return nil, nil
	}

	return assignMetadata(pack(sentences, cfg)), nil
}

func normalizeConfig(cfg Config) (Config, error) {
	defaults := DefaultConfig()

	if cfg.MaxWordsPerChunk == 0 {
		cfg.MaxWordsPerChunk = defaults.MaxWordsPerChunk
	}
	if cfg.MaxCharsPerChunk == 0 {
		cfg.MaxCharsPerChunk = defaults.MaxCharsPerChunk
	}
	if cfg.LongformThreshold == 0 {
		cfg.LongformThreshold = defaults.LongformThreshold
	}

	if cfg.MaxWordsPerChunk < 1 || cfg.MaxCharsPerChunk < 1 {
		return Config{}, errInvalidConfig
	}
	if cfg.LongformThreshold < 0 {
		cfg.LongformThreshold = defaults.LongformThreshold
	}

	return cfg, nil
}

func sanitize(text string) string {
	if text == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range text {
		switch r {
		case '\r', '\f', '\v', '\u2028', '\u2029':
			b.WriteRune('\n')
		case '\n':
			b.WriteRune('\n')
		case '\t':
			b.WriteRune(' ')
		default:
			switch {
			case unicode.IsControl(r):
			case unicode.IsSpace(r):
				b.WriteRune(' ')
			default:
				b.WriteRune(r)
			}
		}
	}

	rawLines := strings.Split(b.String(), "\n")
	type lineInfo struct {
		text   string
		bullet bool
	}

	lines := make([]lineInfo, 0, len(rawLines))
	blankRun := 0
	for _, line := range rawLines {
		cleaned, bullet := cleanLine(line)
		if cleaned == "" {
			blankRun++
			if blankRun <= 2 {
				lines = append(lines, lineInfo{})
			}
			continue
		}
		blankRun = 0
		lines = append(lines, lineInfo{text: cleaned, bullet: bullet})
	}

	paragraphs := make([]string, 0)
	current := make([]string, 0)
	for _, line := range lines {
		if line.text == "" {
			if len(current) > 0 {
				paragraphs = append(paragraphs, strings.Join(current, " "))
				current = current[:0]
			}
			continue
		}
		if line.bullet && len(current) > 0 {
			paragraphs = append(paragraphs, strings.Join(current, " "))
			current = current[:0]
		}
		current = append(current, line.text)
		if line.bullet {
			paragraphs = append(paragraphs, strings.Join(current, " "))
			current = current[:0]
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, strings.Join(current, " "))
	}

	var out strings.Builder
	for i, paragraph := range paragraphs {
		paragraph = collapseSpaces(strings.TrimSpace(paragraph))
		if paragraph == "" {
			continue
		}
		if out.Len() == 0 {
			out.WriteString(paragraph)
			continue
		}
		prev := lastNonSpaceRune(out.String())
		if isSentenceEnd(prev) {
			out.WriteString(" ")
		} else {
			out.WriteString(". ")
		}
		out.WriteString(paragraph)
		if i == len(paragraphs)-1 {
			break
		}
	}

	return collapseSpaces(strings.TrimSpace(out.String()))
}

func cleanLine(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}

	runes := []rune(line)
	i := 0
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	bullet := i < len(runes) && isBulletMarker(runes[i])
	if bullet {
		i++
		for i < len(runes) && unicode.IsSpace(runes[i]) {
			i++
		}
		line = string(runes[i:])
	}

	return collapseSpaces(strings.TrimSpace(line)), bullet
}

func isBulletMarker(r rune) bool {
	switch r {
	case '•', '◦', '▪', '▸', '✓', '✗', '✔', '–', '—', '*', '-':
		return true
	default:
		return false
	}
}

func collapseSpaces(s string) string {
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

func segment(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	runes := []rune(text)
	var sentences []string
	start := 0

	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '.', '!', '?':
			if !isBoundary(runes, i) {
				continue
			}

			end := i + 1
			for end < len(runes) {
				if strings.ContainsRune(`"'”’)]}`, runes[end]) {
					end++
					continue
				}
				if (runes[end] == '!' || runes[end] == '?') && (runes[end-1] == '!' || runes[end-1] == '?') {
					end++
					continue
				}
				break
			}

			part := strings.TrimSpace(string(runes[start:end]))
			if part != "" {
				sentences = append(sentences, part)
			}
			start = end
			for start < len(runes) && unicode.IsSpace(runes[start]) {
				start++
			}
			i = end - 1
		}
	}

	if start < len(runes) {
		tail := strings.TrimSpace(string(runes[start:]))
		if tail != "" {
			sentences = append(sentences, tail)
		}
	}

	return sentences
}

func isBoundary(runes []rune, i int) bool {
	r := runes[i]
	if r == '!' || r == '?' {
		return true
	}

	prev := rune(0)
	if i > 0 {
		prev = runes[i-1]
	}
	next := rune(0)
	if i+1 < len(runes) {
		next = runes[i+1]
	}

	if unicode.IsDigit(prev) && unicode.IsDigit(next) {
		return false
	}

	token := surroundingToken(runes, i)
	lowerToken := strings.ToLower(token)
	if isURLToken(lowerToken) {
		return false
	}
	if abbreviationSet[lowerToken] {
		return false
	}

	if unicode.IsUpper(prev) && isInitialToken(runes, i) {
		return false
	}

	return true
}

func surroundingToken(runes []rune, i int) string {
	start := i
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	end := i + 1
	for end < len(runes) && !unicode.IsSpace(runes[end]) {
		end++
	}
	token := strings.Trim(string(runes[start:end]), `"'“”‘’()[]{}<>`)
	return token
}

func isURLToken(token string) bool {
	return strings.HasPrefix(token, "http://") || strings.HasPrefix(token, "https://") || strings.HasPrefix(token, "www.")
}

func isInitialToken(runes []rune, i int) bool {
	start := i - 1
	if start < 0 {
		return false
	}
	if start > 0 && !unicode.IsSpace(runes[start-1]) {
		return false
	}

	j := i + 1
	for j < len(runes) && unicode.IsSpace(runes[j]) {
		j++
	}
	return j < len(runes) && unicode.IsUpper(runes[j])
}

var abbreviationSet = map[string]bool{
	"mr.":     true,
	"mrs.":    true,
	"dr.":     true,
	"prof.":   true,
	"sr.":     true,
	"jr.":     true,
	"vs.":     true,
	"etc.":    true,
	"i.e.":    true,
	"e.g.":    true,
	"no.":     true,
	"fig.":    true,
	"dept.":   true,
	"inc.":    true,
	"ltd.":    true,
	"corp.":   true,
	"st.":     true,
	"ave.":    true,
	"blvd.":   true,
	"ph.d.":   true,
	"m.d.":    true,
	"b.a.":    true,
	"m.a.":    true,
	"approx.": true,
	"est.":    true,
	"vol.":    true,
	"ch.":     true,
	"pp.":     true,
	"ed.":     true,
	"rev.":    true,
}

func pack(sentences []string, cfg Config) []string {
	chunks := make([]string, 0, len(sentences))
	var current string

	flush := func() {
		if strings.TrimSpace(current) != "" {
			chunks = append(chunks, strings.TrimSpace(current))
			current = ""
		}
	}

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		if !withinLimits(sentence, cfg) {
			flush()
			chunks = append(chunks, fallbackSplit(sentence, cfg)...)
			continue
		}

		candidate := sentence
		if current != "" {
			candidate = current + " " + sentence
		}
		if withinLimits(candidate, cfg) {
			current = candidate
			continue
		}

		flush()
		current = sentence
	}

	flush()
	return chunks
}

func fallbackSplit(sentence string, cfg Config) []string {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return nil
	}

	var parts []string
	remaining := sentence
	for remaining != "" {
		if withinLimits(remaining, cfg) {
			parts = append(parts, remaining)
			break
		}

		runes := []rune(remaining)
		lastGood := 0
		words := 0
		inWord := false

		for i, r := range runes {
			if unicode.IsSpace(r) {
				inWord = false
			} else if !inWord {
				words++
				inWord = true
			}

			if i+1 > cfg.MaxCharsPerChunk || words > cfg.MaxWordsPerChunk {
				break
			}
			lastGood = i + 1
		}

		if lastGood == 0 {
			lastGood = minInt(cfg.MaxCharsPerChunk, len(runes))
		}

		breakPos := 0
		for i := lastGood - 1; i >= 1; i-- {
			if runes[i] == ',' || runes[i] == ';' || runes[i] == ':' {
				breakPos = i + 1
				break
			}
		}
		if breakPos == 0 {
			for i := lastGood - 1; i >= 1; i-- {
				if unicode.IsSpace(runes[i]) {
					breakPos = i
					break
				}
			}
		}
		if breakPos == 0 {
			breakPos = lastGood
		}

		part := strings.TrimSpace(string(runes[:breakPos]))
		if part == "" {
			part = string(runes[:lastGood])
			breakPos = lastGood
		}

		parts = append(parts, part)
		remaining = strings.TrimSpace(string(runes[breakPos:]))
	}

	return parts
}

func assignMetadata(chunks []string) []Chunk {
	if len(chunks) == 0 {
		return nil
	}

	out := make([]Chunk, 0, len(chunks))
	last := len(chunks) - 1
	for i, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		out = append(out, Chunk{
			Index: i,
			Text:  chunk,
			Words: len(strings.Fields(chunk)),
			Chars: runeLen(chunk),
			BOT:   i == 0,
			EOT:   i == last,
		})
	}
	return out
}

func withinLimits(text string, cfg Config) bool {
	return len(strings.Fields(text)) <= cfg.MaxWordsPerChunk && runeLen(text) <= cfg.MaxCharsPerChunk
}

func runeLen(s string) int {
	return len([]rune(s))
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?'
}

func lastNonSpaceRune(s string) rune {
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		if !unicode.IsSpace(runes[i]) {
			return runes[i]
		}
	}
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
