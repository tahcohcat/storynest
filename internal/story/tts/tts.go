// internal/story/tts/tts.go
package tts

type Config struct {
	Type   string
	Speed  float64
	Volume float64
	Voice  string
}

// Engine interface for text-to-speech functionality
type Engine interface {
	Speak(text string) error
	SetVoice(voice string) error
	SetSpeed(speed float64) error
	SetVolume(volume float64) error
	Stop() error
	Pause() error
	Resume() error
	IsPlaying() bool
	GetAvailableVoices() ([]string, error)
}

// CacheableEngine extends Engine with cache management capabilities
type CacheableEngine interface {
	Engine
	GetCacheStats() (map[string]interface{}, error)
	ClearCache() error
}

// VoiceInfo provides detailed information about available voices
type VoiceInfo struct {
	Name         string `json:"name"`
	LanguageCode string `json:"language_code"`
	Gender       string `json:"gender"`
	Natural      bool   `json:"natural"`
	Description  string `json:"description"`
}

// EnhancedEngine extends Engine with additional capabilities
type EnhancedEngine interface {
	Engine
	GetVoiceInfo() ([]VoiceInfo, error)
	IsPaused() bool
}
