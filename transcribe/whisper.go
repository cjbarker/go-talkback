package transcribe

// #cgo LDFLAGS: -lggml
// // ggml_backend_load_all is in libggml (not re-exported by libwhisper).
// // When using the Homebrew whisper-cpp build, backends are plugins in
// // $(GGML_PREFIX)/libexec/*.so and must be loaded explicitly — unlike
// // standalone builds where libggml-cpu.dylib registers itself via a
// // static constructor. Calling this once before model init is idempotent.
// extern void ggml_backend_load_all(void);
import "C"

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

// Transcriber holds a loaded whisper model and transcribes PCM audio.
type Transcriber struct {
	model whisper.Model
}

// New loads a ggml model file and returns a ready Transcriber.
// The model is kept in memory for the lifetime of the application.
func New(modelPath string) (*Transcriber, error) {
	// Load all ggml compute backends (CPU, Metal, BLAS) from the system
	// plugin directory. Required when using the Homebrew whisper-cpp build,
	// which uses ggml's plugin architecture instead of linking backends
	// statically into libggml-cpu.dylib / libggml-metal.dylib.
	C.ggml_backend_load_all()

	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("transcribe: load model %q: %w", modelPath, err)
	}
	return &Transcriber{model: model}, nil
}

// Transcribe converts mono 16 kHz float32 PCM samples to text.
// It returns the concatenated text of all segments.
func (t *Transcriber) Transcribe(samples []float32) (string, error) {
	if len(samples) == 0 {
		return "", nil
	}

	ctx, err := t.model.NewContext()
	if err != nil {
		return "", fmt.Errorf("transcribe: new context: %w", err)
	}

	// Use all available CPU threads for speed; Metal acceleration is
	// automatically used when whisper.cpp was built with WHISPER_METAL=1.
	ctx.SetThreads(uint(runtime.NumCPU()))

	// English-only models (e.g. ggml-base.en.bin) don't support SetLanguage.
	if ctx.IsMultilingual() {
		_ = ctx.SetLanguage("en")
	}

	if err := ctx.Process(samples, nil, nil, nil); err != nil {
		return "", fmt.Errorf("transcribe: process: %w", err)
	}

	var sb strings.Builder
	for {
		seg, err := ctx.NextSegment()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("transcribe: next segment: %w", err)
		}
		text := strings.TrimSpace(seg.Text)
		if text != "" {
			if sb.Len() > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(text)
		}
	}
	return sb.String(), nil
}

// Close releases the model's memory.
func (t *Transcriber) Close() error {
	return t.model.Close()
}
