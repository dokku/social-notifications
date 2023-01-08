package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type DevtoArticle struct {
	ID        int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	ArticleID int64  `gorm:"not null" form:"article_id" json:"article_id"`
	Title     string `gorm:"not null" form:"title" json:"title"`
}

type DevtoResponse []DevtoArticleResult

type DevtoArticleResult struct {
	TypeOf                 string     `json:"type_of"`
	ID                     int        `json:"id"`
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	ReadablePublishDate    string     `json:"readable_publish_date"`
	Slug                   string     `json:"slug"`
	Path                   string     `json:"path"`
	URL                    string     `json:"url"`
	CommentsCount          int        `json:"comments_count"`
	PublicReactionsCount   int        `json:"public_reactions_count"`
	CollectionID           int        `json:"collection_id"`
	PublishedTimestamp     time.Time  `json:"published_timestamp"`
	PositiveReactionsCount int        `json:"positive_reactions_count"`
	CoverImage             string     `json:"cover_image"`
	SocialImage            string     `json:"social_image"`
	CanonicalURL           string     `json:"canonical_url"`
	CreatedAt              time.Time  `json:"created_at"`
	EditedAt               *time.Time `json:"edited_at"`
	CrosspostedAt          *time.Time `json:"crossposted_at"`
	PublishedAt            time.Time  `json:"published_at"`
	LastCommentAt          time.Time  `json:"last_comment_at"`
	ReadingTimeMinutes     int        `json:"reading_time_minutes"`
	TagList                []string   `json:"tag_list"`
	Tags                   string     `json:"tags"`
	User                   struct {
		Name            string `json:"name"`
		Username        string `json:"username"`
		TwitterUsername string `json:"twitter_username"`
		GithubUsername  string `json:"github_username"`
		UserID          int    `json:"user_id"`
		WebsiteURL      string `json:"website_url"`
		ProfileImage    string `json:"profile_image"`
		ProfileImage90  string `json:"profile_image_90"`
	} `json:"user"`
	Organization struct {
		Name           string `json:"name"`
		Username       string `json:"username"`
		Slug           string `json:"slug"`
		ProfileImage   string `json:"profile_image"`
		ProfileImage90 string `json:"profile_image_90"`
	} `json:"organization"`
	FlareTag struct {
		Name         string `json:"name"`
		BgColorHex   string `json:"bg_color_hex"`
		TextColorHex string `json:"text_color_hex"`
	} `json:"flare_tag"`
}

var devtoIconUrl = "https://emoji.slack-edge.com/T085AJH3L/devto-rainbow/387781e03f7a17fe.png"

func getDevtoArticles(config *Config) ([]DevtoArticleResult, error) {
	var results []DevtoArticleResult
	page := 1
	for {
		log.WithField("page", page).Info("Fetching page")
		var response DevtoResponse
		client := resty.New()
		_, err := client.R().
			SetQueryParams(map[string]string{
				"per_page": "100",
				"page":     strconv.FormatInt(int64(page), 10),
				"tag":      config.Tag,
			}).
			SetResult(&response).
			Get("https://dev.to/api/articles")
		if err != nil {
			return results, err
		}

		page += 1
		if len(response) == 0 {
			break
		}

		results = append(results, response...)
	}

	return results, nil
}

func sendSlackNotificationForDevtoArticle(result DevtoArticleResult, config *Config) error {
	if !config.NotifySlack {
		return nil
	}

	logFields := log.Fields{
		"article_id": result.ID,
		"title":      result.Title,
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New article on Dev.to!",
		AuthorName: result.User.Username,
		AuthorLink: fmt.Sprintf("https://dev.to/%s", result.User.Username),
		Title:      result.Title,
		TitleLink:  result.URL,
		Footer:     "Dev.to Article Notification",
		FooterIcon: devtoIconUrl,
		Ts:         json.Number(strconv.FormatInt(int64(result.CreatedAt.Unix()), 10)),
	}

	log.WithFields(logFields).Info("Notifying slack")
	messageOpts := []slack.MsgOption{
		slack.MsgOptionAsUser(false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionIconEmoji(":devto-rainbow:"),
		slack.MsgOptionText("New article on <"+result.URL+"|Dev.to>", false),
		slack.MsgOptionUsername("Dev.to Article Notifications"),
		slack.MsgOptionDisableLinkUnfurl(),
	}

	api := slack.New(config.SlackToken)
	if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
		return err
	}

	return nil
}

func processDevtoArticles(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&DevtoArticle{}); err != nil {
		return fmt.Errorf("error migrating DevtoArticle: %w", err)
	}

	log.Info("Fetching articles")
	results, err := getDevtoArticles(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("article_count", len(results)).Info("Processing articles")
	for _, result := range results {
		logFields := log.Fields{
			"article_id": result.ID,
			"title":      result.Title,
		}

		var entity DevtoArticle
		if dbResult := db.First(&entity, "article_id = ?", result.ID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new article")
		entity = DevtoArticle{
			ArticleID: int64(result.ID),
			Title:     result.Title,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting article into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForDevtoArticle(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting article to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_article_count": len(results),
		"inserted_article_count":  inserted,
		"notified_article_count":  notified,
	}).Info("Done with dev.to articles")

	return nil
}
