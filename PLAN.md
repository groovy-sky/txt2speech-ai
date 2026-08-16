# PLAN.md

# Magpie TTS Longform Text Preparation Plan

## Purpose

Implement a Go text-preparation pipeline for NVIDIA Magpie TTS longform synthesis.

The pipeline must accept arbitrary free-format text and produce natural, bounded chunks suitable for sequential TTS generation. It should prioritize sentence boundaries, avoid splitting common abbreviations or numeric values incorrectly, and provide fallback word/character splitting for unusually long or poorly formatted input.

The output will be consumed by a stateful Magpie TTS backend that preserves longform context across chunks.

---

## Goals

- Accept free-format input:
  - paragraphs
  - line breaks
  - bullet lists
  - copied web content
  - inconsistent whitespace
  - plain text with little punctuation
- Normalize and sanitize text before synthesis.
- Split text into natural sentence-oriented chunks.
- Support configurable:
  - maximum words per chunk
  - maximum characters per chunk
  - longform trigger threshold
- Avoid splitting at:
  - common abbreviations
  - decimal numbers
  - initials
  - URLs where possible
  - version numbers
- Preserve punctuation and readable phrasing.
- Provide deterministic chunk metadata:
  - chunk index
  - text
  - word count
  - character count
  - beginning-of-text flag
  - end-of-text flag
- Provide fallback behavior when a sentence is too long.

---

## Non-Goals

This package will not:

- Run Magpie inference directly.
- Manage PyTorch/TensorRT tensors.
- Perform neural text normalization.
- Guarantee exact speech duration.
- Guarantee linguistically perfect sentence segmentation for every language.
- Concatenate WAV files or decode Magpie audio codec tokens.

The package prepares text and chunk metadata only. The TTS backend is responsible for retaining Magpie longform state and decoding generated audio.

---

## High-Level Pipeline

```text
Raw Input
   |
   v
Sanitizer / Normalizer
   |
   v
Optional Text Normalization
   |
   v
Sentence Segmentation
   |
   v
Sentence Packing
   |
   +--> Chunk fits word + character limits
   |
   v
Long-Sentence Fallback Splitter
   |
   v
Chunk Metadata Assignment
   |
   v
[]Chunk
```
