package docker_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEntrypointLongTextSequentialMerge(t *testing.T) {
	repoRoot := repositoryRoot(t)
	tmpRoot := t.TempDir()
	binDir := filepath.Join(tmpRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(binDir): %v", err)
	}

	buildTextchunks(t, repoRoot, filepath.Join(binDir, "textchunks"))
	writeExecutable(t, filepath.Join(binDir, "magpie-cli"), fakeMagpieScript)
	writeExecutable(t, filepath.Join(binDir, "ffmpeg"), fakeFFmpegScript)

	outputDir := filepath.Join(tmpRoot, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(outputDir): %v", err)
	}
	outputWAV := filepath.Join(outputDir, "speech.wav")
	magpieLog := filepath.Join(tmpRoot, "magpie.log")
	stateDir := filepath.Join(tmpRoot, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(stateDir): %v", err)
	}

	cmd := exec.Command("/bin/sh",
		filepath.Join(repoRoot, "docker", "entrypoint.sh"),
		"--text", "First part. Second part. Third part.",
		"--lang", "en",
		"--speaker", "Aria",
		"--output", outputWAV,
		"--max-words", "2",
		"--max-chars", "20",
		"--longform-threshold", "1",
	)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"MAGPIE_LOG="+magpieLog,
		"MAGPIE_STATE_DIR="+stateDir,
		"TMPDIR="+tmpRoot,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("entrypoint failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	logLines := readTrimmedLines(t, magpieLog)
	wantChunks := []string{
		"1|First part.",
		"2|Second part.",
		"3|Third part.",
	}
	if len(logLines) != len(wantChunks) {
		t.Fatalf("magpie log lines = %q, want %q", logLines, wantChunks)
	}
	for i := range wantChunks {
		if logLines[i] != wantChunks[i] {
			t.Fatalf("magpie log line %d = %q, want %q", i, logLines[i], wantChunks[i])
		}
	}

	wavBytes, err := os.ReadFile(outputWAV)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", outputWAV, err)
	}
	wantWAV := strings.Join([]string{
		"MERGED",
		"SEGMENT:First part.",
		"SEGMENT:Second part.",
		"SEGMENT:Third part.",
	}, "\n") + "\n"
	if string(wavBytes) != wantWAV {
		t.Fatalf("merged wav = %q, want %q", string(wavBytes), wantWAV)
	}

	mp3Bytes, err := os.ReadFile(strings.TrimSuffix(outputWAV, ".wav") + ".mp3")
	if err != nil {
		t.Fatalf("ReadFile(mp3): %v", err)
	}
	wantMP3 := "MP3\n" + wantWAV
	if string(mp3Bytes) != wantMP3 {
		t.Fatalf("mp3 contents = %q, want %q", string(mp3Bytes), wantMP3)
	}

	assertNoChunkTempDirs(t, tmpRoot)
}

