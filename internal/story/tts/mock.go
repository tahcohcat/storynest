package tts

import (
	"strings"
	"time"

	"github.com/fatih/color"
)

// MockTTSEngine - placeholder implementation
type MockTTSEngine struct {
	playing bool
	paused  bool
	speed   float64
	volume  float64
	voice   string
}

func (m *MockTTSEngine) SetBookContext(provider, bookID string) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTTSEngine) GetAvailableVoices() ([]string, error) {
	return []string{"mock-voice"}, nil
}

func NewMockTTSEngine(c Config) *MockTTSEngine {
	return &MockTTSEngine{
		speed:  c.Speed,
		volume: c.Volume,
		voice:  "default",
	}
}

func (m *MockTTSEngine) Speak(text string) error {
	m.playing = true
	m.paused = false

	// Simulate reading time based on text length
	words := len(strings.Fields(text))
	duration := time.Duration(float64(words)/150.0*m.speed) * time.Minute

	color.Yellow("🔊 Reading aloud... (simulated for %v)", duration)

	// In a real implementation, you would integrate with:
	// - github.com/hajimehoshi/oto for audio output
	// - A TTS library like eSpeak, Festival, or cloud TTS APIs

	time.Sleep(2 * time.Second) // Simulate some reading time
	m.playing = false
	return nil
}

func (m *MockTTSEngine) SetVoice(voice string) error {
	m.voice = voice
	return nil
}

func (m *MockTTSEngine) SetSpeed(speed float64) error {
	m.speed = speed
	return nil
}

func (m *MockTTSEngine) SetVolume(volume float64) error {
	m.volume = volume
	return nil
}

func (m *MockTTSEngine) Stop() error {
	m.playing = false
	m.paused = false
	return nil
}

func (m *MockTTSEngine) Pause() error {
	if m.playing {
		m.paused = true
	}
	return nil
}

func (m *MockTTSEngine) Resume() error {
	if m.paused {
		m.paused = false
	}
	return nil
}

func (m *MockTTSEngine) IsPlaying() bool {
	return m.playing && !m.paused
}
