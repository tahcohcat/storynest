package generator

//import (
//	"context"
//	"encoding/json"
//	"fmt"
//	"io"
//	"net/http"
//	"storynest/internal/domain/story"
//	"strings"
//)
//
//type Gutendex struct {
//	baseURL string
//}
//
//func NewGutendex() *Gutendex {
//	return &Gutendex{
//		baseURL: "https://gutendex.com",
//	}
//}
//
//type GutendexAPIResponse struct {
//	Count    int     `json:"count"`
//	Next     *string `json:"next"`     // pointer since it can be null
//	Previous *string `json:"previous"` // pointer since it can be null
//	Results  []Book  `json:"results"`
//}
//
//type Book struct {
//	ID            int               `json:"id"`
//	Title         string            `json:"title"`
//	Authors       []Author          `json:"authors"`
//	Summaries     []string          `json:"summaries"`
//	Translators   []Translator      `json:"translators"`
//	Subjects      []string          `json:"subjects"`
//	Bookshelves   []string          `json:"bookshelves"`
//	Languages     []string          `json:"languages"`
//	Copyright     bool              `json:"copyright"`
//	MediaType     string            `json:"media_type"`
//	Formats       map[string]string `json:"formats"`
//	DownloadCount int               `json:"download_count"`
//}
//
//type Author struct {
//	Name      string `json:"name"`
//	BirthYear int    `json:"birth_year"`
//	DeathYear int    `json:"death_year"`
//}
//
//type Translator struct {
//	Name      string `json:"name"`
//	BirthYear int    `json:"birth_year"`
//	DeathYear int    `json:"death_year"`
//}
//
//func (g *Gutendex) ListOnlineResources() ([]*story.OnlineResource, error) {
//
//	// https://gutendex.com/books/?topic=children&mime_type=text%2F&languages=en
//	url := fmt.Sprintf("%s/books/?topic=children&mime_type=text&languages=en", g.baseURL)
//
//	resp, err := http.Get(url)
//	if err != nil {
//		return nil, fmt.Errorf("failed to fetch gutendex: %w", err)
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		body, _ := io.ReadAll(resp.Body)
//		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
//	}
//
//	var apiResp GutendexAPIResponse
//	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
//		return nil, fmt.Errorf("decode error: %w", err)
//	}
//
//	resources := make([]*story.OnlineResource, 0, len(apiResp.Results))
//	for _, r := range apiResp.Results {
//		authors := make([]string, len(r.Authors))
//		for i, a := range r.Authors {
//			authors[i] = a.Name
//		}
//
//		resources = append(resources, &story.OnlineResource{
//			ID:          fmt.Sprintf("%d", r.ID),
//			Description: r.Summaries[0],
//			Metadata: map[string]string{
//				"title":    r.Title,
//				"authors":  strings.Join(authors, ","),
//				"subjects": strings.Join(r.Subjects, ","),
//			},
//			Name: r.Title,
//			URL: func() string {
//				return fmt.Sprintf("%s/ebooks/%d.txt.utf-8", g.baseURL, r.ID)
//			}(),
//		})
//	}
//
//	return resources, nil
//}
//
//func (g *Gutendex) LoadResource(ctx context.Context, r *story.OnlineResource) (*story.Item, error) {
//
//	if r == nil {
//		return nil, fmt.Errorf("nil gutendex item")
//	}
//
//	if !strings.HasPrefix(r.URL, "http") {
//		return nil, fmt.Errorf("invalid url: %s", r.URL)
//	}
//
//	// todo: improve this
//	resp, err := http.Get(r.URL)
//	if err != nil {
//		return nil, fmt.Errorf("failed to fetch text: %w", err)
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
//	}
//
//	body, err := io.ReadAll(resp.Body)
//	if err != nil {
//		return nil, fmt.Errorf("read error: %w", err)
//	}
//
//	return &story.Item{
//		ID:          r.ID,
//		Title:       r.Name,
//		Author:      r.Metadata["authors"],
//		Content:     string(body),
//		AgeGroup:    r.Metadata["age_groups"],
//		Genre:       r.Metadata["genre"],
//		Duration:    r.Metadata["duration"],
//		Description: r.Metadata["description"],
//	}, nil
//}
