// textchunks prints one TTS chunk per line for the supplied text.
// Each chunk fits within the model's roughly-20-second generation window.
// Output lines are plain text; empty input produces no output.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/groovy-sky/txt2speech-ai/textprep"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("textchunks", flag.ContinueOnError)
	fs.SetOutput(stderr)

	text := fs.String("text", "", "input text to split into chunks (required)")
	maxWords := fs.Int("max-words", 0, "maximum words per chunk (optional)")
	maxChars := fs.Int("max-chars", 0, "maximum characters/runes per chunk (optional)")
	longformThreshold := fs.Int("longform-threshold", 0, "minimum text length that triggers longform chunking (optional)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*text) == "" {
		fmt.Fprintln(stderr, "textchunks: --text is required and must not be empty")
		return 2
	}

	cfg := textprep.Config{
		MaxWordsPerChunk:  *maxWords,
		MaxCharsPerChunk:  *maxChars,
		LongformThreshold: *longformThreshold,
	}
	chunks, err := textprep.Prepare(*text, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "textchunks: %v\n", err)
		return 1
	}

	for _, chunk := range chunks {
		fmt.Fprintln(stdout, chunk.Text)
	}

	return 0
}
