package main

import (
	"time"
)

type HackerNewsResponse struct {
	Hits             []HackerNewsResult `json:"hits"`
	NbHits           int                `json:"nbHits"`
	Page             int                `json:"page"`
	NbPages          int                `json:"nbPages"`
	HitsPerPage      int                `json:"hitsPerPage"`
	ExhaustiveNbHits bool               `json:"exhaustiveNbHits"`
	ExhaustiveTypo   bool               `json:"exhaustiveTypo"`
	Exhaustive       struct {
		NbHits bool `json:"nbHits"`
		Typo   bool `json:"typo"`
	} `json:"exhaustive"`
	Query               string `json:"query"`
	Params              string `json:"params"`
	ProcessingTimeMS    int    `json:"processingTimeMS"`
	ProcessingTimingsMS struct {
		AfterFetch struct {
			Format struct {
				Highlighting int `json:"highlighting"`
				Total        int `json:"total"`
			} `json:"format"`
			Total int `json:"total"`
		} `json:"afterFetch"`
		Fetch struct {
			Scanning int `json:"scanning"`
			Total    int `json:"total"`
		} `json:"fetch"`
		Request struct {
			RoundTrip int `json:"roundTrip"`
		} `json:"request"`
		Total int `json:"total"`
	} `json:"processingTimingsMS"`
	ServerTimeMS int `json:"serverTimeMS"`
}

type HackerNewsResult struct {
	CreatedAt       time.Time `json:"created_at"`
	Title           string    `json:"title"`
	URL             string    `json:"url"`
	Author          string    `json:"author"`
	Points          int       `json:"points"`
	StoryText       string    `json:"story_text"`
	CommentText     string    `json:"comment_text"`
	NumComments     int       `json:"num_comments"`
	StoryID         int       `json:"story_id"`
	StoryTitle      string    `json:"story_title"`
	StoryURL        string    `json:"story_url"`
	ParentID        int       `json:"parent_id"`
	CreatedAtI      int       `json:"created_at_i"`
	RelevancyScore  int       `json:"relevancy_score"`
	Tags            []string  `json:"_tags"`
	ObjectID        string    `json:"objectID"`
	HighlightResult struct {
		Author struct {
			Value        string   `json:"value"`
			MatchLevel   string   `json:"matchLevel"`
			MatchedWords []string `json:"matchedWords"`
		} `json:"author"`
		CommentText struct {
			Value            string   `json:"value"`
			MatchLevel       string   `json:"matchLevel"`
			FullyHighlighted bool     `json:"fullyHighlighted"`
			MatchedWords     []string `json:"matchedWords"`
		} `json:"comment_text"`
		StoryTitle struct {
			Value        string   `json:"value"`
			MatchLevel   string   `json:"matchLevel"`
			MatchedWords []string `json:"matchedWords"`
		} `json:"story_title"`
		StoryURL struct {
			Value        string   `json:"value"`
			MatchLevel   string   `json:"matchLevel"`
			MatchedWords []string `json:"matchedWords"`
		} `json:"story_url"`
		Title struct {
			Value        string   `json:"value"`
			MatchLevel   string   `json:"matchLevel"`
			MatchedWords []string `json:"matchedWords"`
		} `json:"title"`
		URL struct {
			Value        string   `json:"value"`
			MatchLevel   string   `json:"matchLevel"`
			MatchedWords []string `json:"matchedWords"`
		} `json:"url"`
	} `json:"_highlightResult"`
}

var hackernewsIconUrl = "https://emoji.slack-edge.com/T085AJH3L/hacker-news/0daae30bfa8eefc6.png"
