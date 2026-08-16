#!/bin/sh
set -eu

# Parse arguments, extracting wrapper-controlled options.
text=""
lang=""
speaker=""
seed=""
threads=""
output=""
max_words=""
max_chars=""
longform_threshold=""

needs_value_for=""

for arg in "$@"; do
    if [ -n "$needs_value_for" ]; then
        case "$needs_value_for" in
            text)    text="$arg" ;;
            lang)    lang="$arg" ;;
            speaker) speaker="$arg" ;;
            seed)    seed="$arg" ;;
            threads) threads="$arg" ;;
            output)  output="$arg" ;;
            max_words) max_words="$arg" ;;
            max_chars) max_chars="$arg" ;;
            longform_threshold) longform_threshold="$arg" ;;
        esac
        needs_value_for=""
        continue
    fi

    case "$arg" in
        --text)      needs_value_for=text ;;
        --text=*)    text="${arg#--text=}" ;;
        --lang)      needs_value_for=lang ;;
        --lang=*)    lang="${arg#--lang=}" ;;
        --speaker)   needs_value_for=speaker ;;
        --speaker=*) speaker="${arg#--speaker=}" ;;
        --seed)      needs_value_for=seed ;;
        --seed=*)    seed="${arg#--seed=}" ;;
        --threads)   needs_value_for=threads ;;
        --threads=*) threads="${arg#--threads=}" ;;
        --output)    needs_value_for=output ;;
        --output=*)  output="${arg#--output=}" ;;
        --max-words) needs_value_for=max_words ;;
        --max-words=*) max_words="${arg#--max-words=}" ;;
        --max-chars) needs_value_for=max_chars ;;
        --max-chars=*) max_chars="${arg#--max-chars=}" ;;
        --longform-threshold) needs_value_for=longform_threshold ;;
        --longform-threshold=*) longform_threshold="${arg#--longform-threshold=}" ;;
        *)
            printf 'unknown option: %s\n' "$arg" >&2
            exit 2
            ;;
    esac
done

if [ -n "$needs_value_for" ]; then
    printf 'missing value for --%s\n' "$needs_value_for" >&2
    exit 2
fi

# Validate required arguments.
if [ -z "$text" ];    then printf 'error: --text is required\n'    >&2; exit 2; fi
if [ -z "$lang" ];    then printf 'error: --lang is required\n'    >&2; exit 2; fi
if [ -z "$speaker" ]; then printf 'error: --speaker is required\n' >&2; exit 2; fi
if [ -z "$output" ];  then printf 'error: --output is required\n'  >&2; exit 2; fi

# Determine the MP3 output path.
case "$output" in
    *.[Ww][Aa][Vv]) mp3_output="${output%.*}.mp3" ;;
    *)               mp3_output="${output}.mp3" ;;
esac

# Create a per-run temp directory and ensure cleanup on exit.
tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/txt2speech-ai.XXXXXX")
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

# Split text into TTS chunks; one chunk per line.
chunks_file="$tmpdir/chunks.txt"
set -- --text "$text"
if [ -n "$max_words" ]; then
    set -- "$@" --max-words "$max_words"
fi
if [ -n "$max_chars" ]; then
    set -- "$@" --max-chars "$max_chars"
fi
if [ -n "$longform_threshold" ]; then
    set -- "$@" --longform-threshold "$longform_threshold"
fi
if ! textchunks "$@" > "$chunks_file"; then
    printf 'error: failed to split text into TTS-safe chunks\n' >&2
    exit 1
fi

if [ ! -s "$chunks_file" ]; then
    printf 'error: no text chunks produced\n' >&2
    exit 1
fi

total_chunks=$(grep -cve '^[[:space:]]*$' "$chunks_file")
if [ "$total_chunks" -eq 0 ]; then
    printf 'error: no text chunks produced\n' >&2
    exit 1
fi

# Synthesize each chunk into its own WAV segment, collecting an ffmpeg concat list.
concat_list="$tmpdir/concat.txt"
: > "$concat_list"

idx=0
while IFS= read -r chunk; do
    [ -z "$chunk" ] && continue
    seg="$tmpdir/seg_${idx}.wav"
    set -- say --model /opt/magpie/model.gguf --lang "$lang" --speaker "$speaker"
    if [ -n "$seed" ]; then
        set -- "$@" --seed "$seed"
    fi
    if [ -n "$threads" ]; then
        set -- "$@" --threads "$threads"
    fi
    set -- "$@" --text "$chunk" --output "$seg"
    if ! magpie-cli "$@"; then
        printf 'error: synthesis failed for chunk %d of %d\n' "$((idx + 1))" "$total_chunks" >&2
        exit 1
    fi
    printf "file '%s'\n" "$seg" >> "$concat_list"
    idx=$((idx + 1))
done < "$chunks_file"

if [ "$idx" -eq 0 ]; then
    printf 'error: no segments synthesized\n' >&2
    exit 1
fi

# Concatenate all WAV segments in order into the final WAV.
if [ "$idx" -eq 1 ]; then
    cp "$tmpdir/seg_0.wav" "$output"
else
    if ! ffmpeg -y -nostdin -f concat -safe 0 \
        -i "$concat_list" \
        -c:a pcm_s16le \
        "$output"; then
        printf 'error: failed to concatenate %d audio chunks\n' "$idx" >&2
        exit 1
    fi
fi

# Convert the final WAV to MP3.
if ! ffmpeg -y -nostdin \
    -i "$output" \
    -vn \
    -codec:a libmp3lame \
    -q:a 2 \
    "$mp3_output"; then
    printf 'error: failed to convert WAV to MP3\n' >&2
    exit 1
fi

printf 'Created MP3: %s\n' "$mp3_output"
