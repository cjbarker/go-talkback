// Package launchagent manages a macOS LaunchAgent for launch-at-login.
// It installs/uninstalls a plist to ~/Library/LaunchAgents/ and queries
// whether the agent is currently registered.
package launchagent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const bundleID = "com.cbarker.talkback"

var plistTmpl = template.Must(template.New("launchagent").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cbarker.talkback</string>

    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecPath}}</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <false/>

    <key>StandardOutPath</key>
    <string>{{.LogDir}}/talkback.log</string>

    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/talkback.log</string>
</dict>
</plist>
`))

type plistData struct {
	ExecPath string
	LogDir   string
}

func agentPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", bundleID+".plist"), nil
}

// IsInstalled reports whether the LaunchAgent plist exists on disk.
func IsInstalled() bool {
	p, err := agentPlistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Install writes the LaunchAgent plist using the running binary's path and
// calls launchctl load to register it immediately without requiring logout.
func Install() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("launchagent: resolve executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("launchagent: eval symlinks: %w", err)
	}

	plistPath, err := agentPlistPath()
	if err != nil {
		return fmt.Errorf("launchagent: plist path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("launchagent: mkdir: %w", err)
	}

	home, _ := os.UserHomeDir()
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, plistData{
		ExecPath: execPath,
		LogDir:   filepath.Join(home, "Library", "Logs"),
	}); err != nil {
		return fmt.Errorf("launchagent: render template: %w", err)
	}

	if err := os.WriteFile(plistPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("launchagent: write plist: %w", err)
	}

	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %w: %s", err, out)
	}
	return nil
}

// Uninstall calls launchctl unload and removes the plist.
func Uninstall() error {
	plistPath, err := agentPlistPath()
	if err != nil {
		return err
	}

	// Ignore error if not loaded.
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("launchagent: remove plist: %w", err)
	}
	return nil
}
