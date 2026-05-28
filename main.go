package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"fyne.io/systray"
	"golang.design/x/hotkey"

	"github.com/cbarker/go-talkback/audio"
	"github.com/cbarker/go-talkback/config"
	"github.com/cbarker/go-talkback/dialog"
	"github.com/cbarker/go-talkback/filler"
	"github.com/cbarker/go-talkback/floatbutton"
	hk "github.com/cbarker/go-talkback/hotkey"
	"github.com/cbarker/go-talkback/inject"
	"github.com/cbarker/go-talkback/launchagent"
	"github.com/cbarker/go-talkback/logger"
	"github.com/cbarker/go-talkback/models"
	"github.com/cbarker/go-talkback/transcribe"
)

const defaultModelEnv = "TALKBACK_MODEL"

// state values for the atomic state machine.
const (
	stateIdle       int32 = 0
	stateListening  int32 = 1
	stateProcessing int32 = 2
)

var (
	recorder    *audio.Recorder
	transcriber *transcribe.Transcriber
	transcMu    sync.Mutex
	appState    atomic.Int32
	appConfig   config.Config
	exitOnce    sync.Once

	// Menu items updated from any goroutine.
	mToggle    *systray.MenuItem
	mModelTop  *systray.MenuItem   // parent "Model: …" item
	modelItems []*systray.MenuItem // one per models.All entry
	mHotkeyTop *systray.MenuItem   // parent "Hotkey: …" item
	mFloat          *systray.MenuItem
	mLaunchAtLogin  *systray.MenuItem
	mStripFillers   *systray.MenuItem
	mQuit           *systray.MenuItem

	floatVisible    bool
	noFillerRemoval bool
	noFillerCLIFlag bool

	// Hotkey relay — the event loop always reads from these two channels.
	// When the active handler changes, only the relay goroutine is swapped.
	keydownRelay = make(chan hotkey.Event, 1)
	keyupRelay   = make(chan hotkey.Event, 1)

	handlerMu     sync.Mutex
	activeHandler *hk.Handler
	relayStop     chan struct{}
)

