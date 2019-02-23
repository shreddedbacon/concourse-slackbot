# Concourse SlackBot
This is a simple slack bot that can start jobs in Concourse.

# Run me
## BOSH Release
Use the [Concourse Slackbot BOSH Release](https://github.com/shreddedbacon/concourse-slackbot-boshrelease)

## Alternatively use Docker
Create `config.json` using `example-config.json` as a starter and modify to suit your slack and concourse setup.
```
# Build docker image with binary loaded and run it
make build-run

# Compile and output to builds/concoursebot and run builds/concoursebot
make build && make run
```

# Set up Slack
Create app [here](https://api.slack.com/apps)

Add a bot to the app

Edit the app permissions
* channels:history
* channels:read
* chat:write:bot
* groups:read
* users:read
* bot

Install to your workspace
