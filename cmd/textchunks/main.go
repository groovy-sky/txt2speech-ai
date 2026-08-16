// textchunks prints one TTS chunk per line for the supplied text.
// Each chunk fits within the model's roughly-20-second generation window.
// Output lines are plain text; empty input produces no output.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/groovy-sky/txt2speech-ai/textprep"
)

func main() {
	text := flag.String("text", "", "input text to split into chunks (required)")
	flag.Parse()

	if strings.TrimSpace(*text) == "" {
		fmt.Fprintln(os.Stderr, "textchunks: --text is required and must not be empty")
		os.Exit(2)
	}

	chunks, err := textprep.Prepare(*text, textprep.DefaultConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "textchunks: %v\n", err)
		os.Exit(1)
	}

	for _, chunk := range chunks {
		fmt.Println(chunk.Text)
	}
}
