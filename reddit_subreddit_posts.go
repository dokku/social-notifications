package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

func getRedditPosts(config *Config) ([]RedditPostResult, error) {
	var results []RedditPostResult
	var response RedditResponse
	client := resty.New()
	_, err := client.R().
		SetHeaders(map[string]string{
			"authority":                 "www.reddit.com",
			"pragma":                    "no-cache",
			"cache-control":             "no-cache",
			"sec-ch-ua":                 `"Google Chrome";v="89", "Chromium";v="89", ";Not A Brand";v="99"`,
			"sec-ch-ua-mobile":          "?0",
			"upgrade-insecure-requests": "1",
			"user-agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36",
			"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"dnt":                       "1",
			"sec-fetch-site":            "none",
			"sec-fetch-mode":            "navigate",
			"sec-fetch-user":            "?1",
			"sec-fetch-dest":            "document",
			"accept-language":           "en-GB,en;q=0.9",
		}).
		SetResult(&response).
		Get(fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=100", config.Tag))
	if err != nil {
		return results, err
	}

	results = append(results, response.Data.Children...)

	return results, nil
}

func sendSlackNotificationForRedditPost(result RedditPostResult, config *Config) error {
	if !config.NotifySlack {
		return nil
	}

	logFields := log.Fields{
		"article_id": result.Data.ID,
		"title":      result.Data.Title,
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New post on Reddit!",
		AuthorName: result.Data.Author,
		AuthorLink: fmt.Sprintf("https://www.reddit.com/user/%s", result.Data.Author),
		Title:      result.Data.Title,
		TitleLink:  result.Data.URL,
		Text:       result.Data.Selftext,
		Footer:     "Reddit Post Notification",
		FooterIcon: redditIconURL,
		Ts:         json.Number(strconv.FormatInt(int64(result.Data.CreatedUtc), 10)),
	}

	log.WithFields(logFields).Info("Notifying slack")
	messageOpts := []slack.MsgOption{
		slack.MsgOptionAsUser(false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionIconEmoji(":reddit:"),
		slack.MsgOptionText("New post on <"+result.Data.URL+"|Reddit>", false),
		slack.MsgOptionUsername("Reddit Post Notifications"),
		slack.MsgOptionDisableLinkUnfurl(),
	}

	api := slack.New(config.SlackToken)
	if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
		return err
	}

	return nil
}

func processRedditPosts(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&RedditPost{}); err != nil {
		return fmt.Errorf("error migrating RedditPost: %w", err)
	}

	log.Info("Fetching posts")
	results, err := getRedditPosts(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("post_count", len(results)).Info("Processing posts")
	for _, result := range results {
		logFields := log.Fields{
			"post_id": result.Data.ID,
			"title":   result.Data.Title,
		}

		var entity RedditPost
		if dbResult := db.First(&entity, "post_id = ?", result.Data.ID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new post")
		entity = RedditPost{
			PostID: result.Data.ID,
			Title:  result.Data.Title,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting post into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForRedditPost(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting post to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_post_count": len(results),
		"inserted_post_count":  inserted,
		"notified_post_count":  notified,
	}).Info("Done with reddit posts")

	return nil
}
