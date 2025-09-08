package nest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"storynest/internal/cli/scheme/colours"
	"storynest/internal/domain/library"
	"storynest/internal/domain/library/guten"
	"storynest/internal/domain/story"
	"storynest/internal/story/tts"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// StoryNest main application structure
type StoryNest struct {
	onlineLibrary library.CachedOnlineLibrary

	libraries []library.StoryLibrary
	Tts       tts.Engine
	ctx       context.Context
	Cancel    context.CancelFunc
}

func NewStoryNest() *StoryNest {

	engine, err := tts.NewEngine(tts.Config{
		Type:   tts.EngineTypeAuto.String(),
		Speed:  1.0,
		Volume: 1.0,
		Voice:  "default",
	})

	if err != nil {
		logrus.WithError(err).Fatal("failed to create tts engine")
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &StoryNest{
		onlineLibrary: guten.NewGutenbergCache("./cache", 4*24*time.Hour),

		// todo: remove once we have a guten
		libraries: []library.StoryLibrary{},
		Tts:       engine,
		ctx:       ctx,
		Cancel:    cancel,
	}
}

func (sn *StoryNest) SetVoice(voice string) error {
	logrus.WithField("voice", voice).Info("set voice")
	return sn.Tts.SetVoice(voice)
}

func (sn *StoryNest) ShowWelcome() {
	fmt.Println()
	colours.Title.Println("🌟 Welcome to StoryNest! 🌟")
	fmt.Println()
	colours.Info.Println("📚 Available commands:")
	fmt.Println("  • storynest list      - Browse available stories")
	fmt.Println("  • storynest random    - Get a surprise story")
	fmt.Println("  • storynest read      - Choose a specific story")
	fmt.Println("  • storynest libraries - Manage story sources")
	fmt.Println("  • storynest settings  - Configure voice settings")
	fmt.Println()
	colours.Prompt.Println("✨ Ready for a magical story adventure? ✨")
}

func (sn *StoryNest) LoadSampleLibraries() {
	// Sample library with demo stories
	sampleLibrary := library.StoryLibrary{
		Name: "Classic Tales Collection",
		URL:  "https://api.example.com/classic-tales",
		Stories: []story.Item{
			{
				ID:          "goldilocks",
				Title:       "Goldilocks and the Three Bears",
				Author:      "Traditional",
				Content:     "Once upon a time, there was a little girl named Goldilocks...",
				AgeGroup:    "3-6 years",
				Genre:       "Fairy Tale",
				Duration:    "5 minutes",
				Description: "A classic tale about curiosity and consequences",
			},
			{
				ID:          "three-pigs",
				Title:       "The Three Little Pigs",
				Author:      "Traditional",
				Content:     "Once there were three little pigs who left home to build houses...",
				AgeGroup:    "3-7 years",
				Genre:       "Fairy Tale",
				Duration:    "6 minutes",
				Description: "A story about hard work and perseverance",
			},
			{
				ID:          "red-riding-hood",
				Title:       "Little Red Riding Hood",
				Author:      "Traditional",
				Content:     "Little Red Riding Hood lived with her mother in a cottage...",
				AgeGroup:    "4-8 years",
				Genre:       "Fairy Tale",
				Duration:    "7 minutes",
				Description: "A tale about being careful with strangers",
			},
		},
	}

	modernLibrary := library.StoryLibrary{
		Name: "Modern Adventures",
		URL:  "https://api.example.com/modern-stories",
		Stories: []story.Item{
			{
				ID:          "space-cat",
				Title:       "Captain Whiskers' Space Adventure",
				Author:      "Luna Starweaver",
				Content:     "Captain Whiskers was no ordinary cat. He had his own spaceship...",
				AgeGroup:    "5-9 years",
				Genre:       "Science Fiction",
				Duration:    "8 minutes",
				Description: "A brave cat explores the galaxy",
			},
			{
				ID:          "magic-garden",
				Title:       "The Secret Magic Garden",
				Author:      "Rose Greenthumb",
				Content:     "Behind the old oak tree, Emma discovered a hidden gate...",
				AgeGroup:    "4-8 years",
				Genre:       "Fantasy",
				Duration:    "10 minutes",
				Description: "A girl discovers a magical world in her backyard",
			},
		},
	}

	sn.libraries = append(sn.libraries, sampleLibrary, modernLibrary)
}

func (sn *StoryNest) ListStories(cmd *cobra.Command, args []string) {
	genre, _ := cmd.Flags().GetString("genre")
	ageGroup, _ := cmd.Flags().GetString("age")

	fmt.Println()
	colours.Title.Println("📚 Available Stories 📚")
	fmt.Println()

	onlineLibrary, err := sn.onlineLibrary.GetLibrary()
	if err != nil {
		colours.Error.Println(err)
	}

	// todo:
	count := 0
	for _, story := range onlineLibrary.Stories {
		count++
		fmt.Printf("  %d. ", count)
		colours.Title.Printf("%s", story.Title)
		fmt.Printf(" by ")
		colours.Author.Printf("%s", story.Author)
		fmt.Printf("\n     🎯 Age: %s | 🎭 Genre: %s | ⏱️ Duration: %s\n",
			story.AgeGroup, story.Genre, story.Duration)
		fmt.Printf("     💡 %s\n", story.Description)
		colours.Info.Printf("     ID: %s\n", story.ID)
		fmt.Println()
	}

	for _, lib := range sn.libraries {
		colours.Info.Printf("📖 From %s:\n", lib.Name)

		for _, story := range lib.Stories {
			// Apply filters
			if genre != "" && !strings.Contains(strings.ToLower(story.Genre), strings.ToLower(genre)) {
				continue
			}
			if ageGroup != "" && !strings.Contains(strings.ToLower(story.AgeGroup), strings.ToLower(ageGroup)) {
				continue
			}

			count++
			fmt.Printf("  %d. ", count)
			colours.Title.Printf("%s", story.Title)
			fmt.Printf(" by ")
			colours.Author.Printf("%s", story.Author)
			fmt.Printf("\n     🎯 Age: %s | 🎭 Genre: %s | ⏱️ Duration: %s\n",
				story.AgeGroup, story.Genre, story.Duration)
			fmt.Printf("     💡 %s\n", story.Description)
			colours.Info.Printf("     ID: %s\n", story.ID)
			fmt.Println()
		}
	}

	if count == 0 {
		colours.Warning.Println("🔍 No stories found matching your criteria.")
	} else {
		colours.Success.Printf("✨ Found %d wonderful stories! ✨\n", count)
	}
}

func (sn *StoryNest) ReadRandomStory(cmd *cobra.Command, args []string) {
	stories := sn.getAllStories()
	if len(stories) == 0 {
		colours.Error.Println("❌ No stories available!")
		return
	}

	rand.Seed(time.Now().UnixNano())
	randomStory := stories[rand.Intn(len(stories))]

	fmt.Println()
	colours.Prompt.Println("🎲 Random Story Selection! 🎲")
	fmt.Println()

	voice, _ := cmd.Flags().GetString("voice")
	if err := sn.Tts.SetVoice(voice); err != nil {
		colours.Error.Println("❌ voice '%s' not found on current tts engine!\n")
	}

	sn.displayAndReadStory(randomStory)
}

func (sn *StoryNest) ReadStory(cmd *cobra.Command, args []string) {
	interactive, _ := cmd.Flags().GetBool("interactive")

	voice, _ := cmd.Flags().GetString("voice")
	if err := sn.Tts.SetVoice(voice); err != nil {
		colours.Error.Println("❌ voice '%s' not found on current tts engine!\n")
	}

	if len(args) == 0 || interactive {
		sn.interactiveStorySelection()
		return
	}

	storyID := args[0]
	story := sn.findStoryByID(storyID)

	if story == nil {
		colours.Error.Printf("❌ Story with ID '%s' not found!\n", storyID)
		return
	}

	sn.displayAndReadStory(*story)
}

func (sn *StoryNest) interactiveStorySelection() {
	stories := sn.getAllStories()
	if len(stories) == 0 {
		colours.Error.Println("❌ No stories available!")
		return
	}

	fmt.Println()
	colours.Title.Println("📚 Choose Your Story Adventure! 📚")
	fmt.Println()

	for i, story := range stories {
		fmt.Printf("%d. ", i+1)
		colours.Title.Printf("%s", story.Title)
		fmt.Printf(" by ")
		colours.Author.Printf("%s", story.Author)
		fmt.Printf(" (%s)\n", story.Duration)
	}

	fmt.Println()
	colours.Prompt.Print("🌟 Enter the number of your chosen story (or 'q' to quit): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "q" || input == "quit" {
		colours.Warning.Println("👋 Maybe next time! Sweet dreams! 🌙")
		return
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(stories) {
		colours.Error.Println("❌ Invalid selection! Please try again.")
		return
	}

	selectedStory := stories[choice-1]
	sn.displayAndReadStory(selectedStory)
}

func (sn *StoryNest) displayAndReadStory(story story.Item) {
	fmt.Println()
	colours.Title.Printf("📖 %s\n", story.Title)
	colours.Author.Printf("✍️  by %s\n", story.Author)
	fmt.Printf("🎯 Age Group: %s | 🎭 Genre: %s | ⏱️ Duration: %s\n",
		story.AgeGroup, story.Genre, story.Duration)
	fmt.Printf("💡 %s\n", story.Description)
	fmt.Println()

	colours.Prompt.Print("🎧 Ready to listen? Press Enter to start (or 'skip' to just show text): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if strings.ToLower(input) == "skip" {
		fmt.Println()
		colours.Info.Println("📄 Story Text:")
		fmt.Println(story.Content)
		return
	}

	fmt.Println()
	colours.Success.Println("🎵 Starting story playback... 🎵")
	fmt.Println("💡 Press Ctrl+C to stop anytime")
	fmt.Println()

	// Start reading the story
	go func() {
		if err := sn.Tts.Speak(story.Content); err != nil {
			colours.Error.Printf("❌ TTS Error: %v\n", err)
		} else {
			colours.Success.Println("✅ Story finished! 🌟")
			colours.Prompt.Println("😴 Sleep tight! 🌙")
		}
	}()

	// Set book context for TTS caching if the engine supports it

	// Extract provider from story ID
	provider := extractProviderFromStoryID(story.ID)
	// Extract book ID from story ID (remove provider prefix)
	bookID := extractBookIDFromStoryID(story.ID)

	sn.Tts.SetBookContext(provider, bookID)

	colours.Info.Printf("🗂️ Using cache: %s/%s\n", provider, bookID)

	// Wait for user input or context cancellation
	sn.waitForUserInput()
}

// Helper function to extract provider from story ID
func extractProviderFromStoryID(storyID string) string {
	if strings.HasPrefix(storyID, "gutenberg-") {
		return "gutenberg"
	}
	// Add other providers as needed
	return "unknown"
}

// Helper function to extract book ID from story ID
func extractBookIDFromStoryID(storyID string) string {
	if strings.HasPrefix(storyID, "gutenberg-") {
		return strings.TrimPrefix(storyID, "gutenberg-")
	}
	// For other providers, return the full ID if no prefix is found
	return storyID
}

func (sn *StoryNest) waitForUserInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-sn.ctx.Done():
			return
		default:
			fmt.Print("\n⏸️  Press 'p' to pause/resume, 's' to stop, or Enter to continue: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			switch input {
			case "p", "pause":
				if sn.Tts.IsPlaying() {
					sn.Tts.Pause()
					colours.Warning.Println("⏸️  Paused")
				} else {
					sn.Tts.Resume()
					colours.Success.Println("▶️  Resumed")
				}
			case "s", "stop":
				sn.Tts.Stop()
				colours.Warning.Println("⏹️  Stopped")
				return
			case "":
				continue
			default:
				colours.Info.Println("ℹ️  Use 'p' for pause/resume, 's' to stop")
			}
		}
	}
}

