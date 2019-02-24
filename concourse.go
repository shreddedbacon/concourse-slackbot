package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	/* Slack */
	"github.com/nlopes/slack"
)

func doConcourseTask(rtm *slack.RTM, msgChannel string, configuration Configuration, command int) {
	response := configuration.Commands[command].AcceptResponse
	rtm.SendMessage(rtm.NewOutgoingMessage(response, msgChannel))
	concourseTeam := configuration.Commands[command].Options.Team
	concoursePipeline := configuration.Commands[command].Options.Pipeline
	concourseJob := configuration.Commands[command].Options.Job
	concourseSkipoutput := configuration.Commands[command].Options.Skipoutput
	concourseUrl := configuration.ConcourseURL
	concourseUsername := configuration.ConcourseUsername
	concoursePassword := configuration.ConcoursePassword
	output, err := concourseRunJob(concourseTeam, concoursePipeline, concourseJob, concourseUrl, concourseUsername, concoursePassword, concourseSkipoutput)
	if err != nil {
		response = "```\n" +
			string(err.Error()) +
			"```"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msgChannel))
	} else {
		// If the output is going to be large, only show the last 2000 characters
		if len(output) > 2000 {
			output = output[len(output)-2000:]
		}
		response = "```\n" +
			output +
			"```"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msgChannel))
	}
}

// Connect to concourse, run the job, wait for it to finish then return the output
func concourseRunJob(concourseTeam string, concoursePipeline string, concourseJob string, concourseUrl string, concourseUsername string, concoursePassword string, concourseSkipoutput bool) (string, error) {
	authToken, authErr := concourseAuth(concourseUrl, concourseUsername, concoursePassword)
	if authErr != nil {
		return "", authErr
	} else {
		preCheckErr := concoursePreCheck(concourseTeam, concoursePipeline, concourseJob, concourseUrl, authToken)
		if preCheckErr != nil {
			return "", preCheckErr
		} else {
			triggerErr := concourseTrigger(concourseTeam, concoursePipeline, concourseJob, concourseUrl, authToken)
			if triggerErr != nil {
				return "", triggerErr
			} else {
				buildId, statusErr := concourseStatusCheck(concourseTeam, concoursePipeline, concourseJob, concourseUrl, authToken)
				if statusErr != nil {
					return "", statusErr
				} else {
					if concourseSkipoutput == false {
						buildOutput, buildErr := concourseGetEventLog(concoursePipeline, concourseJob, concourseUrl, authToken, buildId)
						if buildErr != nil {
							return "", buildErr
						} else {
							return buildOutput, nil
						}
					} else {
						return "Job output for `" + concoursePipeline + "/" + concourseJob + "` has been skipped.", nil
					}
				}
			}
		}
	}
}

