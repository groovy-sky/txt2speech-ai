#!/bin/sh
set -eu

# Parse arguments, extracting wrapper-controlled options.
text=""
lang=""
speaker=""
seed=""
threads=""
output=""

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

# Build the base magpie-cli argument list (text and output are per-chunk).
magpie_base="--model /opt/magpie/model.gguf --lang $lang --speaker $speaker"
if [ -n "$seed" ];    then magpie_base="$magpie_base --seed $seed"; fi
if [ -n "$threads" ]; then magpie_base="$magpie_base --threads $threads"; fi

# Create a per-run temp directory and ensure cleanup on exit.
tmpdir=$(mktemp -d)
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

# Split text into TTS chunks; one chunk per line.
chunks_file="$tmpdir/chunks.txt"
textchunks --text "$text" > "$chunks_file"

if [ ! -s "$chunks_file" ]; then
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
    # shellcheck disable=SC2086
    magpie-cli say $magpie_base --text "$chunk" --output "$seg"
    printf 'file %s\n' "$seg" >> "$concat_list"
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
    ffmpeg -y -nostdin -f concat -safe 0 \
        -i "$concat_list" \
        -c:a pcm_s16le \
        "$output"
fi

# Convert the final WAV to MP3.
ffmpeg -y -nostdin \
    -i "$output" \
    -vn \
    -codec:a libmp3lame \
    -q:a 2 \
    "$mp3_output"

printf 'Created MP3: %s\n' "$mp3_output"
