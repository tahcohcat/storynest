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
	"storynest/internal/cli/scheme/colours"
	"storynest/internal/domain/library"
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
		libraries: []library.StoryLibrary{},
		Tts:       engine,
		ctx:       ctx,
		Cancel:    cancel,
	}
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

	count := 0
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

	sn.displayAndReadStory(randomStory)
}

func (sn *StoryNest) ReadStory(cmd *cobra.Command, args []string) {
	interactive, _ := cmd.Flags().GetBool("interactive")

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

	// Wait for user input or context cancellation
	sn.waitForUserInput()
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
