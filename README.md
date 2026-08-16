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

The result is a 22.05 kHz mono PCM WAV at `output/speech.wav`.

Optional flags include `--seed N` for deterministic generation and
`--threads N` to control CPU use. Available speakers are `Aria`, `Jason`,
`John`, `Leo`, and `Sofia`. Supported language codes are `en`, `es`, `de`,
`fr`, `it`, `pt-BR`, `hi`, `vi`, `ko`, `ar-AE`, `ar-SA`, and `ar-MSA`.

The runtime is MIT licensed. The model weights are distributed under the
[`mudler/magpie-tts.cpp-gguf`](https://huggingface.co/mudler/magpie-tts.cpp-gguf)
repository license.
