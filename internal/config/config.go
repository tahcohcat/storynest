package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func SetDefaults() {
	viper.SetDefault("tts.type", "auto") // Auto-select best engine
	viper.SetDefault("tts.voice", "default")
	viper.SetDefault("tts.speed", 1.0)
	viper.SetDefault("tts.volume", 0.8)

	// Try to use Chirp if Google credentials are available, otherwise auto-select
	if hasGoogleCredentials() {
		viper.SetDefault("tts.type", "chirp")
		viper.SetDefault("tts.voice", "en-US-Journey-F") // Child-friendly default
	} else {
		viper.SetDefault("tts.type", "auto") // Auto-select best engine
		viper.SetDefault("tts.voice", "default")
	}

	viper.SetDefault("tts.cache_enabled", true)
	viper.SetDefault("tts.cache_path", "C:\\Users\\tahcoh\\AppData\\Local\\storynest")
	viper.SetDefault("tts.cache_max_size_mb", 500) // 500MB cache limit
}

func hasGoogleCredentials() bool {
	// Same implementation as in engine.go
	keyPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if keyPath != "" {
		if _, err := os.Stat(keyPath); err == nil {
			return true
		}
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		defaultPath := filepath.Join(homeDir, ".config", "storynest", "google-credentials.json")
		if _, err := os.Stat(defaultPath); err == nil {
			return true
		}
	}

	if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		return true
	}

	return false
}
