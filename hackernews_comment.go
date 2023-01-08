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

type HackerNewsComment struct {
	ID       int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	ObjectID string `gorm:"not null" form:"object_id" json:"object_id"`
}

func getHackernewsComments(config *Config) ([]HackerNewsResult, error) {
	var stories []HackerNewsResult
	page := 0
	for {
		log.WithField("page", page).Info("Fetching page")
		var results HackerNewsResponse
		client := resty.New()
		_, err := client.R().
			SetQueryParams(map[string]string{
				"query": config.Tag,
				"tags":  "comment",
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

		for _, comment := range results.Hits {
			if len(comment.HighlightResult.Author.MatchedWords) > 0 {
				if !strings.Contains(comment.HighlightResult.Author.Value, config.Tag) {
					continue
				}
			}
			if len(comment.HighlightResult.CommentText.MatchedWords) > 0 {
				if !strings.Contains(comment.HighlightResult.CommentText.Value, config.Tag) {
					continue
				}
			}
			if len(comment.HighlightResult.StoryTitle.MatchedWords) > 0 {
				if !strings.Contains(comment.HighlightResult.StoryTitle.Value, config.Tag) {
					continue
				}
			}
			if len(comment.HighlightResult.StoryURL.MatchedWords) > 0 {
				if !strings.Contains(comment.HighlightResult.StoryURL.Value, config.Tag) {
					continue
				}
			}

			stories = append(stories, comment)
		}
	}

	return stories, nil
}

func sendSlackNotificationForHackernewsComment(comment HackerNewsResult, config *Config) error {
	logFields := log.Fields{
		"question_id": comment.ObjectID,
	}

	link := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", comment.ObjectID)
	fields := []slack.AttachmentField{
		{
			Title: "Type",
			Value: "✍️",
			Short: true,
		},
	}

	if len(comment.HighlightResult.URL.Value) > 0 {
		fields = append(fields, slack.AttachmentField{
			Title: "Original Link",
			Value: comment.HighlightResult.URL.Value,
			Short: true,
		})
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New comment on Hacker News!",
		AuthorName: comment.Author,
		AuthorLink: fmt.Sprintf("https://news.ycombinator.com/user?id=%s", comment.Author),
		TitleLink:  link,
		Footer:     "Hacker News Comment Notification",
		FooterIcon: hackernewsIconUrl,
		Ts:         json.Number(strconv.FormatInt(int64(comment.CreatedAt.Unix()), 10)),
		Fields:     fields,
	}

	if config.NotifySlack {
		log.WithFields(logFields).Info("Notifying slack")
		messageOpts := []slack.MsgOption{
			slack.MsgOptionAsUser(false),
			slack.MsgOptionAttachments(attachment),
			slack.MsgOptionIconEmoji(":hacker-news:"),
			slack.MsgOptionText("New comment on <"+link+"|Hacker News>", false),
			slack.MsgOptionUsername("Hacker News Comment Notifications"),
			slack.MsgOptionDisableLinkUnfurl(),
		}

		api := slack.New(config.SlackToken)
		if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
			return err
		}
	}

	return nil
}

func processHackernewsComments(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&HackerNewsComment{}); err != nil {
		return fmt.Errorf("error migrating HackerNewsComment: %w", err)
	}

	log.Info("Fetching comments")
	results, err := getHackernewsComments(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("comment_count", len(results)).Info("Processing questions")
	for _, result := range results {
		logFields := log.Fields{
			"comment_object_id": result.ObjectID,
		}

		var entity HackerNewsComment
		if dbResult := db.First(&entity, "object_id = ?", result.ObjectID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new comment")
		entity = HackerNewsComment{
			ObjectID: result.ObjectID,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting comment into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForHackernewsComment(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting comment to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_comment_count": len(results),
		"inserted_comment_count":  inserted,
		"notified_comment_count":  notified,
	}).Info("Done with hacker news stories")

	return nil
}
