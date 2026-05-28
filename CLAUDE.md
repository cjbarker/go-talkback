# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`go-talkback` is a macOS menu bar app written in Go. Hold **Option+Space** to record microphone audio; release to transcribe it locally with whisper.cpp (Metal-accelerated) and type the result at whatever text field has focus using CGEvent Unicode injection.

## Build

**Prerequisites:**
```sh
brew install whisper-cpp   # installs libwhisper
brew install ggml          # installs libggml (required as a separate formula)
```

`make build` automatically installs stub dylibs for `libggml-cpu`, `libggml-metal`, and `libggml-blas` into `/opt/homebrew/opt/ggml/lib/`. These stubs are empty — the actual symbols live in the monolithic `libggml` shipped by the Homebrew `ggml` formula. They exist solely to satisfy the whisper.cpp Go bindings (v1.8+), which link against the split ggml backends that Homebrew doesn't build separately.

Build the binary:
```sh
make build        # produces ./talkback (also re-signs with codesign)
```

Or manually (CGo flags must be set):
```sh
CGO_CFLAGS="-I/opt/homebrew/opt/whisper-cpp/include -I/opt/homebrew/opt/ggml/include" \
CGO_LDFLAGS="-L/opt/homebrew/opt/whisper-cpp/lib -rpath /opt/homebrew/opt/whisper-cpp/lib -L/opt/homebrew/opt/ggml/lib -rpath /opt/homebrew/opt/ggml/lib" \
CGO_ENABLED=1 go build -o talkback .
```

> **Note:** `make build` runs `codesign --force --sign -` after the Go linker step. This is required on macOS 15+ because the `-ldflags -X` version substitution invalidates the linker's ad-hoc signature.

## Testing

```sh
make test            # go test ./... -timeout 120s
go test ./audio/...  # single package
```

The `transcribe` integration test requires a downloaded model and is skipped automatically when none is found.

## Running

Download a model first (one-time):
```sh
make model              # downloads ggml-base.en.bin (~142 MB)
make model MODEL=small.en  # or a larger model for better accuracy
```

Run:
```sh
TALKBACK_MODEL=~/.local/share/go-talkback/models/ggml-base.en.bin ./talkback
# or
./talkback -model ~/.local/share/go-talkback/models/ggml-small.en.bin
./talkback -log-level debug   # debug | info | warn | error
```

Config is persisted to `~/.config/go-talkback/config.json` (hotkey, active model, float button state, launch-at-login).

**Required macOS permissions** (prompted on first use or grant manually in System Settings → Privacy & Security):
- Microphone
- Accessibility (for CGEvent to type into other apps)
- Input Monitoring (for global hotkey)

## Packaging

```sh
make app   # builds dist/Talkback.app (requires make build + make icns)
make dmg   # builds dist/Talkback-<version>.dmg
```

## Architecture

The pipeline on hotkey release:

```
malgo audio capture (16 kHz mono float32)
  → recorder.Stop() returns []float32
  → transcribe.Transcribe() calls whisper.cpp Process()
  → inject.Text() calls CGEventKeyboardSetUnicodeString
```

**Thread model:** `systray.Run` owns the main thread (macOS requirement). The hotkey library (`golang.design/x/hotkey`) works within this because systray already runs a Cocoa event loop. All heavy work (whisper inference) runs in a separate goroutine triggered on keyup.

**State machine** (`main.go`): `stateIdle → stateListening → stateProcessing → stateIdle` using `atomic.Int32`. CompareAndSwap guards transitions so concurrent menu clicks and hotkey events don't race.

## Packages

| Package | File(s) | Role |
|---|---|---|
| `audio` | `audio/capture.go` | malgo device: `Start()` / `Stop() []float32` |
| `transcribe` | `transcribe/whisper.go` | Load ggml model once; `Transcribe([]float32) string` |
| `inject` | `inject/inject.go`, `inject/inject.m` | CGo → CoreGraphics CGEvent Unicode injection |
| `hotkey` | `hotkey/hotkey.go`, `hotkey/presets.go` | Thin wrapper around golang.design/x/hotkey; `Presets` slice for menu |
| `config` | `config/config.go` | Load/save JSON config at `~/.config/go-talkback/config.json` |
| `models` | `models/models.go` | Model catalog, path resolution, download with progress callback |
| `dialog` | `dialog/dialog.{go,h,m}` | NSAlert / NSProgressIndicator wrappers (CGo → AppKit) |
| `floatbutton` | `floatbutton/floatbutton.{go,h,m}` | Floating push-to-talk button (NSWindow); `Press`/`Release`/`Moved` channels |
| `launchagent` | `launchagent/launchagent.go` | Install/uninstall `~/Library/LaunchAgents` plist |
| `logger` | `logger/logger.go` | slog level configuration |
| main | `main.go`, `icons.go`, `icons.m` | systray menu bar, state machine, pipeline wiring |

## Key constraints

- `CGO_ENABLED=1` is mandatory — all four external libraries (malgo, whisper.cpp, systray, hotkey) require CGo on macOS.
- Audio buffer is capped at 30 s (whisper.cpp hard limit per `Process()` call).
- English-only ggml models (e.g. `ggml-base.en.bin`) do not support `SetLanguage`; multilingual models do. The transcribe package checks `IsMultilingual()` before calling it.
- The duplicate `-rpath` linker warnings during build are harmless — they come from the whisper.cpp bindings' own `#cgo LDFLAGS` overlapping with what we supply.
