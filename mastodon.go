package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

var mastodonIconURL = "https://emoji.slack-edge.com/T085AJH3L/mastodon/18ff0c46d671d904.png"

type MastodonToot struct {
	ID     int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	TootID string `gorm:"not null" form:"toot_id" json:"toot_id"`
}

type MastodonTootResult struct {
	ID                 string      `json:"id"`
	CreatedAt          time.Time   `json:"created_at"`
	InReplyToID        interface{} `json:"in_reply_to_id"`
	InReplyToAccountID interface{} `json:"in_reply_to_account_id"`
	Sensitive          bool        `json:"sensitive"`
	SpoilerText        string      `json:"spoiler_text"`
	Visibility         string      `json:"visibility"`
	Language           string      `json:"language"`
	URI                string      `json:"uri"`
	URL                string      `json:"url"`
	RepliesCount       int         `json:"replies_count"`
	ReblogsCount       int         `json:"reblogs_count"`
	FavouritesCount    int         `json:"favourites_count"`
	EditedAt           interface{} `json:"edited_at"`
	Content            string      `json:"content"`
	Reblog             interface{} `json:"reblog"`
	Account            struct {
		ID             string        `json:"id"`
		Username       string        `json:"username"`
		Acct           string        `json:"acct"`
		DisplayName    string        `json:"display_name"`
		Locked         bool          `json:"locked"`
		Bot            bool          `json:"bot"`
		Discoverable   bool          `json:"discoverable"`
		Group          bool          `json:"group"`
		CreatedAt      time.Time     `json:"created_at"`
		Note           string        `json:"note"`
		URL            string        `json:"url"`
		Avatar         string        `json:"avatar"`
		AvatarStatic   string        `json:"avatar_static"`
		Header         string        `json:"header"`
		HeaderStatic   string        `json:"header_static"`
		FollowersCount int           `json:"followers_count"`
		FollowingCount int           `json:"following_count"`
		StatusesCount  int           `json:"statuses_count"`
		LastStatusAt   string        `json:"last_status_at"`
		Emojis         []interface{} `json:"emojis"`
		Fields         []struct {
			Name       string    `json:"name"`
			Value      string    `json:"value"`
			VerifiedAt time.Time `json:"verified_at"`
		} `json:"fields"`
	} `json:"account"`
	MediaAttachments []interface{} `json:"media_attachments"`
	Mentions         []interface{} `json:"mentions"`
	Tags             []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"tags"`
	Emojis []interface{} `json:"emojis"`
	Card   interface{}   `json:"card"`
	Poll   interface{}   `json:"poll"`
}

func getToots(config *Config) ([]MastodonTootResult, error) {
	var response []MastodonTootResult
	client := resty.New()
	_, err := client.R().
		SetResult(&response).
		Get(fmt.Sprintf("http://mastodon.social/api/v1/timelines/tag/%s", config.Tag))

	return response, err
}

func sendSlackNotificationForMastodonToot(result MastodonTootResult, config *Config) error {
	if !config.NotifySlack {
		return nil
	}

	logFields := log.Fields{
		"toot_id": result.ID,
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(result.Content)
	if err != nil {
		return err
	}

	markdown = strings.ReplaceAll(markdown, "\\*", "*")

	link := result.URL

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New toot on Mastodon!",
		AuthorName: result.Account.Acct,
		AuthorLink: result.Account.URL,
		Title:      "New toot on Mastodon!",
		TitleLink:  link,
		Text:       markdown,
		MarkdownIn: []string{"text"},
		Footer:     "Mastodon Toot Notification",
		FooterIcon: mastodonIconURL,
		Ts:         json.Number(strconv.FormatInt(int64(result.CreatedAt.Unix()), 10)),
	}

	log.WithFields(logFields).Info("Notifying slack")
	messageOpts := []slack.MsgOption{
		slack.MsgOptionAsUser(false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionIconEmoji(":mastodon:"),
		slack.MsgOptionText("New toot on <"+link+"|Mastodon>", false),
		slack.MsgOptionUsername("Mastodon Toot Notifications"),
		slack.MsgOptionDisableLinkUnfurl(),
	}

	api := slack.New(config.SlackToken)
	if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
		return err
	}

	return nil
}

func processMastodon(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&MastodonToot{}); err != nil {
		return fmt.Errorf("error migrating MastodonToot: %w", err)
	}

	log.Info("Fetching toots")
	results, err := getToots(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("story_count", len(results)).Info("Processing toots")
	for _, result := range results {
		logFields := log.Fields{
			"toot_id": result.ID,
		}

		var entity MastodonToot
		if dbResult := db.First(&entity, "toot_id = ?", result.ID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new toot")
		entity = MastodonToot{
			TootID: result.ID,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting toot into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForMastodonToot(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting toot to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_toot_count": len(results),
		"inserted_toot_count":  inserted,
		"notified_toot_count":  notified,
	}).Info("Done with mastodon.social toots")

	return nil
}
