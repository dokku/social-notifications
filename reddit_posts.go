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

var redditIconURL = "https://emoji.slack-edge.com/T085AJH3L/reddit/42103923a0791a10.png"

type RedditPost struct {
	ID     int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	PostID string `gorm:"not null" form:"post_id" json:"post_id"`
	Title  string `gorm:"not null" form:"title" json:"title"`
}
type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Modhash string `json:"modhash"`
		Dist    int    `json:"dist"`
		Facets  struct {
		} `json:"facets"`
		After     string `json:"after"`
		GeoFilter string `json:"geo_filter"`
		Children  []struct {
			Kind string         `json:"kind"`
			Data RedditPostItem `json:"data"`
		} `json:"children"`
		Before interface{} `json:"before"`
	} `json:"data"`
}

type RedditPostItem struct {
	ApprovedAtUtc              interface{}   `json:"approved_at_utc"`
	Subreddit                  string        `json:"subreddit"`
	Selftext                   string        `json:"selftext"`
	AuthorFullname             string        `json:"author_fullname"`
	Saved                      bool          `json:"saved"`
	ModReasonTitle             interface{}   `json:"mod_reason_title"`
	Gilded                     int           `json:"gilded"`
	Clicked                    bool          `json:"clicked"`
	Title                      string        `json:"title"`
	LinkFlairRichtext          []interface{} `json:"link_flair_richtext"`
	SubredditNamePrefixed      string        `json:"subreddit_name_prefixed"`
	Hidden                     bool          `json:"hidden"`
	Pwls                       int           `json:"pwls"`
	LinkFlairCSSClass          string        `json:"link_flair_css_class"`
	Downs                      int           `json:"downs"`
	ThumbnailHeight            interface{}   `json:"thumbnail_height"`
	TopAwardedType             interface{}   `json:"top_awarded_type"`
	HideScore                  bool          `json:"hide_score"`
	Name                       string        `json:"name"`
	Quarantine                 bool          `json:"quarantine"`
	LinkFlairTextColor         string        `json:"link_flair_text_color"`
	UpvoteRatio                float64       `json:"upvote_ratio"`
	AuthorFlairBackgroundColor interface{}   `json:"author_flair_background_color"`
	SubredditType              string        `json:"subreddit_type"`
	Ups                        int           `json:"ups"`
	TotalAwardsReceived        int           `json:"total_awards_received"`
	MediaEmbed                 struct {
	} `json:"media_embed"`
	ThumbnailWidth        interface{}   `json:"thumbnail_width"`
	AuthorFlairTemplateID interface{}   `json:"author_flair_template_id"`
	IsOriginalContent     bool          `json:"is_original_content"`
	UserReports           []interface{} `json:"user_reports"`
	SecureMedia           interface{}   `json:"secure_media"`
	IsRedditMediaDomain   bool          `json:"is_reddit_media_domain"`
	IsMeta                bool          `json:"is_meta"`
	Category              interface{}   `json:"category"`
	SecureMediaEmbed      struct {
	} `json:"secure_media_embed"`
	LinkFlairText       string        `json:"link_flair_text"`
	CanModPost          bool          `json:"can_mod_post"`
	Score               int           `json:"score"`
	ApprovedBy          interface{}   `json:"approved_by"`
	IsCreatedFromAdsUI  bool          `json:"is_created_from_ads_ui"`
	AuthorPremium       bool          `json:"author_premium"`
	Thumbnail           string        `json:"thumbnail"`
	Edited              bool          `json:"edited"`
	AuthorFlairCSSClass interface{}   `json:"author_flair_css_class"`
	AuthorFlairRichtext []interface{} `json:"author_flair_richtext"`
	Gildings            struct {
	} `json:"gildings"`
	ContentCategories        interface{}   `json:"content_categories"`
	IsSelf                   bool          `json:"is_self"`
	ModNote                  interface{}   `json:"mod_note"`
	Created                  float64       `json:"created"`
	LinkFlairType            string        `json:"link_flair_type"`
	Wls                      int           `json:"wls"`
	RemovedByCategory        interface{}   `json:"removed_by_category"`
	BannedBy                 interface{}   `json:"banned_by"`
	AuthorFlairType          string        `json:"author_flair_type"`
	Domain                   string        `json:"domain"`
	AllowLiveComments        bool          `json:"allow_live_comments"`
	SelftextHTML             string        `json:"selftext_html"`
	Likes                    interface{}   `json:"likes"`
	SuggestedSort            interface{}   `json:"suggested_sort"`
	BannedAtUtc              interface{}   `json:"banned_at_utc"`
	ViewCount                interface{}   `json:"view_count"`
	Archived                 bool          `json:"archived"`
	NoFollow                 bool          `json:"no_follow"`
	IsCrosspostable          bool          `json:"is_crosspostable"`
	Pinned                   bool          `json:"pinned"`
	Over18                   bool          `json:"over_18"`
	AllAwardings             []interface{} `json:"all_awardings"`
	Awarders                 []interface{} `json:"awarders"`
	MediaOnly                bool          `json:"media_only"`
	LinkFlairTemplateID      string        `json:"link_flair_template_id"`
	CanGild                  bool          `json:"can_gild"`
	Spoiler                  bool          `json:"spoiler"`
	Locked                   bool          `json:"locked"`
	AuthorFlairText          interface{}   `json:"author_flair_text"`
	TreatmentTags            []interface{} `json:"treatment_tags"`
	Visited                  bool          `json:"visited"`
	RemovedBy                interface{}   `json:"removed_by"`
	NumReports               interface{}   `json:"num_reports"`
	Distinguished            interface{}   `json:"distinguished"`
	SubredditID              string        `json:"subreddit_id"`
	AuthorIsBlocked          bool          `json:"author_is_blocked"`
	ModReasonBy              interface{}   `json:"mod_reason_by"`
	RemovalReason            interface{}   `json:"removal_reason"`
	LinkFlairBackgroundColor string        `json:"link_flair_background_color"`
	ID                       string        `json:"id"`
	IsRobotIndexable         bool          `json:"is_robot_indexable"`
	ReportReasons            interface{}   `json:"report_reasons"`
	Author                   string        `json:"author"`
	DiscussionType           interface{}   `json:"discussion_type"`
	NumComments              int           `json:"num_comments"`
	SendReplies              bool          `json:"send_replies"`
	WhitelistStatus          string        `json:"whitelist_status"`
	ContestMode              bool          `json:"contest_mode"`
	ModReports               []interface{} `json:"mod_reports"`
	AuthorPatreonFlair       bool          `json:"author_patreon_flair"`
	AuthorFlairTextColor     interface{}   `json:"author_flair_text_color"`
	Permalink                string        `json:"permalink"`
	ParentWhitelistStatus    string        `json:"parent_whitelist_status"`
	Stickied                 bool          `json:"stickied"`
	URL                      string        `json:"url"`
	SubredditSubscribers     int           `json:"subreddit_subscribers"`
	CreatedUtc               float64       `json:"created_utc"`
	NumCrossposts            int           `json:"num_crossposts"`
	Media                    interface{}   `json:"media"`
	IsVideo                  bool          `json:"is_video"`
}

