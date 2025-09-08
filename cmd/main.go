package main

import (
	"fmt"
	"os"
	"os/signal"
	"storynest/internal/cli/scheme/colours"
	"storynest/internal/config"
	"storynest/internal/story/nest"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {

	config.SetDefaults()

	app := nest.NewStoryNest()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		app.Cancel()
		app.Tts.Stop()
		fmt.Println("\n" + colours.Warning.Sprint("👋 Goodbye! Sweet dreams! 🌙"))
		os.Exit(0)
	}()

	rootCmd := &cobra.Command{
		Use:   "storynest",
		Short: "🏠 A cozy home for bedtime stories",
		Long: `
┌─────────────────────────────────────┐
│  📚 Welcome to StoryNest! 🏠       │
│  A cozy home for bedtime stories    │
│  Read aloud for kids 👶✨          │
└─────────────────────────────────────┘

StoryNest helps you discover and listen to wonderful children's stories
from public libraries around the world. Perfect for bedtime! 🌙
		`,
		Run: func(cmd *cobra.Command, args []string) {
			app.ShowWelcome()
		},
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "📋 List available stories",
		Long:  "Display all available stories from connected libraries",
		Run:   app.ListStories,
	}

	// Random command
	randomCmd := &cobra.Command{
		Use:   "random",
		Short: "🎲 Read a random story",
		Long:  "Select and read a random story from the available collection",
		Run:   app.ReadRandomStory,
	}

	// Read command
	readCmd := &cobra.Command{
		Use:   "read [story-id]",
		Short: "📖 Read a specific story",
		Long:  "Read a story by its ID or select from a list",
		Run:   app.ReadStory,
	}

	// Libraries command
	librariesCmd := &cobra.Command{
		Use:   "libraries",
		Short: "🏛️ Manage story libraries",
		Long:  "Add, remove, or list connected story libraries",
		Run:   app.ManageLibraries,
	}

	// Settings command
	settingsCmd := &cobra.Command{
		Use:   "settings",
		Short: "⚙️ Configure TTS settings",
		Long:  "Adjust voice, speed, and volume settings",
		Run:   app.ConfigureSettings,
	}

	// Add flags
	listCmd.Flags().StringP("genre", "g", "", "Filter by genre")
	listCmd.Flags().StringP("age", "a", "", "Filter by age group")
	readCmd.Flags().StringP("voice", "v", "", "Optional voice to use for reading. See voice list for options")
	readCmd.Flags().BoolP("interactive", "i", false, "Interactive story selection")

	rootCmd.AddCommand(listCmd, randomCmd, readCmd, librariesCmd, settingsCmd)

	// Add Gutenberg commands
	app.AddGutenbergCommands(rootCmd)

	// Load sample data including Gutenberg
	app.LoadSampleLibrariesWithGutenberg()

	if err := rootCmd.Execute(); err != nil {
		colours.Error.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	rootCmd.Run(randomCmd, []string{})
}

// Configuration management with Viper
func init() {
	viper.SetConfigName("storynest")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.storynest")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("tts.voice", "default")
	viper.SetDefault("tts.speed", 1.0)
	viper.SetDefault("tts.volume", 1.0)
	viper.SetDefault("libraries", []string{})

	viper.ReadInConfig()
}