func (sn *StoryNest) ManageLibraries(cmd *cobra.Command, args []string) {
	fmt.Println()
	colours.Title.Println("🏛️ Story Libraries 🏛️")
	fmt.Println()

	for i, library := range sn.libraries {
		fmt.Printf("%d. ", i+1)
		colours.Info.Printf("%s", library.Name)
		fmt.Printf(" (%d stories)\n", len(library.Stories))
		fmt.Printf("   🔗 %s\n", library.URL)
		fmt.Println()
	}

	colours.Success.Printf("✨ Total: %d libraries with %d stories\n",
		len(sn.libraries), len(sn.getAllStories()))
}

func (sn *StoryNest) ConfigureSettings(cmd *cobra.Command, args []string) {
	fmt.Println()
	colours.Title.Println("⚙️ TTS Settings ⚙️")
	fmt.Println()

	colours.Prompt.Println("🎤 Voice Settings:")
	fmt.Println("  • Current voice: default")
	fmt.Println("  • Speed: 1.0x")
	fmt.Println("  • Volume: 100%")
	fmt.Println()

	colours.Info.Println("💡 In a full implementation, you could:")
	fmt.Println("  • Choose different voices (child-friendly, storyteller, etc.)")
	fmt.Println("  • Adjust reading speed")
	fmt.Println("  • Control volume levels")
	fmt.Println("  • Select language preferences")
	fmt.Println("  • Configure audio output devices")
}

