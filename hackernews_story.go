package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func getHackernewsStories(config *Config) ([]HackerNewsResult, error) {
	var results []HackerNewsResult
	page := 0
	for {
		log.WithField("page", page).Info("Fetching page")
		var response HackerNewsResponse
		client := resty.New()
		_, err := client.R().
			SetQueryParams(map[string]string{
				"query": config.Tag,
				"tags":  "story",
				"page":  strconv.FormatInt(int64(page), 10),
			}).
			SetResult(&response).
			Get("http://hn.algolia.com/api/v1/search_by_date")
		if err != nil {
			return results, err
		}

		page += 1
		if len(response.Hits) == 0 {
			break
		}

		for _, result := range response.Hits {
			if len(result.HighlightResult.URL.MatchedWords) > 0 {
				if !strings.Contains(strings.ToLower(result.HighlightResult.URL.Value), config.Tag) {
					continue
				}
			}
			if len(result.HighlightResult.Author.MatchedWords) > 0 {
				if !strings.Contains(strings.ToLower(result.HighlightResult.Author.Value), config.Tag) {
					continue
				}
			}

			results = append(results, result)
		}
	}

	return results, nil
}

func sendSlackNotificationForHackernewsStory(result HackerNewsResult, config *Config) error {
	logFields := log.Fields{
		"question_id": result.ObjectID,
		"title":       result.Title,
	}

	link := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", result.ObjectID)
	fields := []slack.AttachmentField{
		{
			Title: "# Points",
			Value: strconv.FormatInt(int64(result.Points), 10),
			Short: true,
		},
		{
			Title: "# Comments",
			Value: strconv.FormatInt(int64(result.NumComments), 10),
			Short: true,
		},
		{
			Title: "Type",
			Value: "📚",
			Short: true,
		},
	}

	if len(result.HighlightResult.URL.Value) > 0 {
		fields = append(fields, slack.AttachmentField{
			Title: "Original Link",
			Value: result.HighlightResult.URL.Value,
			Short: true,
		})
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New story on Hacker News!",
		AuthorName: result.Author,
		AuthorLink: fmt.Sprintf("https://news.ycombinator.com/user?id=%s", result.Author),
		Title:      result.Title,
		TitleLink:  link,
		Footer:     "Hacker News Story Notification",
		FooterIcon: hackernewsIconUrl,
		Ts:         json.Number(strconv.FormatInt(int64(result.CreatedAt.Unix()), 10)),
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
	results, err := getHackernewsStories(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("story_count", len(results)).Info("Processing questions")
	for _, result := range results {
		logFields := log.Fields{
			"story_object_id": result.ObjectID,
			"title":           result.Title,
		}

		var entity HackerNewsStory
		if dbResult := db.First(&entity, "object_id = ?", result.ObjectID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new story")
		entity = HackerNewsStory{
			ObjectID: result.ObjectID,
			Title:    result.Title,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting story into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForHackernewsStory(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting story to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_story_count": len(results),
		"inserted_story_count":  inserted,
		"notified_story_count":  notified,
	}).Info("Done with hacker news stories")

	return nil
}
