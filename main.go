package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/antihax/optional"
	stackoverflow "github.com/grokify/go-stackoverflow/client"
	"github.com/grokify/go-stackoverflow/util"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Question struct {
	ID    int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	Title string `gorm:"not null" form:"title" json:"title"`
}

type Config struct {
	DatabaseFile    string `required:"false" split_words:"true"`
	LogFormat       string `required:"false" split_words:"true"`
	NotifySlack     bool   `required:"false"`
	Site            string `required:"false"`
	SlackChannel    string `required:"true" split_words:"true"`
	SlackUsername   string `required:"true" split_words:"true"`
	SlackWebhookURL string `required:"true" split_words:"true"`
	Tag             string `required:"true"`
}

var stackoverflowIconURL = "https://emoji.slack-edge.com/T085AJH3L/stackoverflow/35cab7f857fa4681.png"

func LoadConfig() *Config {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		panic(err)
	}

	return &config
}
func CreateDB(databaseFile string) (*gorm.DB, error) {
	if databaseFile == "" {
		databaseFile = "stackoverflow-notifications.db"
	}

	db, err := gorm.Open(sqlite.Open(databaseFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return db, err
	}

	if err := db.AutoMigrate(&Question{}); err != nil {
		return db, fmt.Errorf("error migrating Question: %w", err)
	}
	return db, nil
}

func main() {
	config := LoadConfig()
	if config.LogFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if config.Tag == "" {
		log.Fatal("No TAG environment variable specified")
	}

	db, err := CreateDB(config.DatabaseFile)
	if err != nil {
		log.WithError(err).Fatal("error fetching questions from stackoverflow")
	}

	questions, err := util.GetQuestionsAll(nil, config.Site, &stackoverflow.GetQuestionsOpts{
		Tagged:   optional.NewString(config.Tag),
		Page:     optional.NewInt32(1),
		Pagesize: optional.NewInt32(int32(util.PerPageMax)),
		Sort:     optional.NewString("creation"),
		Order:    optional.NewString("asc"),
	})
	if err != nil {
		log.WithError(err).Fatal("error fetching questions from stackoverflow")
	}

	insertedQuestions := 0
	notifiedQuestions := 0
	log.WithField("question_count", len(questions)).Info("Processing questions")
	for _, question := range questions {
		logFields := log.Fields{
			"question_id": question.QuestionId,
			"title":       question.Title,
		}

		var entity Question
		result := db.First(&entity, "id = ?", question.QuestionId)
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new question")
		entity = Question{
			ID:    question.QuestionId,
			Title: question.Title,
		}

		if result := db.Create(&entity); result.Error != nil {
			log.WithError(result.Error).WithFields(logFields).Fatal("error inserting question into database")
			continue
		}

		insertedQuestions += 1
		answered := "✅"
		if !question.IsAnswered {
			answered = "🚫"
		}

		attachment := slack.Attachment{
			Color:      "#36a64f",
			Fallback:   "New question on StackOverflow!",
			AuthorName: question.Owner.DisplayName,
			AuthorLink: question.Owner.Link,
			AuthorIcon: question.Owner.ProfileImage,
			Title:      question.Title,
			TitleLink:  question.Link,
			Footer:     "Stackoverflow Notification",
			FooterIcon: stackoverflowIconURL,
			Ts:         json.Number(strconv.FormatInt(int64(question.CreationDate), 10)),
			Fields: []slack.AttachmentField{
				{
					Title: "# Views",
					Value: strconv.FormatInt(int64(question.ViewCount), 10),
					Short: true,
				},
				{
					Title: "# Answers",
					Value: strconv.FormatInt(int64(question.AnswerCount), 10),
					Short: true,
				},
				{
					Title: "Answered",
					Value: answered,
					Short: true,
				},
				{
					Title: "Tags",
					Value: strings.Join(question.Tags, ", "),
					Short: true,
				},
			},
		}
		msg := slack.WebhookMessage{
			Channel:     config.SlackChannel,
			IconURL:     stackoverflowIconURL,
			Username:    config.SlackUsername,
			Text:        "New question on <" + question.Link + "|StackOverflow>",
			Attachments: []slack.Attachment{attachment},
		}

		if config.NotifySlack {
			log.WithFields(logFields).Info("Notifying slack")
			if err := slack.PostWebhook(config.SlackWebhookURL, &msg); err != nil {
				log.WithError(err).WithFields(logFields).Fatal("error posting question to slack")
				continue
			}
		}

		notifiedQuestions += 1
	}
	log.WithFields(log.Fields{
		"processed_question_count": len(questions),
		"inserted_question_count":  insertedQuestions,
		"notified_question_count":  notifiedQuestions,
	}).Info("Done")
}