func (sn *StoryNest) getAllStories() []story.Item {
	var allStories []story.Item
	for _, library := range sn.libraries {
		allStories = append(allStories, library.Stories...)
	}
	return allStories
}

func (sn *StoryNest) findStoryByID(id string) *story.Item {
	for _, library := range sn.libraries {
		for _, story := range library.Stories {
			if story.ID == id {
				return &story
			}
		}
	}
	return nil
}

// HTTP client functions for fetching stories from public APIs
func (sn *StoryNest) fetchLibraryFromURL(url string) (*library.StoryLibrary, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var library library.StoryLibrary
	if err := json.Unmarshal(body, &library); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &library, nil
}

// LoadGutenbergLibrary loads stories from Project Gutenberg with caching
func (sn *StoryNest) LoadGutenbergLibrary() error {
	// Get user's cache directory (or use current directory as fallback)
	cacheDir := getCacheDirectory()

	// Create cache with 24-hour refresh
	cache := guten.NewGutenbergCache(cacheDir, 24*time.Hour)

	// Get the library (from cache or API)
	gutenbergLibrary, err := cache.GetLibrary()
	if err != nil {
		return err
	}

	// Add to our libraries
	sn.libraries = append(sn.libraries, *gutenbergLibrary)

	colours.Success.Printf("✨ Loaded %d stories from Project Gutenberg\n", len(gutenbergLibrary.Stories))
	return nil
}

