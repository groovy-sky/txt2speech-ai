#!/bin/sh
set -eu

output=""
needs_output_value=false

for arg in "$@"; do
    if [ "$needs_output_value" = "true" ]; then
        output="$arg"
        needs_output_value=false
        continue
    fi

    case "$arg" in
        --output)
            needs_output_value=true
            ;;
        --output=*)
            output="${arg#--output=}"
            ;;
    esac
done

# Preserve the original CLI behavior if no output path was supplied.
if [ -z "$output" ] || [ "$needs_output_value" = "true" ]; then
    exec magpie-cli say --model /opt/magpie/model.gguf "$@"
fi

# Generate the source WAV first.
magpie-cli say --model /opt/magpie/model.gguf "$@"

# Replace only the final extension, so speech.wav becomes speech.mp3.
case "$output" in
    *.[Ww][Aa][Vv])
        mp3_output="${output%.*}.mp3"
        ;;
    *)
        mp3_output="${output}.mp3"
        ;;
esac

ffmpeg -y -nostdin \
    -i "$output" \
    -vn \
    -codec:a libmp3lame \
    -q:a 2 \
    "$mp3_output"

printf 'Created MP3: %s\n' "$mp3_output"
