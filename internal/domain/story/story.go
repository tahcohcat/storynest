package story

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
