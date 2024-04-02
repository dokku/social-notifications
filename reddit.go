package main

var redditIconURL = "https://emoji.slack-edge.com/T085AJH3L/reddit/42103923a0791a10.png"

type RedditPost struct {
	ID     int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	PostID string `gorm:"not null" form:"post_id" json:"post_id"`
	Title  string `gorm:"not null" form:"title" json:"title"`
}

type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After     any                `json:"after"`
		Dist      int                `json:"dist"`
		Modhash   string             `json:"modhash"`
		GeoFilter string             `json:"geo_filter"`
		Children  []RedditPostResult `json:"children"`
		Before    any                `json:"before"`
	} `json:"data"`
}

type RedditPostResult struct {
	Kind string `json:"kind"`
	Data struct {
		ApprovedAtUtc  any     `json:"approved_at_utc"`
		Author         string  `json:"author"`
		AuthorFullname string  `json:"author_fullname"`
		CreatedUtc     float64 `json:"created_utc"`
		Hidden         bool    `json:"hidden"`
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Permalink      string  `json:"permalink"`
		Selftext       string  `json:"selftext"`
		SelftextHTML   string  `json:"selftext_html"`
		Subreddit      string  `json:"subreddit"`
		Thumbnail      string  `json:"thumbnail"`
		Title          string  `json:"title"`
		URL            string  `json:"url"`
	} `json:"data,omitempty"`
}