// RefreshGutenbergCache forces a refresh of the Gutenberg cache
func (sn *StoryNest) RefreshGutenbergCache(cmd *cobra.Command, args []string) {
	colours.Info.Println("🔄 Refreshing Gutenberg cache...")

	cacheDir := getCacheDirectory()
	cache := guten.NewGutenbergCache(cacheDir, 24*time.Hour)

	// Clear existing cache
	if err := cache.ClearCache(); err != nil {
		colours.Error.Printf("❌ Failed to clear cache: %v\n", err)
		return
	}

	// Fetch fresh data
	gutenbergLibrary, err := cache.GetLibrary()
	if err != nil {
		colours.Error.Printf("❌ Failed to refresh cache: %v\n", err)
		return
	}

	// Update our libraries (remove old Gutenberg library if exists)
	newLibraries := make([]library.StoryLibrary, 0)
	for _, lib := range sn.libraries {
		if lib.Name != "Project Gutenberg Children's Collection" {
			newLibraries = append(newLibraries, lib)
		}
	}
	newLibraries = append(newLibraries, *gutenbergLibrary)
	sn.libraries = newLibraries

	colours.Success.Printf("✅ Cache refreshed! Loaded %d fresh stories from Project Gutenberg\n", len(gutenbergLibrary.Stories))
}

// ShowCacheStatus displays information about the Gutenberg cache
func (sn *StoryNest) ShowCacheStatus(cmd *cobra.Command, args []string) {
	colours.Title.Println("📊 Gutenberg Cache Status")

	cacheDir := getCacheDirectory()
	cache := guten.NewGutenbergCache(cacheDir, 24*time.Hour)

	info, err := cache.GetCacheInfo()
	if err != nil {
		colours.Error.Printf("❌ Failed to get cache info: %v\n", err)
		return
	}

	if info["exists"].(bool) {
		colours.Success.Println("✅ Cache exists")
		colours.Info.Printf("📁 Location: %s\n", filepath.Join(cacheDir, "gutenberg_cache.json"))
		colours.Info.Printf("📏 Size: %d bytes\n", info["size"].(int64))
		colours.Info.Printf("🕐 Last modified: %s\n", info["last_modified"].(time.Time).Format("2006-01-02 15:04:05"))

		if info["is_fresh"].(bool) {
			colours.Success.Println("🔄 Cache is fresh")
		} else {
			colours.Warning.Println("⏰ Cache is stale")
		}

		colours.Info.Printf("⏳ Max age: %.1f hours\n", info["max_age_hours"].(float64))
	} else {
		colours.Warning.Println("❌ Cache does not exist")
		colours.Info.Println("💡 Run 'storynest gutenberg refresh' to create cache")
	}
}

