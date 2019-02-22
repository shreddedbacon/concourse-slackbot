package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	/* Slack */
	"github.com/nlopes/slack"
)

func main() {
	// Load configuration
	configuration := Configuration{}
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Successfully read config.json file")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	configJson, _ := ioutil.ReadAll(jsonFile)
	errJson := json.Unmarshal(configJson, &configuration)
	if errJson != nil {
		log.Fatalln("error:", errJson)
	}

	flyurl := configuration.ConcourseURL
	conuser := configuration.ConcourseUsername
	conpass := configuration.ConcoursePassword
	token := configuration.SlackToken
	startupchannel := configuration.SlackStartChannel
	startupmessage := configuration.SlackStartMessage
	privusers := configuration.PrivilegedUsers

	if flyurl == "" {
		log.Fatalln("concourse_url not set")
	}
	if conuser == "" {
		log.Fatalln("concourse_username not set")
	}
	if conpass == "" {
		log.Fatalln("concourse_password not set")
	}
	if token == "" {
		log.Fatalln("slack_token not set")
	}
	if startupchannel == "" {
		log.Fatalln("slack_start_channel not set")
	}
	if startupmessage == "" {
		log.Fatalln("slack_start_message not set")
	}
	if len(privusers) == 0 {
		log.Fatalln("privileged_users not set")
	}
	api := slack.New(token)
	if configuration.Debug {
		api.SetDebug(true)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			fmt.Println("Connection counter:", ev.ConnectionCount)
			time.Sleep(1000 * time.Millisecond) //sleep for one second before trying to send a message to the channel
			rtm.SendMessage(rtm.NewOutgoingMessage(startupmessage, startupchannel))
		case *slack.MessageEvent:
			info := rtm.GetInfo()
			prefix := fmt.Sprintf("<@%s> ", info.User.ID)
			if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
				go respond(rtm, ev, prefix, api, flyurl, conuser, conpass, privusers, configuration)
			}
			/* eggs */
			obiwan := strings.ToLower(ev.Text)
			if obiwan == "obiwan" {
				response := "*beep boop*"
				rtm.SendMessage(rtm.NewOutgoingMessage(response, ev.Channel))
			}
		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())
		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			break
		default:
			//Take no action
		}
	}
}

/* function to check if string array contains a string */
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func check200(httpcode int) bool {
	if httpcode == 200 {
		return true
	} else {
		return false
	}
}

func redirectPolicyFunc(r *http.Request, rr []*http.Request) error {
	return errors.New("disable")
}

/* function to respond to message event in slack */
func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string, api *slack.Client, flyurl string, conuser string, conpass string, privusers []string, configuration Configuration) {
	var response string
	text := msg.Text
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	user, err := api.GetUserInfo(msg.User)

	if err != nil {
		fmt.Printf("%s\n", err)
	}
	rand.Seed(time.Now().Unix())
	privilegedusers := privusers
	// Respond with a random quote to unknown commands to @bot
	quotes := configuration.Quotes
	n := rand.Int() % len(quotes)
	// Create a list of privileged users to respond to requests that need it with who to ask
	askthem := ""
	comma := ""
	for i, v := range privilegedusers {
		if i == 0 {
			comma = ""
		} else {
			comma = ","
		}
		askthem = askthem + comma + "<@" + v + ">"
	}

	switch text {
	case "good bot":
		response = "I'm trying"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))

	/* example pass response from concourse build output to user that matches regex */
	/*
	case "respond user":
		response = "Got it, I'll send you a preprod login soon, make sure you connect to the VPN first :)"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
		output, err := concourseRunJob(team, pipeline, job, flyurl, conuser, conpass, false)
		if err != nil {
			response = "```\n" +
				string(err.Error()) +
				"```"
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
		} else {
	     // regex to check output for
			regex := regexp.MustCompile(`(http).+?(login)`)
			matches := regex.FindAllString(output, -1)
			response := "Here you go!\n"
			_, _, channelID, err := api.OpenIMChannel(user.ID)
			if err != nil {
				fmt.Printf("%s\n", err)
			}
			params := slack.PostMessageParameters{}
			attachment := slack.Attachment{
				Pretext:    "`" + strings.Join(matches2, "") + "`",
				MarkdownIn: []string{"pretext"},
			}
			params.Attachments = []slack.Attachment{attachment}
			api.PostMessage(channelID, response, params)
		}
  */

	case "do i have permission?":
		if !contains(privilegedusers, string(user.Name)) {
			response = string(user.Name) + ", you are not a keymaster? \n*maybe ask " + askthem + "*"
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
		} else {
			response = "You are a keymaster, " + string(user.Name)
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
		}

	case "help":
		generatedHelp := ""
		for i := range configuration.Commands {
			commandStr := "@concoursebot " + string(configuration.Commands[i].Command)
			numSpaces := 80 - len(commandStr)
			spaces := ""
			for i := 0; i < numSpaces; i++ {
				spaces += " "
			}
			generatedHelp += string(commandStr) + string(spaces) + ": " + configuration.Commands[i].Help + "\n"
		}
		commandStr := "@concoursebot do i have permission?"
		numSpaces := 80 - len(commandStr)
		spaces := ""
		for i := 0; i < numSpaces; i++ {
			spaces += " "
		}
		generatedHelp += string(commandStr) + string(spaces) + ": check if you have permission to do higher level tasks\n"
		response = ">>>Command list:\n" +
			"```\n" +
			generatedHelp +
			"```"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))

	default:
		var randomQuote bool
		randomQuote = true
		for i := range configuration.Commands {
			if configuration.Commands[i].Command == text {
				switch configuration.Commands[i].Type {
				case "concourse":
					randomQuote = false
					if configuration.Commands[i].Options.Privileged == true {
						if !contains(privilegedusers, string(user.Name)) {
							response = "I can't let you do that, Dave. \n*maybe ask " + askthem + "*"
							rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
						} else {
							doConcourseTask(rtm, msg, flyurl, conuser, conpass, configuration.Commands[i].Options.Team, configuration.Commands[i].Options.Pipeline, configuration.Commands[i].Options.Job, configuration.Commands[i].AcceptResponse, configuration.Commands[i].Options.Skipoutput)
						}
					} else {
						doConcourseTask(rtm, msg, flyurl, conuser, conpass, configuration.Commands[i].Options.Team, configuration.Commands[i].Options.Pipeline, configuration.Commands[i].Options.Job, configuration.Commands[i].AcceptResponse, configuration.Commands[i].Options.Skipoutput)
					}
				}
			}
		}
		if randomQuote == true {
			response = quotes[n]
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
		}
	}
}
