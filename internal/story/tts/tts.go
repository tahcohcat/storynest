package tts

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
}