func getRedditPosts(config *Config) ([]RedditPostItem, error) {
	var results []RedditPostItem
	var response RedditResponse
	client := resty.New()
	// todo: force http2 somehow
	_, err := client.R().
		SetQueryParams(map[string]string{
			"q":    config.Tag,
			"type": "link",
			"sort": "updated",
		}).
		SetHeaders(map[string]string{
			"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
		}).
		SetResult(&response).
		Get("https://www.reddit.com/search.json")
	if err != nil {
		return results, err
	}

	for _, item := range response.Data.Children {
		results = append(results, item.Data)
	}

	return results, nil
}

func sendSlackNotificationForRedditPost(result RedditPostItem, config *Config) error {
	if !config.NotifySlack {
		return nil
	}

	logFields := log.Fields{
		"post_id": result.ID,
	}

	fields := []slack.AttachmentField{
		{
			Title: "Subreddit",
			Value: fmt.Sprintf("/r/%s", result.Subreddit),
			Short: true,
		},
	}

	postID := strings.SplitN("result.Name", "_", 2)[1]
	link := fmt.Sprintf("https://www.reddit.com/r/chicagofood/comments/%s/", postID)

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New post on Reddit!",
		AuthorName: result.Author,
		AuthorLink: fmt.Sprintf("https://www.reddit.com/user/%s/", result.Author),
		Title:      result.Title,
		TitleLink:  link,
		Footer:     "Reddit Post Notification",
		FooterIcon: redditIconURL,
		Ts:         json.Number(strconv.FormatInt(int64(result.Created), 10)),
		Fields:     fields,
	}

	log.WithFields(logFields).Info("Notifying slack")
	messageOpts := []slack.MsgOption{
		slack.MsgOptionAsUser(false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionIconEmoji(":reddit:"),
		slack.MsgOptionText("New post on <"+link+"|Reddit>", false),
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
	log.WithField("post_count", len(results)).Info("Processing repositories")
	for _, result := range results {
		logFields := log.Fields{
			"post_id":   result.Name,
			"title":     result.Title,
			"subreddit": result.SubredditNamePrefixed,
		}

		var entity RedditPost
		if dbResult := db.First(&entity, "post_id = ?", result.ID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new post")
		entity = RedditPost{
			PostID: result.Name,
			Title:  result.Title,
		}

		// if dbResult := db.Create(&entity); dbResult.Error != nil {
		// 	log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting post into database")
		// 	continue
		// }

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