func concourseAuth(concourseUrl string, concourseUsername string, concoursePassword string) (string, error) {
	cookieJar, _ := cookiejar.New(nil)
	var netClient = &http.Client{
		Timeout:       time.Second * 10,
		Jar:           cookieJar,
		CheckRedirect: redirectPolicyFunc,
	}
	var netClient2 = &http.Client{
		Timeout: time.Second * 10,
		Jar:     cookieJar,
	}
	req1, _ := http.NewRequest("POST", concourseUrl+"/sky/login", bytes.NewBuffer([]byte("")))
	resp1, _ := netClient.Do(req1)
	resploc, _ := resp1.Location()

	req2, _ := http.NewRequest("POST", resploc.String(), bytes.NewBuffer([]byte("")))
	resp2, _ := netClient.Do(req2)
	dump1, _ := httputil.DumpResponse(resp2, true)
	regex2 := regexp.MustCompile(`\/sky\/issuer\/auth\/local\?req=[a-z0-9]+`)
	matches2 := regex2.FindAllString(string(dump1), -1)

	data := url.Values{}
	data.Set("login", concourseUsername)
	data.Add("password", concoursePassword)
	req, _ := http.NewRequest("POST", concourseUrl+matches2[0], strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp, _ := netClient2.Do(req)
	c := cookieJar.Cookies(req.URL)
	if checkHttp200(resp.StatusCode) {
		return c[0].Value, nil
	} else {
		return "", errors.New("Failed to connect to CI\n\nProbably can't access the Concourse URL from where your bot is running")
	}
}

func concourseTrigger(concourseTeam string, concoursePipeline string, concourseJob string, concourseUrl string, authToken string) error {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("POST", concourseUrl+"/api/v1/teams/"+concourseTeam+"/pipelines/"+concoursePipeline+"/jobs/"+concourseJob+"/builds", bytes.NewBuffer([]byte("")))
	req.Header.Add("Authorization", authToken)
	resp, _ := netClient.Do(req)
	if checkHttp200(resp.StatusCode) {
		return nil
	} else {
		return errors.New("Failed to connect to CI\n\nEither Team, Pipeline, or Job don't exist. You should check.")
	}
}

func concoursePreCheck(concourseTeam string, concoursePipeline string, concourseJob string, concourseUrl string, authToken string) error {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	statusResp := &ConcourseStatus{}
	// poll job for succeeded
	req, _ := http.NewRequest("GET", concourseUrl+"/api/v1/teams/"+concourseTeam+"/pipelines/"+concoursePipeline+"/jobs/"+concourseJob+"/builds/latest", nil)
	req.Header.Add("Authorization", authToken)
	resp, _ := netClient.Do(req)
	if checkHttp200(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		if err := json.Unmarshal([]byte(body), statusResp); err != nil {
			return err
		}
		resp.Body.Close()
		if statusResp.Status == "succeeded" || statusResp.Status == "failed" || statusResp.Status == "aborted" {
			return nil
		} else {
			return errors.New("A job for `" + concoursePipeline + "/" + concourseJob + "` is already running, try again soon")
		}
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		if string(body) == "" {
			return nil
		} else {
			return errors.New("Failed to establish connection to CI - Job was queued and may or may not be running still\n\nDo not run this again unless you are sure")
		}
	}
	return errors.New("Something went wrong")
}

func concourseStatusCheck(concourseTeam string, concoursePipeline string, concourseJob string, concourseUrl string, authToken string) (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	statusResp := &ConcourseStatus{}
	// poll job for succeeded
	for {
		req, _ := http.NewRequest("GET", concourseUrl+"/api/v1/teams/"+concourseTeam+"/pipelines/"+concoursePipeline+"/jobs/"+concourseJob+"/builds/latest", nil)
		req.Header.Add("Authorization", authToken)
		resp, _ := netClient.Do(req)
		if checkHttp200(resp.StatusCode) {
			body, _ := ioutil.ReadAll(resp.Body)
			if err := json.Unmarshal([]byte(body), statusResp); err != nil {
				return "", err
			}
			resp.Body.Close()
			if statusResp.Status == "succeeded" {
				buildid := strconv.Itoa(statusResp.ID)
				return buildid, nil
			}
			if statusResp.Status == "failed" {
				return "", errors.New("Job failed, see " + concourseUrl + "/teams/" + concourseTeam + "/pipelines/" + concoursePipeline + "/jobs/" + concourseJob + "/builds/latest")
			}
			if statusResp.Status == "aborted" {
				return "", errors.New("Job aborted, see " + concourseUrl + "/teams/" + concourseTeam + "/pipelines/" + concoursePipeline + "/jobs/" + concourseJob + "/builds/latest")
			}
		} else {
			return "", errors.New("Failed to establish connection to CI - Job was queued and may or may not be running still\n\nDo not run this again unless you are sure")
		}
		time.Sleep(5 * time.Second)
	}
	return "", errors.New("Something went wrong")
}

func concourseGetEventLog(concoursePipeline string, concourseJob string, concourseUrl string, authToken string, buildId string) (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("GET", concourseUrl+"/api/v1/builds/"+buildId+"/events", nil)
	req.Header.Add("Authorization", authToken)
	resp, _ := netClient.Do(req)
	if checkHttp200(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		regex := regexp.MustCompile(`\{\s*"data"\s*:\s*(.+?)\s*,\s*"event"\s*:\s*(.+?)\s*\}`)
		matches := regex.FindAllString(string(body), -1)
		output := ""
		for _, v := range matches {
			payloadData := &ConcourseEvent{
				Data: &Data{},
			}
			err := json.Unmarshal([]byte(v), payloadData)
			if err != nil {
			}
			output = output + payloadData.Data.Payload
		}
		return output, nil
	} else {
		return "", errors.New("Failed to establish connection to CI - Job may or may not have finished, but failed to connect to get the results\n\nYou should check your Concourse is not broken")
	}
}