// Add Gutenberg commands to your main.go rootCmd
func (sn *StoryNest) AddGutenbergCommands(rootCmd *cobra.Command) {
	// Gutenberg parent command
	gutenbergCmd := &cobra.Command{
		Use:   "gutenberg",
		Short: "📚 Manage Project Gutenberg stories",
		Long:  "Access and manage stories from Project Gutenberg's free digital library",
	}

	// Refresh subcommand
	refreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "🔄 Refresh Gutenberg cache",
		Long:  "Download fresh stories from Project Gutenberg API",
		Run:   sn.RefreshGutenbergCache,
	}

	// Status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "📊 Show cache status",
		Long:  "Display information about the local Gutenberg cache",
		Run:   sn.ShowCacheStatus,
	}

	// Load subcommand
	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "📖 Load Gutenberg stories",
		Long:  "Load stories from Project Gutenberg (cached or fresh)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sn.LoadGutenbergLibrary(); err != nil {
				colours.Error.Printf("❌ Failed to load Gutenberg library: %v\n", err)
			}
		},
	}

	gutenbergCmd.AddCommand(refreshCmd, statusCmd, loadCmd)
	rootCmd.AddCommand(gutenbergCmd)
}

// getCacheDirectory returns the appropriate cache directory
func getCacheDirectory() string {
	// Try to use user's cache directory
	if cacheDir, err := os.UserCacheDir(); err == nil {
		storyNestCache := filepath.Join(cacheDir, "storynest")
		return storyNestCache
	}

	// Try user's home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		storyNestCache := filepath.Join(homeDir, ".storynest", "cache")
		return storyNestCache
	}

	// Get current working directory as fallback
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "cache")
	}

	// Final fallback to a simple cache directory in current location
	return "cache"
}

// UpdatedLoadSampleLibraries - modify your existing method to include Gutenberg
func (sn *StoryNest) LoadSampleLibrariesWithGutenberg() {
	// Load your existing sample libraries first
	sn.LoadSampleLibraries()

	// Then try to load Gutenberg stories
	colours.Info.Println("🌐 Loading Project Gutenberg stories...")
	if err := sn.LoadGutenbergLibrary(); err != nil {
		colours.Warning.Printf("⚠️ Could not load Gutenberg stories: %v\n", err)
		colours.Info.Println("💡 You can manually load them later with: storynest gutenberg load")
	}
}

