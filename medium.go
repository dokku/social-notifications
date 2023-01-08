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

var mediumIconURL = "https://emoji.slack-edge.com/T085AJH3L/medium/ea7124868c6b2c68.png"

type MediumArticle struct {
	ID        int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	ArticleID string `gorm:"not null" form:"article_id" json:"article_id"`
	Title     string `gorm:"not null" form:"title" json:"title"`
}

type MediumTopFeedsResponse struct {
	Topfeeds []string `json:"topfeeds"`
	Mode     string   `json:"mode"`
	Tag      string   `json:"tag"`
}

type MediumAuthorResult struct {
	TwitterUsername string `json:"twitter_username"`
	Bio             string `json:"bio"`
	TopWriterIn     struct {
	} `json:"top_writer_in"`
	IsSuspended             bool   `json:"is_suspended"`
	Username                string `json:"username"`
	MediumMemberAt          string `json:"medium_member_at"`
	FollowingCount          int    `json:"following_count"`
	FollowersCount          int    `json:"followers_count"`
	IsWriterProgramEnrolled bool   `json:"is_writer_program_enrolled"`
	AllowNotes              bool   `json:"allow_notes"`
	Fullname                string `json:"fullname"`
	ID                      string `json:"id"`
	ImageURL                string `json:"image_url"`
}

type MediumResult struct {
	ID             string   `json:"id"`
	Tags           []string `json:"tags"`
	Claps          int      `json:"claps"`
	LastModifiedAt string   `json:"last_modified_at"`
	PublishedAt    string   `json:"published_at"`
	URL            string   `json:"url"`
	ImageURL       string   `json:"image_url"`
	Lang           string   `json:"lang"`
	PublicationID  string   `json:"publication_id"`
	WordCount      int      `json:"word_count"`
	Title          string   `json:"title"`
	ReadingTime    float64  `json:"reading_time"`
	ResponsesCount int      `json:"responses_count"`
	Voters         int      `json:"voters"`
	Author         string   `json:"author"`
	Subtitle       string   `json:"subtitle"`
}

func getMediumArticles(config *Config) ([]string, error) {
	var results []string
	log.Info("Fetching page")
	var response MediumTopFeedsResponse
	client := resty.New()
	_, err := client.R().
		SetResult(&response).
		SetHeaders(map[string]string{
			"X-RapidAPI-Key":  config.RapidApiKey,
			"X-RapidAPI-Host": "medium2.p.rapidapi.com",
		}).
		Get(fmt.Sprintf("https://medium2.p.rapidapi.com/topfeeds/%s/new", config.Tag))
	if err != nil {
		return results, err
	}

	return response.Topfeeds, nil
}

func getMediumArticle(articleID string, config *Config) (MediumResult, error) {
	log.WithField("article_id", articleID).Info("Fetching article")
	var response MediumResult
	client := resty.New()
	_, err := client.R().
		SetResult(&response).
		SetHeaders(map[string]string{
			"X-RapidAPI-Key":  config.RapidApiKey,
			"X-RapidAPI-Host": "medium2.p.rapidapi.com",
		}).
		Get(fmt.Sprintf("https://medium2.p.rapidapi.com/article/%s", articleID))
	if err != nil {
		return response, err
	}

	return response, nil
}

func getMediumAuthor(authorID string, config *Config) (MediumAuthorResult, error) {
	log.WithField("author_id", authorID).Info("Fetching author")
	var response MediumAuthorResult
	client := resty.New()
	_, err := client.R().
		SetResult(&response).
		SetHeaders(map[string]string{
			"X-RapidAPI-Key":  config.RapidApiKey,
			"X-RapidAPI-Host": "medium2.p.rapidapi.com",
		}).
		Get(fmt.Sprintf("https://medium2.p.rapidapi.com/user/%s", authorID))
	if err != nil {
		return response, err
	}

	return response, nil
}

func sendSlackNotificationForMediumArticle(result MediumResult, config *Config) error {
	if !config.NotifySlack {
		return nil
	}

	logFields := log.Fields{
		"article_id": result.ID,
		"title":      result.Title,
	}

	author, err := getMediumAuthor(result.Author, config)
	if err != nil {
		return err
	}

	t, err := time.Parse("2006-01-02 15:04:05", result.PublishedAt)
	if err != nil {
		return err
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New article on Medium!",
		AuthorName: author.Fullname,
		AuthorLink: fmt.Sprintf("https://medium.com/@%s", author.Username),
		Title:      result.Title,
		TitleLink:  result.URL,
		Footer:     "Medium Article Notification",
		FooterIcon: mediumIconURL,
		Ts:         json.Number(strconv.FormatInt(int64(t.Unix()), 10)),
	}

	log.WithFields(logFields).Info("Notifying slack")
	messageOpts := []slack.MsgOption{
		slack.MsgOptionAsUser(false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionIconEmoji(":medium:"),
		slack.MsgOptionText("New article on <"+result.URL+"|Medium>", false),
		slack.MsgOptionUsername("Medium Article Notifications"),
		slack.MsgOptionDisableLinkUnfurl(),
	}

	api := slack.New(config.SlackToken)
	if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
		return err
	}

	return nil
}

func processMediumArticles(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&MediumArticle{}); err != nil {
		return fmt.Errorf("error migrating MediumArticle: %w", err)
	}

	if config.RapidApiKey == "" {
		log.Warn("No RAPID_API_KEY specified, skipping medium")
		return nil
	}

	log.Info("Fetching articles")
	results, err := getMediumArticles(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("article_count", len(results)).Info("Processing articles")
	for _, articleID := range results {
		logFields := log.Fields{
			"article_id": articleID,
		}

		var entity MediumArticle
		if dbResult := db.First(&entity, "article_id = ?", articleID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		result, err := getMediumArticle(articleID, config)
		if err != nil {
			return err
		}

		log.WithFields(logFields).Info("Inserting new article")
		entity = MediumArticle{
			ArticleID: result.ID,
			Title:     result.Title,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting article into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForMediumArticle(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting article to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_article_count": len(results),
		"inserted_article_count":  inserted,
		"notified_article_count":  notified,
	}).Info("Done with medium articles")

	return nil
}
