# go-talkback

A macOS menu bar app that transcribes speech to text locally and types the result wherever your cursor is.

Hold **Option+Space** → speak → release → text appears.

Transcription runs entirely on-device via [whisper.cpp](https://github.com/ggml-org/whisper.cpp) with Metal GPU acceleration. No audio leaves your machine.

## Requirements

- macOS 13+
- Apple Silicon or Intel Mac with Metal support
- Go 1.21+
- [Homebrew](https://brew.sh)

## Setup

**1. Install dependencies**

```sh
brew install whisper-cpp
brew install ggml
```

**2. Download a model**

```sh
make model                    # ggml-base.en (~142 MB, fastest)
make model MODEL=small.en     # ggml-small.en (~466 MB, more accurate)
make model MODEL=medium.en    # ggml-medium.en (~1.5 GB)
```

Models are saved to `~/.local/share/go-talkback/models/`.

**3. Build**

```sh
make build
```

This produces `./talkback`.

## Running

```sh
make run
# or
./talkback
./talkback -model ~/.local/share/go-talkback/models/ggml-small.en.bin
```

A microphone icon appears in the menu bar. On first launch, macOS will prompt for:

- **Microphone** — to capture audio
- **Accessibility** — to type text into other apps via CGEvent
- **Input Monitoring** — to detect the global hotkey

Grant all three in **System Settings → Privacy & Security**.

## Usage

| Action | Effect |
|---|---|
| Hold **Option+Space** | Start recording |
| Release **Option+Space** | Transcribe and type result at cursor |
| Click menu bar icon | Open menu |
| Menu → Start/Stop Listening | Toggle recording without hotkey |
| Menu → Model: … | Open model submenu |
| Menu → Model → *name* | Switch active transcription model (downloads if needed) |
| Menu → Hotkey: … | Open hotkey submenu |
| Menu → Hotkey → *preset* | Switch push-to-talk hotkey |
| Menu → Show Floating Button | Show the on-screen push-to-talk button |
| Menu → Hide Floating Button | Dismiss the floating button |
| Menu → Launch at Login | Toggle automatic startup at login |
| Menu → Quit | Exit |

The transcribed text is typed into whatever application has keyboard focus — terminal, browser, editor, chat app, etc.

### Floating button

**Menu → Show Floating Button** displays a small circular 🎤 button that hovers above all windows. Push and hold it to record; release to transcribe and type.

- Drag it anywhere on screen to reposition
- Dark gray when idle, red while recording
- Uses `NSWindowStyleMaskNonactivatingPanel` so clicking it never steals focus from the text field you're dictating into — the transcribed text always goes to the right place

### Hotkey options

| Hotkey | Notes |
|---|---|
| `Option+Space` | Default |
| `Cmd+Shift+Space` | |
| `Ctrl+Space` | |
| `Option+R` | |
| `Cmd+Shift+R` | |

The selection is saved to `~/.config/go-talkback/config.json` and restored on next launch.

## Model comparison

| Model | Size | Speed (M-series) | Accuracy |
|---|---|---|---|
| `base.en` | 142 MB | ~1–2 s for 10 s of speech | Good for clear speech |
| `small.en` | 466 MB | ~2–4 s | Better with accents/noise |
| `medium.en` | 1.5 GB | ~5–8 s | Near human-level |
| `large-v3` | 3.1 GB | ~10–15 s | Best accuracy, multilingual |

English-only models (`*.en`) are faster and more accurate for English. Use `large-v3` (no `.en`) if you need multilingual support.

## CLI reference

```
talkback [flags]

Flags:
  -model <path>       Path to ggml model file (overrides $TALKBACK_MODEL)
  -log-level <level>  Logging verbosity: debug, info, warn, error  (default: warn)
  -help               Show full help and exit
```

Run `talkback -help` for the full reference including model descriptions and required permissions.

## Configuration

| Source | Key | Default | Description |
|---|---|---|---|
| Flag | `-model` | — | Path to ggml model file; takes precedence over `$TALKBACK_MODEL` |
| Env | `TALKBACK_MODEL` | `~/.local/share/go-talkback/models/ggml-base.en.bin` | Path to ggml model file |
| Flag | `-log-level` | `warn` | Log verbosity |
| Config file | `hotkey` | `Option+Space` | Push-to-talk hotkey (set via Hotkey submenu) |
| Config file | `model` | `base.en` | Active model name (set via Model submenu) |
| Config file | `showFloatButton` | `false` | Whether the floating button is visible |
| Config file | `launchAtLogin` | `false` | Whether to start at login (set via menu) |

**Config file:** `~/.config/go-talkback/config.json`

```json
{
  "hotkey": "Option+Space",
  "model": "base.en",
  "showFloatButton": false,
  "launchAtLogin": false
}
```

All settings are persisted automatically when changed via the menu.

## Building manually

If you need to build without `make`:

```sh
CGO_CFLAGS="-I/opt/homebrew/opt/whisper-cpp/include -I/opt/homebrew/opt/ggml/include" \
CGO_LDFLAGS="-L/opt/homebrew/opt/whisper-cpp/lib -rpath /opt/homebrew/opt/whisper-cpp/lib \
             -L/opt/homebrew/opt/ggml/lib -rpath /opt/homebrew/opt/ggml/lib" \
CGO_ENABLED=1 go build -o talkback .
```

The `WHISPER_PREFIX` and `GGML_PREFIX` variables in the Makefile can be overridden if the libraries are installed to non-default locations:

```sh
make build WHISPER_PREFIX=/usr/local/opt/whisper-cpp GGML_PREFIX=/usr/local/opt/ggml
```

## Testing

```sh
make test
```

This runs all packages. The transcription integration test automatically skips if no model is found.

### Test coverage

| Package | Tests | Notes |
|---|---|---|
| `config` | Load defaults, empty hotkey fallback, round-trip save/load, JSON validity | Pure unit tests; uses temp `$HOME` |
| `logger` | Valid levels, case-insensitive input, invalid level errors | Pure unit tests |
| `hotkey` | `Find` known/unknown, duplicate names, default present in presets | Pure unit tests |
| `audio` | Recorder create/close, `Stop` before `Start`, idempotent `Stop` | Skipped if no audio hardware |
| `transcribe` | Invalid model path, empty samples, JFK speech accuracy | Integration; skipped if no model |

### Transcription integration test

`TestTranscribe_JFKSpeech` loads `transcribe/testdata/jfk.wav` (JFK's 1961 inaugural address, ~11 s at 16 kHz mono) and asserts that the output contains the words `ask` and `country`. It runs in under 1 second on Apple Silicon with Metal acceleration.

The test requires a ggml model. It checks `$TALKBACK_MODEL` first, then the default install path. Run `make model` to download one if needed.

```sh
# Run only the transcription tests with verbose output
go test ./transcribe/... -v -timeout 120s

# Point at a specific model
TALKBACK_MODEL=~/.local/share/go-talkback/models/ggml-base.en.bin go test ./transcribe/... -v
```

### Manual end-to-end verification

**Audio capture**
```sh
# Hold Option+Space for ~3 seconds; debug logs show sample count on release
./talkback -log-level debug
```

**Text injection**
```sh
# Open TextEdit or any text field, run talkback, dictate a sentence
# Confirm the transcribed text appears at cursor
```

## Troubleshooting

**`whisper.h` not found**
Ensure `brew install whisper-cpp` and `brew install ggml` completed. Default paths are `/opt/homebrew/opt/whisper-cpp` and `/opt/homebrew/opt/ggml`. Override with `make build WHISPER_PREFIX=… GGML_PREFIX=…` if installed elsewhere.

**Hotkey doesn't work**
Grant Input Monitoring permission in System Settings → Privacy & Security → Input Monitoring, then restart the app.

**Text isn't typed into other apps**
Grant Accessibility permission in System Settings → Privacy & Security → Accessibility, then restart the app.

**App crashes on model load**
Verify the model path is correct (`-model <path>` or `$TALKBACK_MODEL`) and the file is a valid ggml binary (not a partial download). Re-run `make model` to re-download.

**Recording cuts off after 30 seconds**
This is a hard limit imposed by whisper.cpp's `Process()` API. Keep dictation under 30 s per press.

## Architecture overview

```
[Option+Space keydown]  →  audio.Recorder.Start()   →  malgo captures 16 kHz mono float32
[Option+Space keyup]    →  audio.Recorder.Stop()    →  returns []float32 buffer
                        →  transcribe.Transcribe()  →  whisper.cpp Process() (Metal-accelerated)
                        →  inject.Text()            →  CGEventKeyboardSetUnicodeString → cursor
```

The menu bar (fyne.io/systray) runs on the main thread. Whisper inference runs in a goroutine so the UI stays responsive during processing.
