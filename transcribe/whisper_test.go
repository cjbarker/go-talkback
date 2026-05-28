package transcribe_test

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cbarker/go-talkback/transcribe"
)

const testAudio = "testdata/jfk.wav"

// findModel returns a model path to use in tests. It prefers $TALKBACK_MODEL,
// then the default install location. If neither exists the test is skipped.
func findModel(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("TALKBACK_MODEL"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "share", "go-talkback", "models", "ggml-base.en.bin"),
		filepath.Join(home, ".local", "share", "go-talkback", "models", "ggml-small.en.bin"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no ggml model found; set TALKBACK_MODEL or run 'make model'")
	return ""
}

// TestNew_InvalidPath verifies that loading a non-existent model returns an error.
// This test does not require a real model to be present.
func TestNew_InvalidPath(t *testing.T) {
	_, err := transcribe.New("/nonexistent/model.bin")
	if err == nil {
		t.Fatal("expected error for invalid model path, got nil")
	}
}

// TestTranscribe_EmptySamples verifies that transcribing an empty slice
// returns an empty string without error.
func TestTranscribe_EmptySamples(t *testing.T) {
	modelPath := findModel(t)
	tr, err := transcribe.New(modelPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer tr.Close()

	text, err := tr.Transcribe(nil)
	if err != nil {
		t.Fatalf("Transcribe(nil): %v", err)
	}
	if text != "" {
		t.Errorf("Transcribe(nil) = %q, want empty string", text)
	}

	text, err = tr.Transcribe([]float32{})
	if err != nil {
		t.Fatalf("Transcribe([]): %v", err)
	}
	if text != "" {
		t.Errorf("Transcribe([]) = %q, want empty string", text)
	}
}

// TestTranscribe_JFKSpeech transcribes the classic JFK "Ask not" sample and
// checks that key words appear in the output. This is an integration test
// that requires a ggml model and testdata/jfk.wav.
//
// The sample (~11 s) is the final lines of JFK's 1961 inaugural address:
//
//	"And so, my fellow Americans, ask not what your country can do for you —
//	 ask what you can do for your country."
func TestTranscribe_JFKSpeech(t *testing.T) {
	modelPath := findModel(t)

	if _, err := os.Stat(testAudio); err != nil {
		t.Skipf("test audio not found at %s: %v", testAudio, err)
	}

	samples, err := loadWAV(testAudio)
	if err != nil {
		t.Fatalf("loadWAV: %v", err)
	}
	t.Logf("loaded %d samples (%.1f s)", len(samples), float64(len(samples))/16000)

	tr, err := transcribe.New(modelPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer tr.Close()

	text, err := tr.Transcribe(samples)
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	t.Logf("transcription: %q", text)

	lower := strings.ToLower(text)
	for _, want := range []string{"country", "ask"} {
		if !strings.Contains(lower, want) {
			t.Errorf("expected %q in transcription output %q", want, text)
		}
	}
}

// loadWAV reads a 16-bit PCM WAV file and returns the samples as float32
// in the range [-1, 1]. Only the first channel is used for multi-channel files.
func loadWAV(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Parse RIFF/WAVE header.
	var riff [4]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, err
	}
	var chunkSize uint32
	if err := binary.Read(f, binary.LittleEndian, &chunkSize); err != nil {
		return nil, err
	}
	var wave [4]byte
	if _, err := io.ReadFull(f, wave[:]); err != nil {
		return nil, err
	}

	// Walk sub-chunks until we find "fmt " and "data".
	var (
		numChannels   uint16
		bitsPerSample uint16
		dataSize      uint32
		dataFound     bool
	)
	for {
		var id [4]byte
		if _, err := io.ReadFull(f, id[:]); err != nil {
			break
		}
		var size uint32
		if err := binary.Read(f, binary.LittleEndian, &size); err != nil {
			return nil, err
		}
		switch string(id[:]) {
		case "fmt ":
			var audioFmt uint16
			binary.Read(f, binary.LittleEndian, &audioFmt) // 1 = PCM
			binary.Read(f, binary.LittleEndian, &numChannels)
			var sampleRate uint32
			binary.Read(f, binary.LittleEndian, &sampleRate)
			io.ReadFull(f, make([]byte, 6)) // byte rate + block align
			binary.Read(f, binary.LittleEndian, &bitsPerSample)
			if size > 16 {
				io.ReadFull(f, make([]byte, size-16))
			}
		case "data":
			dataSize = size
			dataFound = true
		default:
			io.ReadFull(f, make([]byte, size))
		}
		if dataFound {
			break
		}
	}
	if !dataFound || numChannels == 0 || bitsPerSample == 0 {
		return nil, os.ErrInvalid
	}

	// Read raw PCM bytes.
	raw := make([]byte, dataSize)
	if _, err := io.ReadFull(f, raw); err != nil {
		return nil, err
	}

	// Convert int16 samples to float32; take only channel 0.
	bytesPerSample := int(bitsPerSample / 8)
	frameSize := bytesPerSample * int(numChannels)
	numFrames := len(raw) / frameSize
	samples := make([]float32, numFrames)
	for i := range samples {
		offset := i*frameSize // channel 0 offset
		s := int16(binary.LittleEndian.Uint16(raw[offset : offset+2]))
		samples[i] = float32(s) / 32768.0
	}
	return samples, nil
}
