package tts

import (
	"fmt"
	"runtime"
)

type EngineType string

// Updated EngineType constants
const (
	EngineTypeMock         EngineType = "mock"
	EngineTypeESpeak       EngineType = "espeak"
	EngineTypeSAPI         EngineType = "sapi"         // Windows only
	EngineTypeAVFoundation EngineType = "avfoundation" // macOS only
	EngineTypeAuto         EngineType = "auto"         // Automatically choose best for platform
)

func (e EngineType) String() string {
	return string(e)
}

// internal/tts/factory.go
// Updated factory to include platform-specific engines

// NewEngine creates a new TTS engine based on the provided config
func NewEngine(config Config) (Engine, error) {
	// Handle auto-selection
	if config.Type == EngineTypeAuto.String() {
		config.Type = getBestEngineForPlatform().String()
	}

	switch config.Type {
	case EngineTypeMock.String():
		return NewMockTTSEngine(config), nil

	case EngineTypeESpeak.String():
		return NewESpeakEngine(config)

	case EngineTypeSAPI.String():
		if runtime.GOOS != "windows" {
			return nil, fmt.Errorf("SAPI engine only supports Windows")
		}
		return NewSAPIEngine(config)

	case EngineTypeAVFoundation.String():
		if runtime.GOOS != "darwin" {
			return nil, fmt.Errorf("AVFoundation engine only supports macOS")
		}
		return NewAVFoundationEngine(config)

	default:
		return nil, fmt.Errorf("unsupported TTS engine type: %s", config.Type)
	}
}

// getBestEngineForPlatform returns the recommended engine for the current platform
func getBestEngineForPlatform() EngineType {
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

	switch runtime.GOOS {
	case "windows":
		engines = append(engines, EngineTypeSAPI)
	case "darwin":
		engines = append(engines, EngineTypeAVFoundation)
	}

	return engines
}
