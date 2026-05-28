package logger_test

import (
	"testing"

	"github.com/cbarker/go-talkback/logger"
)

func TestSet_ValidLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error"} {
		if err := logger.Set(level); err != nil {
			t.Errorf("Set(%q) returned unexpected error: %v", level, err)
		}
	}
}

func TestSet_CaseInsensitive(t *testing.T) {
	for _, level := range []string{"DEBUG", "INFO", "WARN", "WARNING", "ERROR", "Warn", "Info"} {
		if err := logger.Set(level); err != nil {
			t.Errorf("Set(%q) returned unexpected error: %v", level, err)
		}
	}
}

func TestSet_InvalidLevel_ReturnsError(t *testing.T) {
	for _, bad := range []string{"", "verbose", "trace", "fatal", "off"} {
		if err := logger.Set(bad); err == nil {
			t.Errorf("Set(%q) expected error, got nil", bad)
		}
	}
}