func main() {
	modelFlag        := flag.String("model", "", "Path to ggml model file (overrides $TALKBACK_MODEL)")
	logLevel         := flag.String("log-level", "warn", "Log level: debug, info, warn, error")
	showVersion      := flag.Bool("version", false, "Print version and exit")
	noFillerFlag     := flag.Bool("no-filler-removal", false, "Disable automatic filler word removal (um, uh, you know, etc.)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `go-talkback — local speech-to-text for macOS

Runs as a menu bar app. Hold Option+Space to record, release to transcribe
and type the result at the current cursor position.

Usage:
  talkback [flags]

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Environment:
  TALKBACK_MODEL   Path to ggml model file. Overridden by -model flag.
                   Default: ~/.local/share/go-talkback/models/ggml-base.en.bin

Models (download with "make model MODEL=<name>"):
  base.en    ~142 MB  fastest, good accuracy for clear speech
  small.en   ~466 MB  better with accents and background noise
  medium.en  ~1.5 GB  near human-level accuracy
  large-v3   ~3.1 GB  best accuracy, supports all languages

macOS permissions required (System Settings → Privacy & Security):
  Microphone       capture audio
  Accessibility    type transcribed text into other apps
  Input Monitoring detect the Option+Space global hotkey

Examples:
  talkback
  talkback -model ~/.local/share/go-talkback/models/ggml-small.en.bin
  talkback -log-level debug
`)
	}

	flag.Parse()
	noFillerCLIFlag = *noFillerFlag

	if *showVersion {
		fmt.Printf("talkback v%s\n", version)
		os.Exit(0)
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cleanup()
		os.Exit(0)
	}()

	if err := logger.Set(*logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "go-talkback: %v\n", err)
		os.Exit(1)
	}

	if *modelFlag != "" {
		os.Setenv(defaultModelEnv, *modelFlag)
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	initIcons()
	appConfig = config.Load()
	if noFillerCLIFlag {
		appConfig.StripFillerWords = false
	}
	noFillerRemoval = !appConfig.StripFillerWords

	// --- First-run model check ---
	// Must happen on the main thread (NSAlert requirement) before any other setup.
	needsDownload := !anyModelInstalled()
	if needsDownload {
		if !dialog.ConfirmModelDownload() {
			systray.Quit()
			return
		}
		// User accepted — download will start after the menu is built.
	}

	var err error

	// --- Load whisper model (skip if we're about to download for the first time) ---
	if !needsDownload {
		modelPath := resolveModelPath()
		// If the user hasn't overridden via flag/env, honour config.Model.
		if modelPath == defaultModelPath() {
			if m := models.Find(appConfig.Model); m != nil {
				modelPath = models.Path(*m)
			}
		}
		slog.Info("loading model", "path", modelPath)

		transcriber, err = transcribe.New(modelPath)
		if err != nil {
			slog.Error("failed to load model", "err", err, "hint", "set TALKBACK_MODEL to the path of a ggml model file")
			os.Exit(1)
		}
		slog.Info("model loaded")
	}

	// --- Init audio recorder ---
	recorder, err = audio.New()
	if err != nil {
		slog.Error("failed to init audio", "err", err)
		os.Exit(1)
	}

	// --- Menu bar setup ---
	systray.SetTitle("")
	systray.SetTooltip("go-talkback v" + version + " — Hold " + appConfig.Hotkey + " to dictate")
	setIdleIcon()

	mToggle = systray.AddMenuItem("Start Listening", "Hold "+appConfig.Hotkey+" or click to toggle")

	// Model submenu
	mModelTop = systray.AddMenuItem("Model: "+appConfig.Model, "Select speech recognition model")
	modelItems = make([]*systray.MenuItem, len(models.All))
	for i, m := range models.All {
		item := mModelTop.AddSubMenuItem(modelLabel(m, appConfig.Model), "")
		if m.Name == appConfig.Model && models.IsDownloaded(m) {
			item.Check()
		}
		modelItems[i] = item
	}

	// Hotkey submenu
	mHotkeyTop = systray.AddMenuItem("Hotkey: "+appConfig.Hotkey, "Configure push-to-talk hotkey")
	presetItems := make([]*systray.MenuItem, len(hk.Presets))
	for i, p := range hk.Presets {
		item := mHotkeyTop.AddSubMenuItem(p.Name, "")
		if p.Name == appConfig.Hotkey {
			item.Check()
		}
		presetItems[i] = item
	}

	mFloat = systray.AddMenuItem("Show Floating Button", "Show a floating push-to-talk button")

	mLaunchAtLogin = systray.AddMenuItem("Launch at Login", "Start Talkback automatically at login")
	if appConfig.LaunchAtLogin {
		mLaunchAtLogin.Check()
	}

	mStripFillers = systray.AddMenuItem("Strip Filler Words", "Remove um, uh, you know, etc. from transcription")
	if appConfig.StripFillerWords {
		mStripFillers.Check()
	}

	systray.AddSeparator()
	mQuit = systray.AddMenuItem("Quit", "Quit go-talkback")

	// --- Register initial hotkey ---
	preset := hk.Find(appConfig.Hotkey)
	if h, err := hk.Register(preset.Mods, preset.Key); err != nil {
		slog.Warn("could not register hotkey, use menu item to toggle listening", "err", err)
	} else {
		handlerMu.Lock()
		activeHandler = h
		startRelay(h)
		handlerMu.Unlock()
	}

	// --- Restore floating button state ---
	if appConfig.ShowFloatButton {
		if appConfig.FloatX != nil && appConfig.FloatY != nil {
			floatbutton.ShowAt(*appConfig.FloatX, *appConfig.FloatY)
		} else {
			floatbutton.Show()
		}
		mFloat.SetTitle("Hide Floating Button")
		floatVisible = true
	}

	// --- First-time download (after menu is fully built) ---
	if needsDownload {
		mToggle.Disable()
		mToggle.SetTitle("Downloading model…")
		go func() {
			base := models.All[0] // base.en — smallest model
			modelItems[0].Disable()
			dialog.ShowProgressWindow()
			dlErr := models.Download(base, func(pct int) {
				dialog.UpdateProgress(pct)
				modelItems[0].SetTitle(fmt.Sprintf("Downloading %s… %d%%", base.Name, pct))
			})
			if dlErr != nil {
				slog.Error("first-run model download failed", "err", dlErr)
				dialog.CloseProgressWindow()
				modelItems[0].SetTitle(modelLabel(base, appConfig.Model))
				modelItems[0].Enable()
				mToggle.SetTitle("Start Listening")
				mToggle.Enable()
				return
			}
			dialog.CloseProgressWithSuccess()
			activateModel(0)
			mToggle.SetTitle("Start Listening")
			mToggle.Enable()
		}()
	}

	// --- First-run launch-at-login offer ---
	if !appConfig.LaunchAtLogin && !launchagent.IsInstalled() {
		go func() {
			if dialog.ConfirmLaunchAtLogin() {
				if err := launchagent.Install(); err != nil {
					slog.Warn("failed to install launch agent", "err", err)
					return
				}
				appConfig.LaunchAtLogin = true
				mLaunchAtLogin.Check()
				if err := config.Save(appConfig); err != nil {
					slog.Warn("failed to save config", "err", err)
				}
			}
		}()
	}

	// Watch each model item for clicks in its own goroutine.
	for i, item := range modelItems {
		i, item := i, item
		go func() {
			for range item.ClickedCh {
				switchModel(i)
			}
		}()
	}

	// Watch each preset item for clicks in its own goroutine.
	for i, item := range presetItems {
		i, item := i, item
		go func() {
			for range item.ClickedCh {
				changeHotkey(hk.Presets[i], presetItems)
			}
		}()
	}

	// --- Main event loop ---
	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				switch appState.Load() {
				case stateIdle:
					startListening()
				case stateListening:
					stopAndTranscribe()
				}

			case <-keydownRelay:
				if appState.Load() == stateIdle {
					startListening()
				}

			case <-keyupRelay:
				if appState.Load() == stateListening {
					stopAndTranscribe()
				}

			case <-floatbutton.Press:
				if appState.Load() == stateIdle {
					startListening()
				}

			case <-floatbutton.Release:
				if appState.Load() == stateListening {
					stopAndTranscribe()
				}

			case pos := <-floatbutton.Moved:
				appConfig.FloatX = &pos.X
				appConfig.FloatY = &pos.Y
				if err := config.Save(appConfig); err != nil {
					slog.Warn("failed to save float position", "err", err)
				}

			case <-mFloat.ClickedCh:
				toggleFloatButton()

			case <-mLaunchAtLogin.ClickedCh:
				toggleLaunchAtLogin()

			case <-mStripFillers.ClickedCh:
				toggleFillerRemoval()

			case <-mQuit.ClickedCh:
				cleanup()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() { cleanup() }

// cleanup releases all resources exactly once. Safe to call from signal handler
// or systray's onExit callback.
func cleanup() {
	exitOnce.Do(func() {
		floatbutton.Hide()

		// Stop relay goroutine.
		handlerMu.Lock()
		if activeHandler != nil {
			activeHandler.Unregister()
			activeHandler = nil
		}
		if relayStop != nil {
			close(relayStop)
			relayStop = nil
		}
		handlerMu.Unlock()

		if recorder != nil {
			recorder.Close()
		}
		if transcriber != nil {
			_ = transcriber.Close()
		}
	})
}

// toggleLaunchAtLogin installs or uninstalls the LaunchAgent, updates the
// menu checkmark, and persists the setting.
func toggleLaunchAtLogin() {
	if appConfig.LaunchAtLogin {
		if err := launchagent.Uninstall(); err != nil {
			slog.Warn("failed to uninstall launch agent", "err", err)
			return
		}
		appConfig.LaunchAtLogin = false
		mLaunchAtLogin.Uncheck()
	} else {
		if err := launchagent.Install(); err != nil {
			slog.Warn("failed to install launch agent", "err", err)
			return
		}
		appConfig.LaunchAtLogin = true
		mLaunchAtLogin.Check()
	}
	if err := config.Save(appConfig); err != nil {
		slog.Warn("failed to save config", "err", err)
	}
}

// toggleFillerRemoval flips the filler-word stripping setting, updates the
// menu checkmark, and persists the setting.
func toggleFillerRemoval() {
	appConfig.StripFillerWords = !appConfig.StripFillerWords
	noFillerRemoval = !appConfig.StripFillerWords
	if appConfig.StripFillerWords {
		mStripFillers.Check()
	} else {
		mStripFillers.Uncheck()
	}
	if err := config.Save(appConfig); err != nil {
		slog.Warn("failed to save config", "err", err)
	}
}

// toggleFloatButton shows or hides the floating button, updates the menu item,
// and persists the new state.
func toggleFloatButton() {
	if floatVisible {
		floatbutton.Hide()
		mFloat.SetTitle("Show Floating Button")
		floatVisible = false
		slog.Info("floating button hidden")
	} else {
		floatbutton.Show()
		mFloat.SetTitle("Hide Floating Button")
		floatVisible = true
		slog.Info("floating button shown")
	}
	appConfig.ShowFloatButton = floatVisible
	if err := config.Save(appConfig); err != nil {
		slog.Warn("failed to save config", "err", err)
	}
}

// startRelay launches a goroutine that forwards events from h into the stable
// relay channels. The caller must hold handlerMu.
func startRelay(h *hk.Handler) {
	if relayStop != nil {
		close(relayStop)
	}
	relayStop = make(chan struct{})
	if h == nil {
		return
	}
	stop := relayStop
	kd := h.Keydown()
	ku := h.Keyup()
	go func() {
		for {
			select {
			case <-stop:
				return
			case e, ok := <-kd:
				if !ok {
					return
				}
				select {
				case keydownRelay <- e:
				case <-stop:
					return
				}
			case e, ok := <-ku:
				if !ok {
					return
				}
				select {
				case keyupRelay <- e:
				case <-stop:
					return
				}
			}
		}
	}()
}

// changeHotkey switches to preset p, updates checkmarks, and persists the choice.
func changeHotkey(p hk.Preset, items []*systray.MenuItem) {
	handlerMu.Lock()
	defer handlerMu.Unlock()

	if activeHandler != nil {
		activeHandler.Unregister()
		activeHandler = nil
	}

	h, err := hk.Register(p.Mods, p.Key)
	if err != nil {
		slog.Warn("failed to register hotkey", "name", p.Name, "err", err)
		startRelay(nil)
		return
	}
	activeHandler = h
	startRelay(h)

	// Update checkmarks.
	for i, item := range items {
		if hk.Presets[i].Name == p.Name {
			item.Check()
		} else {
			item.Uncheck()
		}
	}

	// Update parent label and toggle tooltip.
	mHotkeyTop.SetTitle("Hotkey: " + p.Name)
	mToggle.SetTooltip("Hold " + p.Name + " or click to toggle")
	systray.SetTooltip("go-talkback v" + version + " — Hold " + p.Name + " to dictate")

	// Persist.
	appConfig.Hotkey = p.Name
	if err := config.Save(appConfig); err != nil {
		slog.Warn("failed to save config", "err", err)
	}
	slog.Info("hotkey changed", "hotkey", p.Name)
}

// startListening begins microphone capture.
func startListening() {
	if !appState.CompareAndSwap(stateIdle, stateListening) {
		return
	}
	if err := recorder.Start(); err != nil {
		slog.Error("audio start failed", "err", err)
		appState.Store(stateIdle)
		return
	}
	setListeningIcon()
	floatbutton.SetRecording(true)
	mToggle.SetTitle("Stop Listening")
}

// stopAndTranscribe stops capture, runs whisper, injects text.
func stopAndTranscribe() {
	if !appState.CompareAndSwap(stateListening, stateProcessing) {
		return
	}
	setProcessingIcon()
	mToggle.Disable()

	go func() {
		defer func() {
			appState.Store(stateIdle)
			setIdleIcon()
			floatbutton.SetRecording(false)
			mToggle.SetTitle("Start Listening")
			mToggle.Enable()
		}()

		samples := recorder.Stop()
		if len(samples) == 0 {
			return
		}

		slog.Debug("transcribing", "samples", len(samples))
		transcMu.Lock()
		t := transcriber
		transcMu.Unlock()
		if t == nil {
			slog.Warn("transcriber not ready yet")
			return
		}
		text, err := t.Transcribe(samples)
		if err != nil {
			slog.Error("transcription failed", "err", err)
			return
		}
		slog.Debug("transcription result", "text", text)

		text = strings.TrimSpace(text)
		if !noFillerRemoval {
			text = filler.Strip(text)
		}
		if text != "" {
			inject.Text(text + " ")
		}
	}()
}

// anyModelInstalled reports whether at least one catalog model is present on disk.
func anyModelInstalled() bool {
	for _, m := range models.All {
		if models.IsDownloaded(m) {
			return true
		}
	}
	return false
}

// defaultModelPath returns the hard-coded default model path (base.en).
func defaultModelPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "go-talkback", "models", "ggml-base.en.bin")
}

// resolveModelPath returns the ggml model path from env or default location.
func resolveModelPath() string {
	if p := os.Getenv(defaultModelEnv); p != "" {
		return p
	}
	return defaultModelPath()
}

// modelLabel returns the display string for a model menu item.
// States:
//   - active (downloaded + selected): "✓ base.en (~142 MB) — in use"
//   - downloaded, not active:         "  base.en (~142 MB)"
//   - not downloaded:                 "  base.en (~142 MB) [download]"
func modelLabel(m models.Model, activeName string) string {
	downloaded := models.IsDownloaded(m)
	active := m.Name == activeName && downloaded

	label := m.Name + " (" + m.SizeStr + ")"
	switch {
	case active:
		label = "✓ " + label + " — in use"
	case !downloaded:
		label = "  " + label + " [download]"
	default:
		label = "  " + label
	}
	return label
}

// refreshModelLabels updates all model item titles to reflect current state.
func refreshModelLabels() {
	for i, m := range models.All {
		modelItems[i].SetTitle(modelLabel(m, appConfig.Model))
		if m.Name == appConfig.Model && models.IsDownloaded(m) {
			modelItems[i].Check()
		} else {
			modelItems[i].Uncheck()
		}
	}
	mModelTop.SetTitle("Model: " + appConfig.Model)
}

// switchModel is called when the user clicks a model menu item.
// It guards on stateIdle and initiates download+activation in a goroutine.
func switchModel(idx int) {
	if appState.Load() != stateIdle {
		slog.Info("model switch ignored: not idle")
		return
	}
	m := models.All[idx]

	if !models.IsDownloaded(m) {
		// Start download then activate.
		modelItems[idx].Disable()
		go func() {
			defer modelItems[idx].Enable()
			err := models.Download(m, func(pct int) {
				modelItems[idx].SetTitle(fmt.Sprintf("Downloading %s… %d%%", m.Name, pct))
			})
			if err != nil {
				slog.Error("model download failed", "model", m.Name, "err", err)
				modelItems[idx].SetTitle(modelLabel(m, appConfig.Model))
				return
			}
			activateModel(idx)
		}()
		return
	}

	go activateModel(idx)
}

// activateModel loads the model at idx, swaps the transcriber, and persists the choice.
func activateModel(idx int) {
	m := models.All[idx]
	modelItems[idx].SetTitle("Loading " + m.Name + "…")
	modelItems[idx].Disable()

	newT, err := transcribe.New(models.Path(m))
	if err != nil {
		slog.Error("failed to load model", "model", m.Name, "err", err)
		modelItems[idx].SetTitle(modelLabel(m, appConfig.Model))
		modelItems[idx].Enable()
		return
	}

	transcMu.Lock()
	old := transcriber
	transcriber = newT
	transcMu.Unlock()

	if old != nil {
		_ = old.Close()
	}

	appConfig.Model = m.Name
	if err := config.Save(appConfig); err != nil {
		slog.Warn("failed to save config", "err", err)
	}

	refreshModelLabels()
	for _, item := range modelItems {
		item.Enable()
	}
	slog.Info("model switched", "model", m.Name)
}
