package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"storynest/internal/domain/library"
	"storynest/internal/domain/story"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// GutenCache handles fetching and caching Gutenberg stories
type GutenCache struct {
	cacheDir   string
	cacheFile  string
	maxAge     time.Duration
	httpClient *http.Client
}

// GutendexResponse represents the API response structure
type GutendexResponse struct {
	Count    int            `json:"count"`
	Next     *string        `json:"next"`
	Previous *string        `json:"previous"`
	Results  []GutendexBook `json:"results"`
}

// GutendexBook represents a book from the Gutendex API
type GutendexBook struct {
	ID            int               `json:"id"`
	Title         string            `json:"title"`
	Authors       []Author          `json:"authors"`
	Subjects      []string          `json:"subjects"`
	Languages     []string          `json:"languages"`
	Formats       map[string]string `json:"formats"`
	DownloadCount int               `json:"download_count"`
}

// Author represents an author from the API
type Author struct {
	Name      string `json:"name"`
	BirthYear *int   `json:"birth_year"`
	DeathYear *int   `json:"death_year"`
}

// CachedGutenbergData represents the cached library data
type CachedGutenbergData struct {
	Library     library.StoryLibrary `json:"library"`
	LastUpdated time.Time            `json:"last_updated"`
	TotalBooks  int                  `json:"total_books"`
}

