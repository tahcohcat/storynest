package tts

import "C"
import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// AVFoundationEngine implements macOS AVFoundation TTS
type AVFoundationEngine struct {
	config      Config
	synthesizer unsafe.Pointer
	mutex       sync.RWMutex
}

// NewAVFoundationEngine creates a new macOS AVFoundation TTS engine
func NewAVFoundationEngine(config Config) (*AVFoundationEngine, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("AVFoundation engine only supports macOS")
	}

	engine := &AVFoundationEngine{
		config: config,
	}

	engine.synthesizer = C.createSynthesizer()
	if engine.synthesizer == nil {
		return nil, fmt.Errorf("failed to create AVSpeechSynthesizer")
	}

	// Set finalizer to clean up native resources
	runtime.SetFinalizer(engine, (*AVFoundationEngine).cleanup)

	return engine, nil
}

func (av *AVFoundationEngine) cleanup() {
	if av.synthesizer != nil {
		C.releaseSynthesizer(av.synthesizer)
		av.synthesizer = nil
	}
}

func (av *AVFoundationEngine) Speak(text string) error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	if av.IsPlaying() {
		return fmt.Errorf("already playing")
	}

	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	var cVoice *C.char
	if av.config.Voice != "" && av.config.Voice != "default" {
		cVoice = C.CString(av.config.Voice)
		defer C.free(unsafe.Pointer(cVoice))
	}

	// AVFoundation rate range is 0.0 to 1.0, where 0.5 is normal
	rate := C.float(av.config.Speed * 0.5)
	volume := C.float(av.config.Volume)

	result := C.speakText(av.synthesizer, cText, cVoice, rate, volume)
	if result != 0 {
		return fmt.Errorf("failed to start speech synthesis")
	}

	return nil
}

func (av *AVFoundationEngine) Stop() error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	C.stopSpeaking(av.synthesizer)
	return nil
}

func (av *AVFoundationEngine) Pause() error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	C.pauseSpeaking(av.synthesizer)
	return nil
}

func (av *AVFoundationEngine) Resume() error {
	av.mutex.Lock()
	defer av.mutex.Unlock()

	C.resumeSpeaking(av.synthesizer)
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

	return int(C.isSpeaking(av.synthesizer)) == 1
}

func (av *AVFoundationEngine) IsPaused() bool {
	av.mutex.RLock()
	defer av.mutex.RUnlock()

	return int(C.isPaused(av.synthesizer)) == 1
}

func (av *AVFoundationEngine) GetAvailableVoices() ([]string, error) {
	// In a full implementation, you would enumerate available voices
	// using [AVSpeechSynthesisVoice speechVoices]
	return []string{"com.apple.ttsbundle.Samantha-compact", "com.apple.ttsbundle.Alex-compact"}, nil
}
