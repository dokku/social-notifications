package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type HackerNewsStory struct {
	ID       int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	ObjectID string `gorm:"not null" form:"object_id" json:"object_id"`
	Title    string `gorm:"not null" form:"title" json:"title"`
}

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

func getHackernewsStories(config *Config) ([]HackerNewsResult, error) {
	var stories []HackerNewsResult
	page := 0
	for {
		var results HackerNewsResponse
		client := resty.New()
		_, err := client.R().
			SetQueryParams(map[string]string{
				"query": config.Tag,
				"tags":  "story",
				"page":  strconv.FormatInt(int64(page), 10),
			}).
			SetResult(&results).
			Get("http://hn.algolia.com/api/v1/search_by_date")
		if err != nil {
			return stories, err
		}

		page += 1
		if len(results.Hits) == 0 {
			break
		}

		for _, story := range results.Hits {
			if len(story.HighlightResult.URL.MatchedWords) > 0 {
				if !strings.Contains(story.HighlightResult.URL.Value, config.Tag) {
					continue
				}
			}
			if len(story.HighlightResult.Author.MatchedWords) > 0 {
				if !strings.Contains(story.HighlightResult.Author.Value, config.Tag) {
					continue
				}
			}

			stories = append(stories, story)
		}
	}

	return stories, nil
}

func sendSlackNotificationForHackernewsStory(story HackerNewsResult, config *Config) error {
	logFields := log.Fields{
		"question_id": story.ObjectID,
		"title":       story.Title,
	}

	link := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", story.ObjectID)
	fields := []slack.AttachmentField{
		{
			Title: "# Points",
			Value: strconv.FormatInt(int64(story.Points), 10),
			Short: true,
		},
		{
			Title: "# Comments",
			Value: strconv.FormatInt(int64(story.NumComments), 10),
			Short: true,
		},
		{
			Title: "Type",
			Value: "📚",
			Short: true,
		},
	}

	if len(story.HighlightResult.URL.Value) > 0 {
		fields = append(fields, slack.AttachmentField{
			Title: "Original Link",
			Value: story.HighlightResult.URL.Value,
			Short: true,
		})
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New story on Hacker News!",
		AuthorName: story.Author,
		AuthorLink: fmt.Sprintf("https://news.ycombinator.com/user?id=%s", story.Author),
		Title:      story.Title,
		TitleLink:  link,
		Footer:     "Hacker News Story Notification",
		FooterIcon: hackernewsIconUrl,
		Ts:         json.Number(strconv.FormatInt(int64(story.CreatedAt.Unix()), 10)),
		Fields:     fields,
	}

	if config.NotifySlack {
		log.WithFields(logFields).Info("Notifying slack")
		messageOpts := []slack.MsgOption{
			slack.MsgOptionAsUser(false),
			slack.MsgOptionAttachments(attachment),
			slack.MsgOptionIconEmoji(":hacker-news:"),
			slack.MsgOptionText("New story on <"+link+"|Hacker News>", false),
			slack.MsgOptionUsername("Hacker News Story Notifications"),
			slack.MsgOptionDisableLinkUnfurl(),
		}

		api := slack.New(config.SlackToken)
		if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
			return err
		}
	}

	return nil
}

func processHackernewsStories(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&HackerNewsStory{}); err != nil {
		return fmt.Errorf("error migrating HackerNewsStory: %w", err)
	}

	log.Info("Fetching stories")
	stories, err := getHackernewsStories(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("story_count", len(stories)).Info("Processing questions")
	for _, story := range stories {
		logFields := log.Fields{
			"story_object_id": story.ObjectID,
			"title":           story.Title,
		}

		var entity HackerNewsStory
		result := db.First(&entity, "object_id = ?", story.ObjectID)
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new story")
		entity = HackerNewsStory{
			ObjectID: story.ObjectID,
			Title:    story.Title,
		}

		if result := db.Create(&entity); result.Error != nil {
			log.WithError(result.Error).WithFields(logFields).Fatal("error inserting story into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForHackernewsStory(story, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting story to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_story_count": len(stories),
		"inserted_story_count":  inserted,
		"notified_story_count":  notified,
	}).Info("Done with hacker news stories")

	return nil
}
