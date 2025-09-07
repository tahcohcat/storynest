package library

import "storynest/internal/domain/story"

// Item represents a children's story

// StoryLibrary represents a collection of stories from different sources
type StoryLibrary struct {
	Name    string       `json:"name"`
	URL     string       `json:"url"`
	Stories []story.Item `json:"stories"`
}
