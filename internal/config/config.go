package config

import "github.com/spf13/viper"

func setDefaults() {
	viper.SetDefault("tts.type", "auto") // Auto-select best engine
	viper.SetDefault("tts.voice", "default")
	viper.SetDefault("tts.speed", 1.0)
	viper.SetDefault("tts.volume", 0.8)
}