// NewGutenbergCache creates a new Gutenberg cache instance
func NewGutenbergCache(cacheDir string, maxAge time.Duration) *GutenCache {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logrus.WithError(err).Warn("Failed to create cache directory")
	}

	return &GutenCache{
		cacheDir:  cacheDir,
		cacheFile: filepath.Join(cacheDir, "gutenberg_cache.json"),
		maxAge:    maxAge,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetLibrary returns the Gutenberg library, fetching from cache or API as needed
func (gc *GutenCache) GetLibrary() (*library.StoryLibrary, error) {
	// Check if cache exists and is fresh
	if gc.isCacheFresh() {
		logrus.Info("Loading Gutenberg stories from cache")
		return gc.loadFromCache()
	}

	// Cache is stale or doesn't exist, fetch from API
	logrus.Info("Fetching fresh Gutenberg stories from API")
	library, err := gc.fetchFromAPI()
	if err != nil {
		// If API fails, try to load from cache even if stale
		logrus.WithError(err).Warn("API fetch failed, trying stale cache")
		if cachedLibrary, cacheErr := gc.loadFromCache(); cacheErr == nil {
			return cachedLibrary, nil
		}
		return nil, fmt.Errorf("failed to fetch from API and no cache available: %w", err)
	}

	// Save to cache
	if err := gc.saveToCache(library); err != nil {
		logrus.WithError(err).Warn("Failed to save to cache")
	}

	return library, nil
}

// isCacheFresh checks if the cache file exists and is within the max age
func (gc *GutenCache) isCacheFresh() bool {
	info, err := os.Stat(gc.cacheFile)
	if err != nil {
		return false // Cache doesn't exist
	}

	return time.Since(info.ModTime()) < gc.maxAge
}

// loadFromCache loads the library from the cache file
func (gc *GutenCache) loadFromCache() (*library.StoryLibrary, error) {
	file, err := os.Open(gc.cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	var cached CachedGutenbergData
	if err := json.NewDecoder(file).Decode(&cached); err != nil {
		return nil, fmt.Errorf("failed to decode cache file: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"stories":      len(cached.Library.Stories),
		"last_updated": cached.LastUpdated.Format(time.RFC3339),
		"total_books":  cached.TotalBooks,
	}).Info("Loaded Gutenberg library from cache")

	return &cached.Library, nil
}

// saveToCache saves the library to the cache file
func (gc *GutenCache) saveToCache(library *library.StoryLibrary) error {
	cached := CachedGutenbergData{
		Library:     *library,
		LastUpdated: time.Now(),
		TotalBooks:  len(library.Stories),
	}

	file, err := os.Create(gc.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cached); err != nil {
		return fmt.Errorf("failed to encode cache data: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"stories": len(library.Stories),
		"file":    gc.cacheFile,
	}).Info("Saved Gutenberg library to cache")

	return nil
}

// fetchFromAPI fetches stories from the Gutendx API
func (gc *GutenCache) fetchFromAPI() (*library.StoryLibrary, error) {
	library := &library.StoryLibrary{
		Name:    "Project Gutenberg Children's Collection",
		URL:     "https://gutendx.com/books/",
		Stories: []story.Item{},
	}

	// Fetch multiple pages of children's books
	queries := []string{
		"?topic=children",
		"?topic=juvenile",
		"?topic=fairy",
		"?search=children%20story",
		"?search=bedtime%20story",
	}

	seenIDs := make(map[int]bool)

	for _, query := range queries {
		url := "https://gutendx.com/books/" + query + "&languages=en"
		stories, err := gc.fetchStoriesFromURL(url)
		if err != nil {
			logrus.WithError(err).WithField("url", url).Warn("Failed to fetch from URL")
			continue
		}

		// Add unique stories
		for _, story := range stories {
			// Convert string ID back to int for deduplication
			var bookID int
			fmt.Sscanf(story.ID, "gutenberg-%d", &bookID)

			if !seenIDs[bookID] {
				library.Stories = append(library.Stories, story)
				seenIDs[bookID] = true
			}
		}

		// Add a small delay between requests to be respectful
		time.Sleep(500 * time.Millisecond)
	}

	logrus.WithField("count", len(library.Stories)).Info("Fetched Gutenberg stories from API")
	return library, nil
}

// fetchStoriesFromURL fetches stories from a specific Gutendx URL
func (gc *GutenCache) fetchStoriesFromURL(url string) ([]story.Item, error) {
	resp, err := gc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d for URL %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response GutendexResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return gc.convertBooksToStories(response.Results), nil
}

// convertBooksToStories converts Gutendx books to our story format
func (gc *GutenCache) convertBooksToStories(books []GutendexBook) []story.Item {
	var stories []story.Item

	for _, book := range books {
		// Filter for appropriate content
		if !gc.isChildrensSuitable(book) {
			continue
		}

		// Get author name
		authorName := "Unknown"
		if len(book.Authors) > 0 {
			authorName = book.Authors[0].Name
		}

		// Determine age group based on subjects and title
		ageGroup := gc.determineAgeGroup(book)
		genre := gc.determineGenre(book)

		// Get text content URL (prefer plain text)
		contentURL := gc.getBestTextFormat(book.Formats)
		if contentURL == "" {
			continue // Skip if no readable format available
		}

		// Create story item
		story := story.Item{
			ID:          fmt.Sprintf("gutenberg-%d", book.ID),
			Title:       gc.cleanTitle(book.Title),
			Author:      authorName,
			Content:     fmt.Sprintf("[Content available at: %s]", contentURL),
			AgeGroup:    ageGroup,
			Genre:       genre,
			Duration:    gc.estimateDuration(book),
			Description: gc.createDescription(book),
		}

		stories = append(stories, story)
	}

	return stories
}

// isChildrensSuitable checks if a book is suitable for children
func (gc *GutenCache) isChildrensSuitable(book GutendexBook) bool {
	titleLower := strings.ToLower(book.Title)

	// Check for children-related keywords
	childrenKeywords := []string{
		"children", "child", "juvenile", "young", "fairy", "tale", "story",
		"bedtime", "nursery", "adventure", "magic", "animal", "fantasy",
	}

	for _, keyword := range childrenKeywords {
		if strings.Contains(titleLower, keyword) {
			return true
		}
	}

	// Check subjects
	for _, subject := range book.Subjects {
		subjectLower := strings.ToLower(subject)
		if strings.Contains(subjectLower, "children") ||
			strings.Contains(subjectLower, "juvenile") ||
			strings.Contains(subjectLower, "fairy") {
			return true
		}
	}

	return false
}

// determineAgeGroup estimates appropriate age group
func (gc *GutenCache) determineAgeGroup(book GutendexBook) string {
	titleLower := strings.ToLower(book.Title)

	if strings.Contains(titleLower, "baby") || strings.Contains(titleLower, "nursery") {
		return "0-3 years"
	}
	if strings.Contains(titleLower, "little") || strings.Contains(titleLower, "simple") {
		return "3-6 years"
	}
	if strings.Contains(titleLower, "adventure") || strings.Contains(titleLower, "mystery") {
		return "8-12 years"
	}

	return "4-8 years" // Default for most children's stories
}

// determineGenre determines the story genre
func (gc *GutenCache) determineGenre(book GutendexBook) string {
	titleLower := strings.ToLower(book.Title)

	if strings.Contains(titleLower, "fairy") || strings.Contains(titleLower, "magic") {
		return "Fairy Tale"
	}
	if strings.Contains(titleLower, "adventure") {
		return "Adventure"
	}
	if strings.Contains(titleLower, "animal") {
		return "Animal Story"
	}
	if strings.Contains(titleLower, "mystery") {
		return "Mystery"
	}

	return "Classic Tale"
}

// getBestTextFormat finds the best text format URL
func (gc *GutenCache) getBestTextFormat(formats map[string]string) string {
	// Prefer plain text formats
	preferredFormats := []string{
		"text/plain; charset=utf-8",
		"text/plain",
		"text/html",
	}

	for _, format := range preferredFormats {
		if url, exists := formats[format]; exists {
			return url
		}
	}

	return ""
}

// estimateDuration provides a rough duration estimate
func (gc *GutenCache) estimateDuration(book GutendexBook) string {
	// This is a rough estimate - in a real implementation,
	// you might fetch a sample of the text to get word count
	titleLength := len(strings.Fields(book.Title))

	if titleLength <= 3 {
		return "5-10 minutes"
	} else if titleLength <= 6 {
		return "10-20 minutes"
	} else {
		return "20+ minutes"
	}
}

// createDescription creates a description from available metadata
func (gc *GutenCache) createDescription(book GutendexBook) string {
	if len(book.Subjects) > 0 {
		// Use the first subject as description base
		subject := book.Subjects[0]
		return fmt.Sprintf("A classic tale from Project Gutenberg. %s", subject)
	}

	return "A classic children's story from Project Gutenberg's free digital library."
}

// cleanTitle cleans up book titles
func (gc *GutenCache) cleanTitle(title string) string {
	// Remove common Project Gutenberg suffixes
	cleanTitle := strings.TrimSpace(title)

	// Remove language indicators
	if strings.Contains(cleanTitle, "(English)") {
		cleanTitle = strings.Replace(cleanTitle, "(English)", "", 1)
	}

	return strings.TrimSpace(cleanTitle)
}

// ClearCache removes the cache file
func (gc *GutenCache) ClearCache() error {
	if err := os.Remove(gc.cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	logrus.Info("Cleared Gutenberg cache")
	return nil
}

// GetCacheInfo returns information about the cache
func (gc *GutenCache) GetCacheInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	if stat, err := os.Stat(gc.cacheFile); err == nil {
		info["exists"] = true
		info["size"] = stat.Size()
		info["last_modified"] = stat.ModTime()
		info["is_fresh"] = gc.isCacheFresh()
		info["max_age_hours"] = gc.maxAge.Hours()
	} else {
		info["exists"] = false
	}

	return info, nil
}
