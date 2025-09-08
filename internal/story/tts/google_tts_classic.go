package tts

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/texttospeech/apiv1"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

type GoogleClassicTTSEngine struct {
	client          *texttospeech.Client
	ctx             context.Context
	voice           string
	speed           float64
	volume          float64
	isPlaying       bool
	ctrl            *beep.Ctrl
	format          beep.Format
	streamer        beep.StreamSeekCloser
	done            chan bool
	mu              sync.Mutex
	cacheRootDir    string
	currentProvider string
	currentBookID   string
}

func newGoogleClassicTTSEngine(cacheDir string) (*GoogleClassicTTSEngine, error) {
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %w", err)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	return &GoogleClassicTTSEngine{
		client:       client,
		ctx:          ctx,
		voice:        "en-GB-Chirp3-HD-Umbriel", //some random default
		speed:        1.0,
		volume:       1.0,
		cacheRootDir: cacheDir,
	}, nil
}

// SetBookContext sets the current provider and book ID for caching purposes
// This should be called before Speak() to ensure proper cache organization
func (g *GoogleClassicTTSEngine) SetBookContext(provider, bookID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.currentProvider = provider
	g.currentBookID = bookID
}

// getCacheDirectory returns the hierarchical cache directory for the current book
func (g *GoogleClassicTTSEngine) getCacheDirectory() string {
	if g.currentProvider == "" || g.currentBookID == "" {
		// Fallback to old behavior if context not set
		return g.cacheRootDir
	}

	return filepath.Join(g.cacheRootDir, g.currentProvider, "google_classic", g.currentBookID)
}

// getCacheFilePrefix returns the prefix for cache files (bookID if available)
func (g *GoogleClassicTTSEngine) getCacheFilePrefix() string {
	if g.currentBookID != "" {
		return g.currentBookID
	}
	// Fallback to hash-based naming
	return "audio"
}

func (g *GoogleClassicTTSEngine) Speak(text string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Get the cache directory for this book
	cacheDir := g.getCacheDirectory()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	audioCfg := &texttospeechpb.AudioConfig{
		AudioEncoding: texttospeechpb.AudioEncoding_MP3,
	}

	// Chirp voices often don't support speakingRate/pitch/SSML — skip them
	if !strings.Contains(strings.ToLower(g.voice), "chirp") {
		audioCfg.SpeakingRate = g.speed  // supported generally. See docs.
		audioCfg.VolumeGainDb = g.volume // interpret as dB gain (or map to dB as you prefer)
		// audioCfg.Pitch = ...   // set only if voice supports it
	}

	// Create a unique identifier for this specific text + voice combination
	contentHash := fmt.Sprintf("%x", md5Sum(text+g.voice))[:8] // Use first 8 chars of hash

	// File prefix based on current context
	filePrefix := g.getCacheFilePrefix()

	chunks := splitIntoChunks(text, 4800) // a little under 5000 to be safe

	// Generate MP3 files if not cached
	allChunksExist := true
	for i := 0; i < len(chunks); i++ {
		chunkFileName := fmt.Sprintf("%s_%s_%d.mp3", filePrefix, contentHash, i)
		chunkPath := filepath.Join(cacheDir, chunkFileName)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			allChunksExist = false
			break
		}
	}

	if !allChunksExist {
		fmt.Printf("Generating audio for %s (provider: %s, book: %s)\n",
			filePrefix, g.currentProvider, g.currentBookID)

		for chunkIndex, chunk := range chunks {

			req := &texttospeechpb.SynthesizeSpeechRequest{
				Input: &texttospeechpb.SynthesisInput{
					InputSource: &texttospeechpb.SynthesisInput_Text{Text: chunk},
				},
				Voice: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: "en-US",
					Name:         "en-US-Chirp3-HD-Charon",
				},
				AudioConfig: &texttospeechpb.AudioConfig{
					AudioEncoding: texttospeechpb.AudioEncoding_MP3,
				},
			}
			resp, err := g.client.SynthesizeSpeech(g.ctx, req)
			if err != nil {
				return fmt.Errorf("failed to synthesize chunk %d: %w", chunkIndex, err)
			}

			chunkFileName := fmt.Sprintf("%s_%s_%d.mp3", filePrefix, contentHash, chunkIndex)
			chunkPath := filepath.Join(cacheDir, chunkFileName)

			if err := os.WriteFile(chunkPath, resp.AudioContent, 0644); err != nil {
				return fmt.Errorf("failed to write MP3 chunk %d to %s: %w", chunkIndex, chunkPath, err)
			}

			fmt.Printf("Cached audio chunk %d/%d to: %s\n", chunkIndex+1, len(chunks), chunkPath)
		}
	} else {
		fmt.Printf("Using cached audio for %s (provider: %s, book: %s)\n",
			filePrefix, g.currentProvider, g.currentBookID)
	}

	// Play cached files
	for i := 0; i < len(chunks); i++ {
		
		chunkFileName := fmt.Sprintf("%s_%s_%d.mp3", filePrefix, contentHash, i)
		chunkPath := filepath.Join(cacheDir, chunkFileName)

		f, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open cached MP3 %s: %w", chunkPath, err)
		}

		streamer, format, err := mp3.Decode(f)
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to decode MP3 %s: %w", chunkPath, err)
		}
		g.streamer = streamer
		g.format = format
		g.ctrl = &beep.Ctrl{Streamer: streamer, Paused: false}
		g.done = make(chan bool)

		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			streamer.Close()
			f.Close()
			return err
		}
		speaker.Play(beep.Seq(g.ctrl, beep.Callback(func() {
			g.isPlaying = false
			g.done <- true
		})))

		g.isPlaying = true
	}
	return nil
}

