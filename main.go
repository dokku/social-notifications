package main

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DatabaseFile   string   `required:"false" split_words:"true"`
	LogFormat      string   `required:"false" split_words:"true"`
	NotifySlack    bool     `required:"false" split_words:"true"`
	Services       []string `required:"false" split_words:"true"`
	Site           string   `required:"false" split_words:"true"`
	SlackChannelID string   `required:"true" split_words:"true"`
	SlackToken     string   `required:"true" split_words:"true"`
	Tag            string   `required:"true" split_words:"true"`
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

type processor func(*Config, *gorm.DB) error

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

	services := map[string]processor{
		"stackoverflow":    processStackoverflow,
		"hackernews_story": processHackernewsStories,
	}

	// allow disabling services
	if len(config.Services) > 0 {
		enabledServices := map[string]bool{}
		for _, service := range config.Services {
			enabledServices[service] = true
		}

		disabledServices := []string{}
		for service := range services {
			if !enabledServices[service] {
				disabledServices = append(disabledServices, service)
			}
		}

		for _, service := range disabledServices {
			log.WithField("service", service).Info("Disabling service")
			delete(services, service)
		}
	}

	for service, processor := range services {
		log.WithField("service", service).Info("Processing service")
		if err := processor(config, db); err != nil {
			log.WithError(err).WithField("service", service).Fatal("error processing")
		}
	}
}
