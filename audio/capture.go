package audio

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

const (
	sampleRate = 16000
	channels   = 1
	// maxSeconds is the whisper.cpp hard limit per transcription call.
	maxSeconds = 30
	maxSamples = sampleRate * maxSeconds
)

// Recorder captures mono 16 kHz float32 PCM from the default microphone.
type Recorder struct {
	mu      sync.Mutex
	buf     []float32
	device  *malgo.Device
	ctx     *malgo.AllocatedContext
	running bool
}

// New allocates a Recorder. Call Start/Stop around each recording session.
func New() (*Recorder, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: init context: %w", err)
	}
	return &Recorder{ctx: ctx}, nil
}

// Start begins capturing audio. It is safe to call Start again after Stop.
func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}

	r.buf = r.buf[:0]

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = channels
	deviceConfig.SampleRate = sampleRate
	deviceConfig.Alsa.NoMMap = 1

	onRecv := func(_, input []byte, frameCount uint32) {
		// malgo gives us raw bytes; reinterpret as float32.
		n := int(frameCount) * channels
		floats := unsafe.Slice((*float32)(unsafe.Pointer(&input[0])), n)
		r.mu.Lock()
		defer r.mu.Unlock()
		if len(r.buf) < maxSamples {
			remaining := maxSamples - len(r.buf)
			if n > remaining {
				n = remaining
			}
			r.buf = append(r.buf, floats[:n]...)
		}
	}

	callbacks := malgo.DeviceCallbacks{Data: onRecv}
	dev, err := malgo.InitDevice(r.ctx.Context, deviceConfig, callbacks)
	if err != nil {
		return fmt.Errorf("audio: init device: %w", err)
	}
	if err := dev.Start(); err != nil {
		dev.Uninit()
		return fmt.Errorf("audio: start device: %w", err)
	}

	r.device = dev
	r.running = true
	return nil
}

// Stop halts capture and returns the recorded PCM samples.
func (r *Recorder) Stop() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}
	r.device.Stop()
	r.device.Uninit()
	r.device = nil
	r.running = false

	out := make([]float32, len(r.buf))
	copy(out, r.buf)
	r.buf = r.buf[:0]
	return out
}

// Close releases the malgo context. Call when the application exits.
func (r *Recorder) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		r.device.Stop()
		r.device.Uninit()
		r.running = false
	}
	_ = r.ctx.Uninit()
	r.ctx.Free()
}
