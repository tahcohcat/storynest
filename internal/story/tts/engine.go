package tts

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/viper"
)

type EngineType string

// Updated EngineType constants
const (
	EngineTypeMock          EngineType = "mock"
	EngineTypeESpeak        EngineType = "espeak"
	EngineTypeSAPI          EngineType = "sapi"         // Windows only
	EngineTypeAVFoundation  EngineType = "avfoundation" // macOS only
	EngineTypeGoogleClassic EngineType = "googleclassic"
	EngineTypeAuto          EngineType = "auto" // Automatically choose best for platform
)

func (e EngineType) String() string {
	return string(e)
}

// NewEngine creates a new TTS engine based on the provided config
// todo: integrate this with viper config defaults
func NewEngine(config Config) (Engine, error) {
	// Handle auto-selection
	if config.Type == EngineTypeAuto.String() {
		config.Type = getBestEngineForPlatform().String()
	}

	switch config.Type {
	case EngineTypeMock.String():
		return NewMockTTSEngine(config), nil

	case EngineTypeGoogleClassic.String():
		cachePath := viper.GetString("tts.cache_path")
		return newGoogleClassicTTSEngine(cachePath)

	case EngineTypeESpeak.String():
		return newESpeakEngine(config)

	case EngineTypeSAPI.String():
		if runtime.GOOS != "windows" {
			return nil, fmt.Errorf("SAPI engine only supports Windows")
		}
		return newSAPIEngine(config)

	case EngineTypeAVFoundation.String():
		if runtime.GOOS != "darwin" {
			return nil, fmt.Errorf("AVFoundation engine only supports macOS")
		}
		return newAVFoundationEngine(config)

	default:
		return nil, fmt.Errorf("unsupported TTS engine type: %s", config.Type)
	}
}

func newAVFoundationEngine(config Config) (Engine, error) {
	panic("not implemented on windows")
}

// getBestEngineForPlatform returns the recommended engine for the current platform
func getBestEngineForPlatform() EngineType {

	if hasGoogleCredentials() {
		return EngineTypeGoogleClassic
	}

	switch runtime.GOOS {
	case "windows":
		return EngineTypeSAPI
	case "darwin":
		return EngineTypeAVFoundation
	case "linux":
		return EngineTypeESpeak
	default:
		return EngineTypeESpeak // Cross-platform fallback
	}
}

// GetAvailableEngines returns engines available on the current platform
func GetAvailableEngines() []EngineType {
	engines := []EngineType{EngineTypeMock, EngineTypeESpeak}

	// Add Chirp if Google credentials are available
	if hasGoogleCredentials() {
		engines = append(engines, EngineTypeGoogleClassic)
	}

	switch runtime.GOOS {
	case "windows":
		engines = append(engines, EngineTypeSAPI)
	case "darwin":
		engines = append(engines, EngineTypeAVFoundation)
	}

	return engines
}

// hasGoogleCredentials checks if Google Cloud credentials are available
func hasGoogleCredentials() bool {
	// Check for service account key file
	_, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	return ok
}
