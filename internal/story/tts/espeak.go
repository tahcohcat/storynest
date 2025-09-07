// internal/tts/espeak.go
// Cross-platform eSpeak implementation
package tts

import "C"

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// ESpeakEngine implements TTS using eSpeak/eSpeak-NG
type ESpeakEngine struct {
	config  Config
	cmd     *exec.Cmd
	playing bool
	paused  bool
	mutex   sync.RWMutex
}

// NewESpeakEngine creates a new eSpeak TTS engine
func NewESpeakEngine(config Config) (*ESpeakEngine, error) {
	// Check if eSpeak is available
	espeakPath, err := findESpeakExecutable()
	if err != nil {
		return nil, fmt.Errorf("eSpeak not found: %w", err)
	}

	engine := &ESpeakEngine{
		config: config,
	}

	// Test the installation
	if err := engine.testInstallation(espeakPath); err != nil {
		return nil, fmt.Errorf("eSpeak test failed: %w", err)
	}

	return engine, nil
}

func findESpeakExecutable() (string, error) {
	// Try different possible eSpeak executables
	candidates := []string{"espeak-ng", "espeak"}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("eSpeak executable not found in PATH")
}

func (e *ESpeakEngine) testInstallation(espeakPath string) error {
	cmd := exec.Command(espeakPath, "--version")
	return cmd.Run()
}

func (e *ESpeakEngine) Speak(text string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.playing {
		return fmt.Errorf("already playing")
	}

	espeakPath, err := findESpeakExecutable()
	if err != nil {
		return err
	}

	// Build eSpeak command arguments
	args := []string{}

	// Set voice
	if e.config.Voice != "" && e.config.Voice != "default" {
		args = append(args, "-v", e.config.Voice)
	}

	// Set speed (words per minute, default is 175)
	speed := int(175 * e.config.Speed)
	args = append(args, "-s", strconv.Itoa(speed))

	// Set volume (0-200, default is 100)
	volume := int(100 * e.config.Volume)
	args = append(args, "-a", strconv.Itoa(volume))

	// Add text
	args = append(args, text)

	// Create command
	e.cmd = exec.Command(espeakPath, args...)
	e.playing = true
	e.paused = false

	// Start speaking in background
	go func() {
		defer func() {
			e.mutex.Lock()
			e.playing = false
			e.paused = false
			e.mutex.Unlock()
		}()

		if err := e.cmd.Run(); err != nil {
			// Check if it was intentionally stopped
			if e.cmd.ProcessState != nil && e.cmd.ProcessState.Exited() {
				return
			}
			fmt.Printf("eSpeak error: %v\n", err)
		}
	}()

	return nil
}

func (e *ESpeakEngine) Stop() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.cmd != nil && e.cmd.Process != nil {
		if err := e.cmd.Process.Kill(); err != nil {
			return err
		}
	}

	e.playing = false
	e.paused = false
	return nil
}

func (e *ESpeakEngine) Pause() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.playing || e.paused {
		return nil
	}

	if e.cmd != nil && e.cmd.Process != nil {
		// eSpeak doesn't support pause/resume, so we stop and remember position
		// This is a limitation of the eSpeak command-line interface
		if err := e.cmd.Process.Signal(syscall.SIGSTOP); err != nil {
			return err
		}
		e.paused = true
	}

	return nil
}

func (e *ESpeakEngine) Resume() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.paused {
		return nil
	}

	if e.cmd != nil && e.cmd.Process != nil {
		if err := e.cmd.Process.Signal(syscall.SIGCONT); err != nil {
			return err
		}
		e.paused = false
	}

	return nil
}

func (e *ESpeakEngine) SetVoice(voice string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Validate voice exists
	voices, err := e.GetAvailableVoices()
	if err != nil {
		return err
	}

	voiceFound := false
	for _, v := range voices {
		if v == voice {
			voiceFound = true
			break
		}
	}

	if !voiceFound {
		return fmt.Errorf("voice '%s' not available", voice)
	}

	e.config.Voice = voice
	return nil
}

func (e *ESpeakEngine) SetSpeed(speed float64) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if speed <= 0 || speed > 3.0 {
		return fmt.Errorf("speed must be between 0.1 and 3.0")
	}

	e.config.Speed = speed
	return nil
}

func (e *ESpeakEngine) SetVolume(volume float64) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if volume < 0 || volume > 2.0 {
		return fmt.Errorf("volume must be between 0 and 2.0")
	}

	e.config.Volume = volume
	return nil
}

func (e *ESpeakEngine) IsPlaying() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.playing && !e.paused
}

func (e *ESpeakEngine) IsPaused() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.paused
}

func (e *ESpeakEngine) GetAvailableVoices() ([]string, error) {
	espeakPath, err := findESpeakExecutable()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(espeakPath, "--voices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseESpeakVoices(string(output)), nil
}

func parseESpeakVoices(output string) []string {
	lines := strings.Split(output, "\n")
	voices := make([]string, 0)

	for i, line := range lines {
		// Skip header line
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse voice line: Pty Language Age/Gender VoiceName          File          Other Languages
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			voices = append(voices, fields[3])
		}
	}

	return voices
}

// ===================================

// ===================================

// Installation instructions and dependencies

/*
To use these TTS implementations:

## eSpeak (Cross-platform)

### Ubuntu/Debian:
sudo apt-get install espeak espeak-data

### macOS (via Homebrew):
brew install espeak

### Windows:
Download from: http://espeak.sourceforge.net/download.html

## Windows SAPI
Already available on Windows systems.
No additional installation required.

## macOS AVFoundation
Already available on macOS systems.
Requires Xcode or Command Line Tools for compilation.

## Go build tags usage:
go build -tags "windows" ./...  # For Windows SAPI
go build -tags "darwin" ./...   # For macOS AVFoundation
go build ./...                  # Default build (includes eSpeak and mock)

## Example configuration:
tts:
  type: auto        # Auto-select best engine for platform
  voice: default    # Use default system voice
  speed: 1.0        # Normal speed
  volume: 1.0       # Full volume

## Cloud TTS Integration Example:
For production applications, consider cloud TTS services:
- Google Cloud Text-to-Speech API
- Amazon Polly
- Microsoft Azure Cognitive Services Speech
- IBM Watson Text to Speech

These provide higher quality voices and more features.
*/
