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

type GithubRepository struct {
	ID           int32  `gorm:"AUTO_INCREMENT" form:"id" json:"id"`
	RepositoryID int64  `gorm:"not null" form:"repository_id" json:"repository_id"`
	Title        string `gorm:"not null" form:"title" json:"title"`
}

type GithubResponse struct {
	TotalCount        int                    `json:"total_count"`
	IncompleteResults bool                   `json:"incomplete_results"`
	Items             []GithubRepositoryItem `json:"items"`
}

type GithubRepositoryItem struct {
	ID                       int            `json:"id"`
	NodeID                   string         `json:"node_id"`
	Name                     string         `json:"name"`
	FullName                 string         `json:"full_name"`
	Private                  bool           `json:"private"`
	Owner                    GithubUserItem `json:"owner"`
	HTMLURL                  string         `json:"html_url"`
	Description              interface{}    `json:"description"`
	Fork                     bool           `json:"fork"`
	URL                      string         `json:"url"`
	ForksURL                 string         `json:"forks_url"`
	KeysURL                  string         `json:"keys_url"`
	CollaboratorsURL         string         `json:"collaborators_url"`
	TeamsURL                 string         `json:"teams_url"`
	HooksURL                 string         `json:"hooks_url"`
	IssueEventsURL           string         `json:"issue_events_url"`
	EventsURL                string         `json:"events_url"`
	AssigneesURL             string         `json:"assignees_url"`
	BranchesURL              string         `json:"branches_url"`
	TagsURL                  string         `json:"tags_url"`
	BlobsURL                 string         `json:"blobs_url"`
	GitTagsURL               string         `json:"git_tags_url"`
	GitRefsURL               string         `json:"git_refs_url"`
	TreesURL                 string         `json:"trees_url"`
	StatusesURL              string         `json:"statuses_url"`
	LanguagesURL             string         `json:"languages_url"`
	StargazersURL            string         `json:"stargazers_url"`
	ContributorsURL          string         `json:"contributors_url"`
	SubscribersURL           string         `json:"subscribers_url"`
	SubscriptionURL          string         `json:"subscription_url"`
	CommitsURL               string         `json:"commits_url"`
	GitCommitsURL            string         `json:"git_commits_url"`
	CommentsURL              string         `json:"comments_url"`
	IssueCommentURL          string         `json:"issue_comment_url"`
	ContentsURL              string         `json:"contents_url"`
	CompareURL               string         `json:"compare_url"`
	MergesURL                string         `json:"merges_url"`
	ArchiveURL               string         `json:"archive_url"`
	DownloadsURL             string         `json:"downloads_url"`
	IssuesURL                string         `json:"issues_url"`
	PullsURL                 string         `json:"pulls_url"`
	MilestonesURL            string         `json:"milestones_url"`
	NotificationsURL         string         `json:"notifications_url"`
	LabelsURL                string         `json:"labels_url"`
	ReleasesURL              string         `json:"releases_url"`
	DeploymentsURL           string         `json:"deployments_url"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	PushedAt                 time.Time      `json:"pushed_at"`
	GitURL                   string         `json:"git_url"`
	SSHURL                   string         `json:"ssh_url"`
	CloneURL                 string         `json:"clone_url"`
	SvnURL                   string         `json:"svn_url"`
	Homepage                 interface{}    `json:"homepage"`
	Size                     int            `json:"size"`
	StargazersCount          int            `json:"stargazers_count"`
	WatchersCount            int            `json:"watchers_count"`
	Language                 string         `json:"language"`
	HasIssues                bool           `json:"has_issues"`
	HasProjects              bool           `json:"has_projects"`
	HasDownloads             bool           `json:"has_downloads"`
	HasWiki                  bool           `json:"has_wiki"`
	HasPages                 bool           `json:"has_pages"`
	HasDiscussions           bool           `json:"has_discussions"`
	ForksCount               int            `json:"forks_count"`
	MirrorURL                interface{}    `json:"mirror_url"`
	Archived                 bool           `json:"archived"`
	Disabled                 bool           `json:"disabled"`
	OpenIssuesCount          int            `json:"open_issues_count"`
	License                  interface{}    `json:"license"`
	AllowForking             bool           `json:"allow_forking"`
	IsTemplate               bool           `json:"is_template"`
	WebCommitSignoffRequired bool           `json:"web_commit_signoff_required"`
	Topics                   []interface{}  `json:"topics"`
	Visibility               string         `json:"visibility"`
	Forks                    int            `json:"forks"`
	OpenIssues               int            `json:"open_issues"`
	Watchers                 int            `json:"watchers"`
	DefaultBranch            string         `json:"default_branch"`
	Score                    float64        `json:"score"`
}

type GithubUserItem struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

func getGithubRepositories(config *Config) ([]GithubRepositoryItem, error) {
	var results []GithubRepositoryItem
	page := 1
	for {
		log.WithField("page", page).Info("Fetching page")
		var response GithubResponse
		client := resty.New()
		_, err := client.R().
			SetQueryParams(map[string]string{
				"q":        config.Tag,
				"per_page": "100",
				"page":     strconv.FormatInt(int64(page), 10),
				"sort":     "updated",
			}).
			SetResult(&response).
			Get("https://api.github.com/search/repositories")
		if err != nil {
			return results, err
		}

		page += 1
		if len(response.Items) == 0 {
			break
		}

		results = append(results, response.Items...)
	}

	return results, nil
}

var githubIconUrl = "https://emoji.slack-edge.com/T085AJH3L/github/eeab46c8e8ba02f7.png"

func sendSlackNotificationForGithubRepository(result GithubRepositoryItem, config *Config) error {
	logFields := log.Fields{
		"repository_id": result.ID,
	}

	fields := []slack.AttachmentField{
		{
			Title: "Language",
			Value: result.Language,
			Short: true,
		},
	}

	attachment := slack.Attachment{
		Color:      "#36a64f",
		Fallback:   "New repository on Github!",
		AuthorName: result.Owner.Login,
		AuthorIcon: result.Owner.AvatarURL,
		AuthorLink: result.Owner.HTMLURL,
		Title:      result.FullName,
		TitleLink:  result.HTMLURL,
		Footer:     "Github Repository Notification",
		FooterIcon: githubIconUrl,
		Ts:         json.Number(strconv.FormatInt(int64(result.CreatedAt.Unix()), 10)),
		Fields:     fields,
	}

	if config.NotifySlack {
		log.WithFields(logFields).Info("Notifying slack")
		messageOpts := []slack.MsgOption{
			slack.MsgOptionAsUser(false),
			slack.MsgOptionAttachments(attachment),
			slack.MsgOptionIconEmoji(":github:"),
			slack.MsgOptionText("New repository on <"+result.HTMLURL+"|Github>", false),
			slack.MsgOptionUsername("Github Repository Notifications"),
			slack.MsgOptionDisableLinkUnfurl(),
		}

		api := slack.New(config.SlackToken)
		if _, _, err := api.PostMessage(config.SlackChannelID, messageOpts...); err != nil {
			return err
		}
	}

	return nil
}

func processGithubRepositories(config *Config, db *gorm.DB) error {
	if err := db.AutoMigrate(&GithubRepository{}); err != nil {
		return fmt.Errorf("error migrating GithubRepository: %w", err)
	}

	log.Info("Fetching repositories")
	results, err := getGithubRepositories(config)
	if err != nil {
		return err
	}

	inserted := 0
	notified := 0
	log.WithField("repository_count", len(results)).Info("Processing questions")
	for _, result := range results {
		logFields := log.Fields{
			"repository_id": result.ID,
			"title":         result.FullName,
		}

		var entity GithubRepository
		if dbResult := db.First(&entity, "repository_id = ?", result.ID); !errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
			continue
		}

		log.WithFields(logFields).Info("Inserting new repository")
		entity = GithubRepository{
			RepositoryID: int64(result.ID),
			Title:        result.FullName,
		}

		if dbResult := db.Create(&entity); dbResult.Error != nil {
			log.WithError(dbResult.Error).WithFields(logFields).Fatal("error inserting repository into database")
			continue
		}

		inserted += 1
		if err := sendSlackNotificationForGithubRepository(result, config); err != nil {
			log.WithError(err).WithFields(logFields).Fatal("error posting repository to slack")
			continue
		}

		notified += 1
	}
	log.WithFields(log.Fields{
		"processed_repository_count": len(results),
		"inserted_repository_count":  inserted,
		"notified_repository_count":  notified,
	}).Info("Done with github repositories")

	return nil
}
