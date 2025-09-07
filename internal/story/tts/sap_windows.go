//go:build windows

package tts

import (
	"fmt"
	"os/exec"
	"sync"
)

// SAPIEngine implements Windows SAPI TTS
type SAPIEngine struct {
	config  Config
	voice   uintptr
	playing bool
	paused  bool
	mutex   sync.RWMutex
}

// newSAPIEngine creates a new Windows SAPI TTS engine
func newSAPIEngine(config Config) (*SAPIEngine, error) {
	engine := &SAPIEngine{
		config: config,
	}

	return engine, nil
}

func (s *SAPIEngine) Speak(text string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.playing {
		return fmt.Errorf("already playing")
	}

	s.playing = true

	// Simulate async speech
	go func() {
		defer func() {
			s.mutex.Lock()
			s.playing = false
			s.paused = false
			s.mutex.Unlock()
		}()

		// Use PowerShell to access Windows Speech API
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf(`Add-Type -AssemblyName System.Speech; 
			$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; 
			$synth.Rate = %d; 
			$synth.Volume = %d; 
			$synth.Speak("%s")`,
				int(s.config.Speed*10)-10, // Convert to SAPI range (-10 to 10)
				int(s.config.Volume*100),  // Convert to SAPI range (0 to 100)
				text))

		if err := cmd.Run(); err != nil {
			fmt.Printf("SAPI error: %v\n", err)
		}
	}()

	return nil
}

func (s *SAPIEngine) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.playing = false
	s.paused = false
	return nil
}

func (s *SAPIEngine) Pause() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.playing || s.paused {
		return nil
	}

	s.paused = true
	return nil
}

func (s *SAPIEngine) Resume() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.paused {
		return nil
	}

	s.paused = false
	return nil
}

func (s *SAPIEngine) SetVoice(voice string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.config.Voice = voice
	return nil
}

func (s *SAPIEngine) SetSpeed(speed float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if speed <= 0 || speed > 3.0 {
		return fmt.Errorf("speed must be between 0.1 and 3.0")
	}

	s.config.Speed = speed
	return nil
}

func (s *SAPIEngine) SetVolume(volume float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if volume < 0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0 and 1.0")
	}

	s.config.Volume = volume
	return nil
}

func (s *SAPIEngine) IsPlaying() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.playing && !s.paused
}

func (s *SAPIEngine) IsPaused() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.paused
}

func (s *SAPIEngine) GetAvailableVoices() ([]string, error) {
	return []string{"Microsoft David", "Microsoft Zira", "Microsoft Mark"}, nil
}
