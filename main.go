package main

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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
		log.WithError(err).Fatal("error creating db")
	}

	if err := processStackoverflow(config, db); err != nil {
		log.WithError(err).Fatal("error processing stackoverflow")
	}
}
