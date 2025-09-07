//go:build windows

// internal/tts/sapi_windows.go
package tts

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
)

// SAPIEngine implements Windows SAPI TTS
type SAPIEngine struct {
	config  Config
	voice   uintptr
	playing bool
	paused  bool
	mutex   sync.RWMutex
}

// Windows API constants
const (
	CLSID_SpVoice = "{96749377-3391-11D2-9EE3-00C04F797396}"
	IID_ISpVoice  = "{6C44DF74-72B9-4992-A1EC-EF996E0422D4}"
)

var (
	ole32                = syscall.NewLazyDLL("ole32.dll")
	procCoInitialize     = ole32.NewProc("CoInitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
)

// NewSAPIEngine creates a new Windows SAPI TTS engine
func NewSAPIEngine(config Config) (*SAPIEngine, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("SAPI engine only supports Windows")
	}

	engine := &SAPIEngine{
		config: config,
	}

	// Initialize COM
	if err := engine.initializeCOM(); err != nil {
		return nil, fmt.Errorf("failed to initialize COM: %w", err)
	}

	return engine, nil
}

func (s *SAPIEngine) initializeCOM() error {
	ret, _, _ := procCoInitialize.Call(0)
	if ret != 0 {
		return fmt.Errorf("CoInitialize failed with code: %d", ret)
	}
	return nil
}

func (s *SAPIEngine) Speak(text string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.playing {
		return fmt.Errorf("already playing")
	}

	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Create ISpVoice COM object
	// 2. Set voice parameters
	// 3. Call ISpVoice::Speak() method
	// 4. Handle async speech events

	s.playing = true

	// Simulate async speech
	go func() {
		defer func() {
			s.mutex.Lock()
			s.playing = false
			s.paused = false
			s.mutex.Unlock()
		}()

		// Here you would integrate with actual SAPI calls
		// For now, we'll use a simple system call to demonstrate
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

	// In real implementation, you would call ISpVoice::Speak() with SPF_PURGEBEFORESPEAK flag
	return nil
}

func (s *SAPIEngine) Pause() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.playing || s.paused {
		return nil
	}

	s.paused = true
	// In real implementation: ISpVoice::Pause()
	return nil
}

func (s *SAPIEngine) Resume() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.paused {
		return nil
	}

	s.paused = false
	// In real implementation: ISpVoice::Resume()
	return nil
}

func (s *SAPIEngine) SetVoice(voice string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.config.Voice = voice
	// In real implementation: ISpVoice::SetVoice()
	return nil
}

func (s *SAPIEngine) SetSpeed(speed float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if speed <= 0 || speed > 3.0 {
		return fmt.Errorf("speed must be between 0.1 and 3.0")
	}

	s.config.Speed = speed
	// In real implementation: ISpVoice::SetRate()
	return nil
}

func (s *SAPIEngine) SetVolume(volume float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if volume < 0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0 and 1.0")
	}

	s.config.Volume = volume
	// In real implementation: ISpVoice::SetVolume()
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
	// In real implementation, you would enumerate available SAPI voices
	// using ISpObjectTokenCategory::EnumTokens()
	return []string{"Microsoft David", "Microsoft Zira", "Microsoft Mark"}, nil
}
