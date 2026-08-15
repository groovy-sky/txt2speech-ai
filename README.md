# Magpie TTS Docker image

Minimal CPU-only container for converting text to a WAV file with NVIDIA's
[`magpie_tts_multilingual_357m.v2602.f16.gguf`](https://huggingface.co/nvidia/magpie_tts_multilingual_357m/blob/main/magpie_tts_multilingual_357m.v2602.f16.gguf).
Inference uses the dependency-light
[`magpie-tts.cpp`](https://github.com/mudler/magpie-tts.cpp) runtime, so the
final image contains no Python, PyTorch, NeMo, CUDA, compilers, or source code.

## Build

The build downloads the model (about 1 GB) into the image:

```sh
docker build -t txt2speech-ai .
```

The inference runtime is pinned to a tested commit. Override
`MAGPIE_TTS_REF` or `MODEL_URL` with Docker build arguments when intentionally
upgrading either artifact.

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

The runtime is MIT licensed. The model weights remain subject to the
[NVIDIA Open Model License](https://www.nvidia.com/en-us/agreements/enterprise-software/nvidia-open-model-agreement/).
