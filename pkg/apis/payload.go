package apis

type PayloadNewItem struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Date  string `json:"date"`
}
