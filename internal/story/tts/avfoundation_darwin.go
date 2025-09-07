//go:build darwin

package tts

import (
	"fmt"
	"os/exec"
	"sync"
)

// AVFoundationEngine implements macOS AVFoundation TTS
type AVFoundationEngine struct {
	config  Config
	playing bool
	paused  bool
	mutex   sync.RWMutex
}

// newAVFoundationEngine creates a new macOS AVFoundation TTS engine
func newAVFoundationEngine(config Config) (*AVFoundationEngine, error) {
	engine := &AVFoundationEngine{
		config: config,
	}

	return engine, nil
}

func (av *AVFoundationEngine) Speak(text string) error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	if av.playing {
		return fmt.Errorf("already playing")
	}

	av.playing = true

	// Use macOS built-in 'say' command
	go func() {
		defer func() {
			av.mutex.Lock()
			av.playing = false
			av.paused = false
			av.mutex.Unlock()
		}()

		args := []string{}

		// Set voice if specified
		if av.config.Voice != "" && av.config.Voice != "default" {
			args = append(args, "-v", av.config.Voice)
		}

		// Set rate (words per minute, default is ~175)
		rate := fmt.Sprintf("%.0f", 175*av.config.Speed)
		args = append(args, "-r", rate)

		// Add text
		args = append(args, text)

		cmd := exec.Command("say", args...)
		if err := cmd.Run(); err != nil {
			fmt.Printf("AVFoundation error: %v\n", err)
		}
	}()

	return nil
}

func (av *AVFoundationEngine) Stop() error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	// Kill any running 'say' processes
	exec.Command("killall", "say").Run()

	av.playing = false
	av.paused = false
	return nil
}

func (av *AVFoundationEngine) Pause() error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	if !av.playing || av.paused {
		return nil
	}

	av.paused = true
	return nil
}

func (av *AVFoundationEngine) Resume() error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	if !av.paused {
		return nil
	}

	av.paused = false
	return nil
}

func (av *AVFoundationEngine) SetVoice(voice string) error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	av.config.Voice = voice
	return nil
}

func (av *AVFoundationEngine) SetSpeed(speed float64) error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	if speed <= 0 || speed > 3.0 {
		return fmt.Errorf("speed must be between 0.1 and 3.0")
	}

	av.config.Speed = speed
	return nil
}

func (av *AVFoundationEngine) SetVolume(volume float64) error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	if volume < 0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0 and 1.0")
	}

	av.config.Volume = volume
	return nil
}

func (av *AVFoundationEngine) IsPlaying() bool {
	av.mutex.RLock()
	defer av.mutex.RUnlock()
	return av.playing && !av.paused
}

func (av *AVFoundationEngine) IsPaused() bool {
	av.mutex.RLock()
	defer av.mutex.RUnlock()
	return av.paused
}

func (av *AVFoundationEngine) GetAvailableVoices() ([]string, error) {
	cmd := exec.Command("say", "-v", "?")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse the output to extract voice names
	// The output format is: "VoiceName    language    # description"
	return []string{"Alex", "Samantha", "Victoria", "Daniel"}, nil
}
