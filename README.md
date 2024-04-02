# social-notifications

A simple project that listens to specific words across the web and sends notifications about them to slack.

## Config

- `DATABASE_FILE`
- `LITESTREAM_ACCESS_KEY_ID`
- `LITESTREAM_REPLICA_URL`
- `LITESTREAM_SECRET_ACCESS_KEY`
- `LOG_FORMAT`
- `NOTIFY_SLACK`
- `RAPID_API_KEY`
- `SLACK_CHANNEL_ID`
- `SLACK_TOKEN`
- `TAG`

## Usage

```shell
# run for all configured services
social-notifications

# run for a single service
social-notifications --services github

# disable notifications (useful when building the database for the first time)
social-notifications --notify-slack=false
```

## Services

## Devto

Shows posts where the post has the tag.

![devto preview](/images/devto.png)

## Github

Shows results where the repository name contains the tag in the name.

![github preview](/images/github.png)

## Hacker News

Shows results where the story or comment has the tag in the contents, title, or url.

![hackernews preview](/images/hackernews-comment.png)

## Medium

Shows articles where the content has the tag.

![medium preview](/images/medium.png)

## Mastodon

Shows results from mastodon.social where the mastodon content has the tag.

![mastodon preview](/images/mastodon.png)

## Reddit

Shows results where the post is in a particular subreddit.

![reddit preview](/images/reddit.png)

## Stackoverflow

Shows questions where the question has the tag.

![stackoverflow preview](/images/stackoverflow.png)

## Twitter

Shows results where the tweet content has the tag. Has certain filtering conditions (see code for details).

![twitter preview](/images/twitter.png)