func TestEntrypointFailureHasChunkContextAndCleansUp(t *testing.T) {
	repoRoot := repositoryRoot(t)
	tmpRoot := t.TempDir()
	binDir := filepath.Join(tmpRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(binDir): %v", err)
	}

	buildTextchunks(t, repoRoot, filepath.Join(binDir, "textchunks"))
	writeExecutable(t, filepath.Join(binDir, "magpie-cli"), fakeMagpieScript)
	writeExecutable(t, filepath.Join(binDir, "ffmpeg"), fakeFFmpegScript)

	outputWAV := filepath.Join(tmpRoot, "speech.wav")
	magpieLog := filepath.Join(tmpRoot, "magpie.log")
	stateDir := filepath.Join(tmpRoot, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(stateDir): %v", err)
	}

	cmd := exec.Command("/bin/sh",
		filepath.Join(repoRoot, "docker", "entrypoint.sh"),
		"--text", "First part. Second part. Third part.",
		"--lang", "en",
		"--speaker", "Aria",
		"--output", outputWAV,
		"--max-words", "2",
		"--max-chars", "20",
		"--longform-threshold", "1",
	)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"MAGPIE_LOG="+magpieLog,
		"MAGPIE_STATE_DIR="+stateDir,
		"MAGPIE_FAIL_CHUNK=2",
		"TMPDIR="+tmpRoot,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("entrypoint succeeded unexpectedly")
	}
	if !strings.Contains(stderr.String(), "chunk 2 of 3") {
		t.Fatalf("stderr %q did not mention failed chunk context", stderr.String())
	}
	if _, statErr := os.Stat(outputWAV); !os.IsNotExist(statErr) {
		t.Fatalf("expected no wav output, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(strings.TrimSuffix(outputWAV, ".wav") + ".mp3"); !os.IsNotExist(statErr) {
		t.Fatalf("expected no mp3 output, stat err = %v", statErr)
	}

	logLines := readTrimmedLines(t, magpieLog)
	wantPrefix := []string{"1|First part.", "2|Second part."}
	if len(logLines) != len(wantPrefix) {
		t.Fatalf("magpie log lines = %q, want %q", logLines, wantPrefix)
	}
	for i := range wantPrefix {
		if logLines[i] != wantPrefix[i] {
			t.Fatalf("magpie log line %d = %q, want %q", i, logLines[i], wantPrefix[i])
		}
	}

	assertNoChunkTempDirs(t, tmpRoot)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(file))
}

func buildTextchunks(t *testing.T, repoRoot, output string) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", output, "./cmd/textchunks")
	cmd.Dir = repoRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build textchunks failed: %v\nstderr:\n%s", err, stderr.String())
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func readTrimmedLines(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func assertNoChunkTempDirs(t *testing.T, tmpRoot string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(tmpRoot, "txt2speech-ai.*"))
	if err != nil {
		t.Fatalf("Glob(temp dirs): %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary chunk dirs were not cleaned up: %v", matches)
	}
}

const fakeMagpieScript = `#!/bin/sh
set -eu

log=${MAGPIE_LOG:?}
state_dir=${MAGPIE_STATE_DIR:?}
fail_chunk=${MAGPIE_FAIL_CHUNK:-}
mkdir -p "$state_dir"

text=""
output=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        say)
            shift
            ;;
        --text)
            text="$2"
            shift 2
            ;;
        --output)
            output="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

count_file="$state_dir/count"
count=0
if [ -f "$count_file" ]; then
    count=$(cat "$count_file")
fi
count=$((count + 1))
printf '%s' "$count" > "$count_file"
printf '%s|%s\n' "$count" "$text" >> "$log"

if [ -n "$fail_chunk" ] && [ "$count" -eq "$fail_chunk" ]; then
    printf 'synthetic magpie failure\n' >&2
    exit 9
fi

printf 'SEGMENT:%s\n' "$text" > "$output"
`

const fakeFFmpegScript = `#!/bin/sh
set -eu

input=""
output=""
concat_mode=0
prev=""
for arg in "$@"; do
    if [ "$prev" = "-i" ]; then
        input="$arg"
    fi
    if [ "$arg" = "-f" ]; then
        prev="$arg"
        continue
    fi
    if [ "$prev" = "-f" ] && [ "$arg" = "concat" ]; then
        concat_mode=1
    fi
    prev="$arg"
    output="$arg"
done

if [ "$concat_mode" -eq 1 ]; then
    {
        printf 'MERGED\n'
        while IFS= read -r line; do
            case "$line" in
                "file '"*)
                    segment=${line#file \'}
                    segment=${segment%\'}
                    ;;
                "file "*)
                    segment=${line#file }
                    ;;
                *)
                    continue
                    ;;
            esac
            cat "$segment"
        done < "$input"
    } > "$output"
    exit 0
fi

{
    printf 'MP3\n'
    cat "$input"
} > "$output"
`
