package main

type AuthToken struct {
	Type  string `json:"type"`
	Token string `json:"value"`
}

type ConcourseStatus struct {
	Status string `json:"status"`
	ID     int    `json:"id"`
}

type ConcourseEvent struct {
	Data  *Data  `json:"data"`
	Event string `json:"log"`
}

type Data struct {
	Payload string `json:"payload"`
}

type Configuration struct {
	SlackToken        string   `json:"slack_token"`
	SlackBotName      string   `json:"slack_bot_name"`
	SlackStartChannel string   `json:"slack_start_channel"`
	SlackStartMessage string   `json:"slack_start_message"`
	ConcourseURL      string   `json:"concourse_url"`
	ConcourseUsername string   `json:"concourse_username"`
	ConcoursePassword string   `json:"concourse_password"`
	Debug             bool     `json:"debug"`
	Quotes            []string `json:"quotes"`
	Commands          []struct {
		Command        string `json:"command"`
		Type           string `json:"type"`
		Help           string `json:"help"`
		AcceptResponse string `json:"accept_response"`
		Options        struct {
			Team       string `json:"team,omitempty"`
			Pipeline   string `json:"pipeline,omitempty"`
			Job        string `json:"job,omitempty"`
			Skipoutput bool   `json:"skipoutput,omitempty"`
			Privileged bool   `json:"privileged,omitempty"`
		} `json:"options,omitempty"`
  	PrivilegedUsers   []string `json:"privileged_users"`
	} `json:"commands"`
}
