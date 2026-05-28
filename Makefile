WHISPER_PREFIX ?= /opt/homebrew/opt/whisper-cpp
GGML_PREFIX    ?= /opt/homebrew/opt/ggml

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# CGO_CFLAGS: add whisper.h and ggml.h to include path.
# CGO_LDFLAGS: add the library search path and rpath so the linker finds
# libwhisper + libggml* at build and run time.
# The whisper.cpp Go bindings (v1.8+) link against split ggml backends
# (libggml-cpu, libggml-metal, libggml-blas) that don't exist in the
# Homebrew ggml formula (which uses a monolithic libggml). The 'stubs'
# target installs empty dylibs to $(GGML_PREFIX)/lib to satisfy the
# linker; actual symbols arrive via libggml loaded transitively.

CGO_CFLAGS  = -I$(WHISPER_PREFIX)/include -I$(GGML_PREFIX)/include
CGO_LDFLAGS = -L$(WHISPER_PREFIX)/lib -rpath $(WHISPER_PREFIX)/lib -L$(GGML_PREFIX)/lib -rpath $(GGML_PREFIX)/lib

export CGO_ENABLED = 1
export CGO_CFLAGS
export CGO_LDFLAGS

.PHONY: build run test clean model icns app dmg stubs version

# Install stub dylibs for missing ggml backends into the Homebrew ggml lib dir.
# whisper.cpp Go bindings (v1.8+) reference libggml-cpu / libggml-metal /
# libggml-blas, but the Homebrew ggml formula ships only the monolithic libggml.
# These empty stubs satisfy the linker and dynamic loader; actual symbols arrive
# via libggml loaded transitively through libwhisper.
stubs:
	@for lib in libggml-cpu libggml-metal libggml-blas; do \
	  target="$(GGML_PREFIX)/lib/$${lib}.dylib"; \
	  if [ ! -f "$$target" ]; then \
	    echo "Installing stub: $$target"; \
	    echo "" | cc -x c - -dynamiclib \
	      -install_name "@rpath/$${lib}.dylib" \
	      -o "$$target"; \
	  fi; \
	done

build: stubs
	go build -trimpath -ldflags "$(LDFLAGS)" -o talkback .
	# Re-sign after build: the Go linker applies -ldflags -X substitutions after
	# computing the ad-hoc signature, leaving the signature invalid on macOS 15+.
	# codesign --force replaces the stale linker signature with a fresh one.
	codesign --force --sign - talkback

version:
	@echo $(VERSION)

run: build
	./talkback

# Run all tests. The transcribe integration test requires a model; it is
# skipped automatically when none is found.
test:
	go test ./... -timeout 120s

# Download the base.en model (smallest English-only model, ~142 MB).
# Set MODEL to base, small, medium, or large-v3.
MODEL ?= base.en
MODEL_DIR = $(HOME)/.local/share/go-talkback/models

model:
	mkdir -p $(MODEL_DIR)
	curl -L "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-$(MODEL).bin" \
	     -o "$(MODEL_DIR)/ggml-$(MODEL).bin"
	@echo "Model saved to $(MODEL_DIR)/ggml-$(MODEL).bin"
	@echo "Set TALKBACK_MODEL=$(MODEL_DIR)/ggml-$(MODEL).bin before running."

clean:
	rm -f talkback
	rm -rf build dist

# ── Packaging ──────────────────────────────────────────────────────────────────

SRC_PNG  = assets/icons/walkie-talkie.png
ICONSET  = build/AppIcon.iconset
ICNS     = build/AppIcon.icns
APP_DIR  = dist/Talkback.app
DMG_BG   = build/dmg_bg.png
DMG_BG2X = build/dmg_bg@2x.png
DMG_RW   = build/Talkback-rw.dmg
DMG      = dist/Talkback-$(VERSION).dmg

# --- App icon (.icns) ---
icns: $(ICNS)

$(ICNS): $(SRC_PNG)
	mkdir -p $(ICONSET)
	sips -z 16   16   $(SRC_PNG) --out $(ICONSET)/icon_16x16.png      > /dev/null
	sips -z 32   32   $(SRC_PNG) --out $(ICONSET)/icon_16x16@2x.png  > /dev/null
	sips -z 32   32   $(SRC_PNG) --out $(ICONSET)/icon_32x32.png     > /dev/null
	sips -z 64   64   $(SRC_PNG) --out $(ICONSET)/icon_32x32@2x.png  > /dev/null
	sips -z 128  128  $(SRC_PNG) --out $(ICONSET)/icon_128x128.png   > /dev/null
	sips -z 256  256  $(SRC_PNG) --out $(ICONSET)/icon_128x128@2x.png > /dev/null
	sips -z 256  256  $(SRC_PNG) --out $(ICONSET)/icon_256x256.png   > /dev/null
	sips -z 512  512  $(SRC_PNG) --out $(ICONSET)/icon_256x256@2x.png > /dev/null
	sips -z 512  512  $(SRC_PNG) --out $(ICONSET)/icon_512x512.png   > /dev/null
	sips -z 1024 1024 $(SRC_PNG) --out $(ICONSET)/icon_512x512@2x.png > /dev/null
	iconutil -c icns $(ICONSET) -o $(ICNS)
	@echo "Icon: $(ICNS)"

# --- App bundle (.app) ---
app: build icns
	mkdir -p $(APP_DIR)/Contents/MacOS $(APP_DIR)/Contents/Resources
	cp talkback $(APP_DIR)/Contents/MacOS/talkback
	cp $(ICNS)  $(APP_DIR)/Contents/Resources/AppIcon.icns
	sed 's/VERSION_PLACEHOLDER/$(VERSION)/g' \
	    assets/Info.plist > $(APP_DIR)/Contents/Info.plist
	codesign --force --deep --sign - \
	    --entitlements assets/entitlements.plist \
	    $(APP_DIR)
	@echo "App bundle: $(APP_DIR)"

# --- DMG background image ---
$(DMG_BG) $(DMG_BG2X): assets/make_dmg_bg.swift $(SRC_PNG)
	mkdir -p build
	swift assets/make_dmg_bg.swift $(SRC_PNG) $(DMG_BG) $(DMG_BG2X)

# --- Distributable DMG ---
dmg: app $(DMG_BG) $(DMG_BG2X)
	mkdir -p dist
	rm -f $(DMG_RW) $(DMG)
	# Detach any leftover mount from a prior failed run.
	hdiutil detach /Volumes/Talkback 2>/dev/null || true
	# Create a scratch read-write HFS+ DMG (80 MB is enough for the binary + assets).
	hdiutil create -megabytes 80 -fs HFS+ -volname "Talkback" \
	    -layout SPUD $(DMG_RW)
	hdiutil attach $(DMG_RW) -mountpoint /Volumes/Talkback
	# Copy app bundle and create Applications alias.
	cp -R $(APP_DIR) /Volumes/Talkback/
	ln -s /Applications /Volumes/Talkback/Applications
	# Copy background images into hidden .background/ directory.
	mkdir -p /Volumes/Talkback/.background
	cp $(DMG_BG)   /Volumes/Talkback/.background/background.png
	cp $(DMG_BG2X) /Volumes/Talkback/.background/background@2x.png
	# Configure Finder window layout (icon positions, background, window size).
	osascript assets/dmg_layout.applescript
	sync
	hdiutil detach /Volumes/Talkback
	# Convert to compressed read-only DMG.
	hdiutil convert $(DMG_RW) -format UDZO -imagekey zlib-level=9 -o $(DMG)
	rm -f $(DMG_RW)
	@echo "Ready: $(DMG)"
