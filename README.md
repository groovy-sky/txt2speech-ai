# Magpie TTS Docker image

Minimal CPU-only container for converting text to a WAV file using the
[`magpie-tts.cpp`](https://github.com/mudler/magpie-tts.cpp) runtime.  The
default model is the F16 GGUF from the compatible prebuilt collection
[`mudler/magpie-tts.cpp-gguf`](https://huggingface.co/mudler/magpie-tts.cpp-gguf):

> `magpie-tts-multilingual-357m-f16.gguf`

The final image contains no Python, PyTorch, NeMo, CUDA, compilers, or source
code.

## Build

The build downloads the model (~784 MB) into the image:

```sh
docker build -t txt2speech-ai .
```

The inference runtime is pinned to a tested commit. Override `MAGPIE_TTS_REF`
or `MODEL_URL` with Docker build arguments when intentionally upgrading either
artifact.  Any replacement model **must** be a GGUF that carries the
`magpie.*` metadata expected by this runtime (i.e. sourced from
`mudler/magpie-tts.cpp-gguf` or converted with the same tooling).  A
build-time smoke test (`magpie-cli info`) will fail the image build
immediately if an incompatible GGUF is supplied.

For a smaller image you can use the quantised variant instead:

```sh
docker build \
  --build-arg MODEL_URL=https://huggingface.co/mudler/magpie-tts.cpp-gguf/resolve/main/magpie-tts-multilingual-357m-q8_0.gguf \
  -t txt2speech-ai .
```

## Publish to GitHub Container Registry

The GitHub Actions workflow builds pull requests and publishes
`ghcr.io/<owner>/<repository>` when `Dockerfile` or a file under `docker/`
changes on any branch. It authenticates with the built-in `GITHUB_TOKEN`, so
no custom repository variables or secrets are required.

The workflow publishes `latest` from the default branch, the branch or Git
tag, and a `sha-<commit>` tag. Pull requests only build the image and do not
push it.

## Convert text to audio

```sh
mkdir -p output
docker run --rm \
  -v "$PWD/output:/output" \
  txt2speech-ai \
  --text "Hello from Magpie." \
  --lang en \
  --speaker Aria \
  --output /output/speech.wav
```

The container entrypoint splits the supplied text into bounded chunks (each
fitting within the model's roughly 20-second generation window), synthesizes
the chunks serially with `magpie-cli`, concatenates the resulting WAV segments
in the original order, and finally converts the combined WAV to MP3 using
`ffmpeg` with the libmp3lame codec at VBR quality 2 (`-q:a 2`).  Both files
are kept, so the above command produces:

```
output/
├── speech.wav
└── speech.mp3
```

For text longer than a single model generation (multi-sentence or long-form
input), pass the full text as one `--text` value.  The entrypoint handles
splitting automatically:

```sh
docker run --rm \
  -v "$PWD/output:/output" \
  txt2speech-ai \
  --text "The quick brown fox jumped over the lazy dog. This is a second sentence that will be synthesized as a separate chunk and then joined. A third sentence completes the example." \
  --lang en \
  --speaker Aria \
  --output /output/speech.wav
```

Each sentence (or sub-sentence piece if a sentence is unusually long) is
synthesized independently; the final WAV and MP3 contain all segments in
order.

The core CLI options `--text`, `--lang`, `--speaker`, and `--output` are
required. `--seed`, `--threads`, and the chunk-tuning flags are optional:

| Option | Description |
|---|---|
| `--text TEXT` | Input text (required) |
| `--lang CODE` | Language code, e.g. `en` (required) |
| `--speaker NAME` | Speaker voice, e.g. `Aria` (required) |
| `--output PATH` | Output WAV path (required) |
| `--seed N` | RNG seed for deterministic generation (optional) |
| `--threads N` | CPU thread count (optional) |
| `--max-words N` | Override the per-chunk word cap used for long-text splitting (optional) |
| `--max-chars N` | Override the per-chunk rune/character cap used for long-text splitting (optional) |
| `--longform-threshold N` | Override the text length at which longform chunking is forced (optional) |

Available speakers: `Aria`, `Jason`, `John`, `Leo`, `Sofia`.  Supported
language codes: `en`, `es`, `de`, `fr`, `it`, `pt-BR`, `hi`, `vi`, `ko`,
`ar-AE`, `ar-SA`, `ar-MSA`.

The runtime is MIT licensed. The model weights are distributed under the
[`mudler/magpie-tts.cpp-gguf`](https://huggingface.co/mudler/magpie-tts.cpp-gguf)
repository license.

## Go text preparation package

This repository also includes a small Go package for preparing free-form text
for NVIDIA Magpie TTS longform synthesis. It sanitizes copied text, segments it
into sentence-like units, and packs those units into bounded chunks with
deterministic metadata.

```go
package main

import (
	"fmt"

	"github.com/groovy-sky/txt2speech-ai/textprep"
)

func main() {
	chunks, err := textprep.Prepare("Dr. Smith went home.\nHe slept.", textprep.DefaultConfig())
	if err != nil {
		panic(err)
	}

	for _, chunk := range chunks {
		fmt.Printf("%d: %q BOT=%t EOT=%t\n", chunk.Index, chunk.Text, chunk.BOT, chunk.EOT)
	}
}
```

Run the package tests with:

```sh
go test ./textprep/
```

The `textchunks` CLI also exposes the same optional chunking controls used by the
container entrypoint:

```sh
go run ./cmd/textchunks \
  --text "First sentence. Second sentence. Third sentence." \
  --max-words 2 \
  --max-chars 20 \
  --longform-threshold 1
```
