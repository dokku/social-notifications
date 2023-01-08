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
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type Question struct {
	ID    int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	Title string `gorm:"not null" form:"title" json:"title"`
}

var stackoverflowIconURL = "https://emoji.slack-edge.com/T085AJH3L/stackoverflow/35cab7f857fa4681.png"

func getQuestions(config *Config) ([]stackoverflow.Question, error) {
	questions, err := util.GetQuestionsAll(nil, config.Site, &stackoverflow.GetQuestionsOpts{
		Tagged:   optional.NewString(config.Tag),
		Page:     optional.NewInt32(1),
		Pagesize: optional.NewInt32(int32(util.PerPageMax)),
		Sort:     optional.NewString("creation"),
		Order:    optional.NewString("asc"),
	})
	if err != nil {
		return questions, fmt.Errorf("error fetching questions from stackoverflow: %w", err)
	}

	return questions, nil
}

func sendSlackNotificationForStackoverflow(question stackoverflow.Question, config *Config) error {
	logFields := log.Fields{
		"question_id": question.QuestionId,
		"title":       question.Title,
	}

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
			return err
		}
	}

	return nil
}

func processStackoverflow(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&Question{}); err != nil {
		return fmt.Errorf("error migrating Question: %w", err)
	}

	questions, err := getQuestions(config)
	if err != nil {
		return err
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
		if err := sendSlackNotificationForStackoverflow(question, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting question to slack")
			continue
		}

		notifiedQuestions += 1
	}
	log.WithFields(log.Fields{
		"processed_question_count": len(questions),
		"inserted_question_count":  insertedQuestions,
		"notified_question_count":  notifiedQuestions,
	}).Info("Done with stackoverflow")

	return nil
}