// ConfigureTTSEngine allows users to configure TTS engine settings
func (sn *StoryNest) ConfigureTTSEngine(cmd *cobra.Command, args []string) {
	fmt.Println()
	colours.Title.Println("🎤 TTS Engine Configuration 🎤")
	fmt.Println()

	// Show current engine
	colours.Info.Printf("Current Engine: %s\n", sn.getCurrentEngineName())
	fmt.Println()

	// Show available engines
	engines := tts.GetAvailableEngines()
	colours.Prompt.Println("Available TTS Engines:")
	for i, engine := range engines {
		fmt.Printf("  %d. %s", i+1, engine)
		if string(engine) == sn.getCurrentEngineName() {
			colours.Success.Print(" (current)")
		}
		fmt.Println()
	}
	fmt.Println()

	colours.Prompt.Print("Select engine number (or press Enter to keep current): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		colours.Info.Println("Keeping current engine")
		return
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(engines) {
		colours.Error.Println("❌ Invalid selection")
		return
	}

	selectedEngine := engines[choice-1]

	// Create new engine with current config
	config := tts.Config{
		Type: string(selectedEngine),

		//todo:
		Speed:  1.0,
		Volume: 1.0,
		Voice:  "default",
	}

	newEngine, err := tts.NewEngine(config)
	if err != nil {
		colours.Error.Printf("❌ Failed to create %s engine: %v\n", selectedEngine, err)
		return
	}

	sn.Tts = newEngine
	colours.Success.Printf("✅ Switched to %s engine\n", selectedEngine)

	// If it's Chirp, show additional configuration options
	if selectedEngine == tts.EngineTypeGoogleClassic {
		sn.configureChirpSettings()
	}
}

func (sn *StoryNest) configureChirpSettings() {
	fmt.Println()
	colours.Title.Println("🌟 Google Chirp TTS Configuration 🌟")
	fmt.Println()

	// Show voice selection
	voices, err := sn.Tts.GetAvailableVoices()
	if err != nil {
		colours.Error.Printf("❌ Failed to get available voices: %v\n", err)
		return
	}

	colours.Prompt.Println("Available Chirp Voices (recommended for children):")
	fmt.Println()

	// Group voices by type for better presentation
	journeyVoices := []string{}
	neuralVoices := []string{}
	standardVoices := []string{}

	for _, voice := range voices {
		if strings.Contains(voice, "Journey") {
			journeyVoices = append(journeyVoices, voice)
		} else if strings.Contains(voice, "Neural") {
			neuralVoices = append(neuralVoices, voice)
		} else {
			standardVoices = append(standardVoices, voice)
		}
	}

	// Show Journey voices (best for children)
	if len(journeyVoices) > 0 {
		colours.Success.Println("🌟 Journey Voices (Best for Children):")
		for i, voice := range journeyVoices {
			gender := "Unknown"
			if strings.Contains(voice, "Journey-F") {
				gender = "Female"
			} else if strings.Contains(voice, "Journey-D") || strings.Contains(voice, "Journey-O") {
				gender = "Male"
			}
			fmt.Printf("  %d. %s (%s)\n", i+1, voice, gender)
		}
		fmt.Println()
	}

	// Show Neural voices
	if len(neuralVoices) > 0 {
		colours.Info.Println("🧠 Neural Voices (High Quality):")
		for i, voice := range neuralVoices {
			fmt.Printf("  %d. %s\n", len(journeyVoices)+i+1, voice)
		}
		fmt.Println()
	}

	// Show standard voices
	if len(standardVoices) > 0 {
		colours.Info.Println("📢 Standard Voices:")
		for i, voice := range standardVoices {
			fmt.Printf("  %d. %s\n", len(journeyVoices)+len(neuralVoices)+i+1, voice)
		}
		fmt.Println()
	}

	colours.Prompt.Print("Select voice number (or press Enter for default): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "" {
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(voices) {
			colours.Error.Println("❌ Invalid selection")
			return
		}

		selectedVoice := voices[choice-1]
		if err := sn.Tts.SetVoice(selectedVoice); err != nil {
			colours.Error.Printf("❌ Failed to set voice: %v\n", err)
			return
		}

		colours.Success.Printf("✅ Voice set to: %s\n", selectedVoice)
	}

	// Configure speed
	fmt.Println()
	colours.Prompt.Print("Enter speaking speed (0.25-4.0, current: 1.0): ")
	speedInput, _ := reader.ReadString('\n')
	speedInput = strings.TrimSpace(speedInput)

	if speedInput != "" {
		speed, err := strconv.ParseFloat(speedInput, 64)
		if err != nil || speed < 0.25 || speed > 4.0 {
			colours.Error.Println("❌ Speed must be between 0.25 and 4.0")
		} else {
			if err := sn.Tts.SetSpeed(speed); err != nil {
				colours.Error.Printf("❌ Failed to set speed: %v\n", err)
			} else {
				colours.Success.Printf("✅ Speed set to: %.2f\n", speed)
			}
		}
	}

	// Show cache information if available
	if cacheable, ok := sn.Tts.(tts.CacheableEngine); ok {
		fmt.Println()
		colours.Info.Println("📁 Cache Information:")
		if stats, err := cacheable.GetCacheStats(); err == nil {
			colours.Info.Printf("  Cache Directory: %s\n", stats["cache_directory"])
			colours.Info.Printf("  Cached Files: %d\n", stats["cached_files"])
			colours.Info.Printf("  Total Size: %.2f MB\n", stats["total_size_mb"])
		}
	}
}

// Show TTS Engine Status
func (sn *StoryNest) ShowTTSStatus(cmd *cobra.Command, args []string) {
	fmt.Println()
	colours.Title.Println("🎤 TTS Engine Status 🎤")
	fmt.Println()

	// Current engine info
	colours.Success.Printf("Engine: %s\n", sn.getCurrentEngineName())
	colours.Info.Printf("Status: %s\n", sn.getTTSStatus())

	// Show voices if available
	if voices, err := sn.Tts.GetAvailableVoices(); err == nil && len(voices) > 0 {
		colours.Info.Printf("Available Voices: %d\n", len(voices))
		if len(voices) <= 10 {
			for _, voice := range voices {
				fmt.Printf("  • %s\n", voice)
			}
		} else {
			for i := 0; i < 5; i++ {
				fmt.Printf("  • %s\n", voices[i])
			}
			fmt.Printf("  ... and %d more\n", len(voices)-5)
		}
	}

	// Show cache stats for Chirp
	if cacheable, ok := sn.Tts.(tts.CacheableEngine); ok {
		fmt.Println()
		colours.Info.Println("📁 Cache Statistics:")
		if stats, err := cacheable.GetCacheStats(); err == nil {
			fmt.Printf("  Directory: %s\n", stats["cache_directory"])
			fmt.Printf("  Files: %d\n", stats["cached_files"])
			fmt.Printf("  Size: %.2f MB\n", stats["total_size_mb"])
		}
	}
}

// Clear TTS Cache
func (sn *StoryNest) ClearTTSCache(cmd *cobra.Command, args []string) {
	if cacheable, ok := sn.Tts.(tts.CacheableEngine); ok {
		colours.Info.Println("🧹 Clearing TTS cache...")
		if err := cacheable.ClearCache(); err != nil {
			colours.Error.Printf("❌ Failed to clear cache: %v\n", err)
		} else {
			colours.Success.Println("✅ TTS cache cleared successfully!")
		}
	} else {
		colours.Warning.Println("⚠️ Current TTS engine doesn't support caching")
	}
}

func (sn *StoryNest) getCurrentEngineName() string {
	// This would need to be implemented based on how you track the current engine
	// For now, return a placeholder
	return "Unknown"
}

func (sn *StoryNest) getTTSStatus() string {
	if sn.Tts.IsPlaying() {
		return "🔊 Playing"
	}
	if enhanced, ok := sn.Tts.(tts.EnhancedEngine); ok && enhanced.IsPaused() {
		return "⏸️ Paused"
	}
	return "⏹️ Stopped"
}

// Add these commands to your main.go rootCmd setup:

// AddTTSCommands adds TTS management commands to the CLI
func (sn *StoryNest) AddTTSCommands(rootCmd *cobra.Command) {
	// TTS parent command
	ttsCmd := &cobra.Command{
		Use:   "tts",
		Short: "🎤 Manage text-to-speech settings",
		Long:  "Configure and manage TTS engines, voices, and settings",
	}

	// Configure subcommand
	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "⚙️ Configure TTS engine",
		Long:  "Select and configure TTS engine and voice settings",
		Run:   sn.ConfigureTTSEngine,
	}

	// Status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "📊 Show TTS status",
		Long:  "Display current TTS engine status and available voices",
		Run:   sn.ShowTTSStatus,
	}

	// Clear cache subcommand
	clearCacheCmd := &cobra.Command{
		Use:   "clear-cache",
		Short: "🧹 Clear TTS cache",
		Long:  "Clear cached audio files (applies to Chirp TTS)",
		Run:   sn.ClearTTSCache,
	}

	// Test TTS subcommand
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "🔊 Test TTS with sample text",
		Long:  "Test the current TTS engine with sample text",
		Run: func(cmd *cobra.Command, args []string) {
			testText := "Hello! This is a test of the StoryNest text-to-speech system. How does it sound?"
			if len(args) > 0 {
				testText = strings.Join(args, " ")
			}

			colours.Info.Printf("🔊 Testing TTS with: \"%s\"\n", testText)
			if err := sn.Tts.Speak(testText); err != nil {
				colours.Error.Printf("❌ TTS test failed: %v\n", err)
			} else {
				colours.Success.Println("✅ TTS test started successfully!")
			}
		},
	}

	ttsCmd.AddCommand(configureCmd, statusCmd, clearCacheCmd, testCmd)
	rootCmd.AddCommand(ttsCmd)
}
