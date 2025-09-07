package story

type OnlineResource struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Provider    string            `json:"product"`
	Metadata    map[string]string `json:"metadata"`
	URL         string            `json:"url"`
}

type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Content     string `json:"content"`
	AgeGroup    string `json:"age_group"`
	Genre       string `json:"genre"`
	Duration    string `json:"duration"`
	Description string `json:"description"`
}
