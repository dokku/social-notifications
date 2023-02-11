package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/g8rswimmer/go-twitter/v2"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

var twitterIconURL = "https://emoji.slack-edge.com/T085AJH3L/twitter/290f7fdbde70c82d.png"

type TwitterTweet struct {
	ID      int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	TweetID string `gorm:"not null" form:"tweet_id" json:"tweet_id"`
}

type authorize struct {
	Token string
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

func getTweets(config *Config) ([]*twitter.TweetDictionary, error) {
	var results []*twitter.TweetDictionary
	client := &twitter.Client{
		Authorizer: authorize{
			Token: config.TwitterBearerToken,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}

	// search for all tweets in the last day, with a max of 100
	// we'll ignore pagination for now since its unlikely to be needed for dokku...
	opts := twitter.TweetRecentSearchOpts{
		MaxResults: 100,
		Expansions: []twitter.Expansion{
			twitter.ExpansionEntitiesMentionsUserName,
			twitter.ExpansionAuthorID,
			twitter.ExpansionReferencedTweetsID,
		},
		StartTime: time.Now().AddDate(0, 0, -1),
		TweetFields: []twitter.TweetField{
			twitter.TweetFieldCreatedAt,
			twitter.TweetFieldConversationID,
			twitter.TweetFieldAttachments,
			twitter.TweetFieldLanguage,
		},
	}

	tweetResponse, err := client.TweetRecentSearch(context.Background(), config.Tag, opts)
	if err != nil {
		return results, fmt.Errorf("tweet lookup error: %v", err)
	}

	// this is rough but many tweets should be ignored in these languages because they refer to either:
	// - some pop artist's dog (kpop I think)
	// - count dooku (a mispelling from star wars)
	// - something crappy (telegu I believe)
	// ideally we can parse the entities and tell if its actually about dokku,
	// but honestly I don't care too much
	ignoreLanguages := map[string]bool{
		"es": true,
		"et": true,
		"ja": true,
		"in": true,
		"it": true,
	}

	// ignore anything with these words too
	ignoreWords := []string{
		"caliphate",
		"chennai",
		"chatta",
		"chettha",
		"comte",
		"conde",
		"disney",
		"dokkan",
		"hera",
		"imarat",
		"isis",
		"luke",
		"kadyrov",
		"movie",
		"shiseru",
		"sushi",
		"tamil",
		"theatre",
		"theater",
		"umarov",
	}

	// ignore these authors completely
	ignoreAuthors := []string{"dokku"}

	// allow all tweets with these words to go through
	allowWords := []string{"caprover", "coolify", "heroku"}

	for _, tweet := range tweetResponse.Raw.TweetDictionaries() {
		ignore := false
		for _, word := range allowWords {
			if strings.Contains(strings.ToLower(tweet.Tweet.Text), word) {
				results = append(results, tweet)
				ignore = true
				break
			}
		}

		if ignoreLanguages[tweet.Tweet.Language] {
			continue
		}

		for _, word := range ignoreWords {
			if strings.Contains(strings.ToLower(tweet.Tweet.Text), word) {
				ignore = true
				break
			}
		}

		for _, author := range ignoreAuthors {
			if tweet.Author.UserName == author {
				ignore = true
				break
			}
		}

		// ignore anyone with the tag in the name
		if strings.Contains(strings.ToLower(tweet.Author.UserName), config.Tag) {
			continue
		}

		// ignore anyone with the tag in the username
		if strings.Contains(strings.ToLower(tweet.Author.Name), config.Tag) {
			continue
		}

		for _, mention := range tweet.Mentions {
			if strings.Contains(strings.ToLower(mention.User.UserName), config.Tag) {
				ignore = true
				break
			}

			// ignore anyone with the tag in the username
			if strings.Contains(strings.ToLower(mention.User.Name), config.Tag) {
				ignore = true
				break
			}
		}

		// ignore retweets
		for _, reference := range tweet.ReferencedTweets {
			if reference.Reference.Type == "retweeted" {
				ignore = true
				break
			}
		}

		if ignore {
			continue
		}

		results = append(results, tweet)
	}

	return results, nil
}

func sendSlackNotificationForTwitterTweet(result *twitter.TweetDictionary, config *Config) error {
	if !config.NotifySlack {
		return nil
	}

	logFields := log.Fields{
		"tweet_id": result.Tweet.ID,
	}

	t, err := time.Parse(time.RFC3339, result.Tweet.CreatedAt)
	if err != nil {
		return err
	}

	link := fmt.Sprintf("https://twitter.com/%s/status/%s", result.Author.UserName, result.Tweet.ID)

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New tweet on Twitter!",
		AuthorName: result.Author.UserName,
		AuthorLink: fmt.Sprintf("https://twitter.com/%s", result.Author.UserName),
		Title:      result.Tweet.Text,
		TitleLink:  link,
		Footer:     "Twitter Tweet Notification",
		FooterIcon: twitterIconURL,
		Ts:         json.Number(strconv.FormatInt(int64(t.Unix()), 10)),
	}

	log.WithFields(logFields).Info("Notifying slack")
	messageOpts := []slack.MsgOption{
		slack.MsgOptionAsUser(false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionIconEmoji(":twitter:"),
		slack.MsgOptionText("New tweet on <"+link+"|Twitter>", false),
		slack.MsgOptionUsername("Twitter Tweet Notifications"),
		slack.MsgOptionDisableLinkUnfurl(),
	}

	api := slack.New(config.SlackToken)
	if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
		return err
	}

	return nil
}

func processTwitter(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&TwitterTweet{}); err != nil {
		return fmt.Errorf("error migrating TwitterTweet: %w", err)
	}

	if config.TwitterBearerToken == "" {
		log.Warn("No TWITTER_BEARER_TOKEN specified, skipping twitter")
		return nil
	}

	log.Info("Fetching tweets")
	results, err := getTweets(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("tweet_count", len(results)).Info("Processing tweets")
	for _, result := range results {
		logFields := log.Fields{
			"tweet_id": result.Tweet.ID,
		}

		var entity TwitterTweet
		if dbResult := db.First(&entity, "tweet_id = ?", result.Tweet.ID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new tweet")
		entity = TwitterTweet{
			TweetID: result.Tweet.ID,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting tweet into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForTwitterTweet(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting tweet to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_tweet_count": len(results),
		"inserted_tweet_count":  inserted,
		"notified_tweet_count":  notified,
	}).Info("Done with twitter tweets")

	return nil
}