func (g *GoogleClassicTTSEngine) SetVoice(voice string) error {
	g.voice = voice
	return nil
}

func (g *GoogleClassicTTSEngine) SetSpeed(speed float64) error {
	g.speed = speed
	return nil
}

func (g *GoogleClassicTTSEngine) SetVolume(volume float64) error {
	g.volume = volume
	return nil
}

func (g *GoogleClassicTTSEngine) Stop() error {
	if g.streamer != nil {
		g.streamer.Close()
	}
	g.isPlaying = false
	return nil
}

func (g *GoogleClassicTTSEngine) Pause() error {
	if g.ctrl != nil {
		speaker.Lock()
		g.ctrl.Paused = true
		speaker.Unlock()
	}
	return nil
}

func (g *GoogleClassicTTSEngine) Resume() error {
	if g.ctrl != nil {
		speaker.Lock()
		g.ctrl.Paused = false
		speaker.Unlock()
	}
	return nil
}

func (g *GoogleClassicTTSEngine) IsPlaying() bool {
	return g.isPlaying
}

func (g *GoogleClassicTTSEngine) GetAvailableVoices() ([]string, error) {
	resp, err := g.client.ListVoices(g.ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return nil, err
	}
	voices := []string{}
	for _, v := range resp.Voices {
		voices = append(voices, v.Name)
	}
	return voices, nil
}

// GetCacheStats returns cache statistics for the current engine
func (g *GoogleClassicTTSEngine) GetCacheStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalFiles int64
	var totalSize int64

	// Walk through the entire cache directory tree
	err := filepath.Walk(g.cacheRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			totalFiles++
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return stats, err
	}

	stats["cache_directory"] = g.cacheRootDir
	stats["cached_files"] = totalFiles
	stats["total_size_mb"] = float64(totalSize) / (1024 * 1024)

	// Get provider-specific stats if context is available
	if g.currentProvider != "" {
		providerDir := filepath.Join(g.cacheRootDir, g.currentProvider, "google_classic")
		var providerFiles int64
		var providerSize int64

		filepath.Walk(providerDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
				providerFiles++
				providerSize += info.Size()
			}
			return nil
		})

		stats["provider"] = g.currentProvider
		stats["provider_files"] = providerFiles
		stats["provider_size_mb"] = float64(providerSize) / (1024 * 1024)
	}

	return stats, nil
}

// ClearCache removes all cached files
func (g *GoogleClassicTTSEngine) ClearCache() error {
	return os.RemoveAll(g.cacheRootDir)
}

// ClearProviderCache removes cached files for a specific provider
func (g *GoogleClassicTTSEngine) ClearProviderCache(provider string) error {
	providerDir := filepath.Join(g.cacheRootDir, provider, "google_classic")
	return os.RemoveAll(providerDir)
}

// ClearBookCache removes cached files for a specific book
func (g *GoogleClassicTTSEngine) ClearBookCache(provider, bookID string) error {
	bookDir := filepath.Join(g.cacheRootDir, provider, "google_classic", bookID)
	return os.RemoveAll(bookDir)
}

// ListCachedBooks returns a list of cached books for each provider
func (g *GoogleClassicTTSEngine) ListCachedBooks() (map[string][]string, error) {
	result := make(map[string][]string)

	// Walk through provider directories
	providerDirs, err := os.ReadDir(g.cacheRootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil // Empty cache
		}
		return nil, err
	}

	for _, providerDir := range providerDirs {
		if !providerDir.IsDir() {
			continue
		}

		providerName := providerDir.Name()
		engineDir := filepath.Join(g.cacheRootDir, providerName, "google_classic")

		bookDirs, err := os.ReadDir(engineDir)
		if err != nil {
			continue // Skip if engine directory doesn't exist
		}

		var books []string
		for _, bookDir := range bookDirs {
			if bookDir.IsDir() {
				books = append(books, bookDir.Name())
			}
		}

		if len(books) > 0 {
			result[providerName] = books
		}
	}

	return result, nil
}

// helper functions remain the same
func md5Sum(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func splitIntoChunks(text string, limit int) []string {
	var chunks []string
	runes := []rune(text) // safe for UTF-8
	for i := 0; i < len(runes); i += limit {
		end := i + limit
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
