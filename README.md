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

The container entrypoint first runs Magpie to produce the WAV, then automatically
converts it to MP3 using `ffmpeg` with the libmp3lame codec at VBR quality 2
(`-q:a 2`).  Both files are kept, so the above command produces:

```
output/
├── speech.wav
└── speech.mp3
```

If `--output` is not supplied the container falls back to executing Magpie
directly (no MP3 conversion is attempted).

Optional flags include `--seed N` for deterministic generation and
`--threads N` to control CPU use. Available speakers are `Aria`, `Jason`,
`John`, `Leo`, and `Sofia`. Supported language codes are `en`, `es`, `de`,
`fr`, `it`, `pt-BR`, `hi`, `vi`, `ko`, `ar-AE`, `ar-SA`, and `ar-MSA`.

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
