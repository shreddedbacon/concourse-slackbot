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

func doConcourseTask(rtm *slack.RTM, msg *slack.MessageEvent, flyurl string, conuser string, conpass string, team string, pipeline string, job string, response string, skipoutput bool) {
	rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	output, err := concourseRunJob(team, pipeline, job, flyurl, conuser, conpass, skipoutput)
	if err != nil {
		response = "```\n" +
			string(err.Error()) +
			"```"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	} else {
		response = "```\n" +
			output +
			"```"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	}
}

// Connect to concourse, run the job, wait for it to finish then return the output
func concourseRunJob(team string, pipeline string, job string, flyurl string, conuser string, conpass string, skipoutput bool) (string, error) {
	authtoken, autherr := concourseAuth(flyurl, conuser, conpass)
	if autherr != nil {
		return "", autherr
	} else {
		precheckerr := concoursePreCheck(team, pipeline, job, flyurl, authtoken)
		if precheckerr != nil {
			return "", precheckerr
		} else {
			triggererr := concourseTrigger(team, pipeline, job, flyurl, authtoken)
			if triggererr != nil {
				return "", triggererr
			} else {
				buildid, statuserr := concourseStatusCheck(team, pipeline, job, flyurl, authtoken)
				if statuserr != nil {
					return "", statuserr
				} else {
					if skipoutput == false {
						buildoutput, builderr := concourseGetEventLog(pipeline, job, flyurl, authtoken, buildid)
						if builderr != nil {
							return "", builderr
						} else {
							return buildoutput, nil
						}
					} else {
						return "Job output for `" + pipeline + "/" + job + "` has been skipped.", nil
					}
				}
			}
		}
	}
}


func concourseAuth(flyurl string, conuser string, conpass string) (string, error) {
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
	req1, _ := http.NewRequest("POST", flyurl+"/sky/login", bytes.NewBuffer([]byte("")))
	resp1, _ := netClient.Do(req1)
	resploc, _ := resp1.Location()

	req2, _ := http.NewRequest("POST", resploc.String(), bytes.NewBuffer([]byte("")))
	resp2, _ := netClient.Do(req2)
	dump1, _ := httputil.DumpResponse(resp2, true)
	regex2 := regexp.MustCompile(`\/sky\/issuer\/auth\/local\?req=[a-z0-9]+`)
	matches2 := regex2.FindAllString(string(dump1), -1)

	data := url.Values{}
	data.Set("login", conuser)
	data.Add("password", conpass)
	req, _ := http.NewRequest("POST", flyurl+matches2[0], strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp, _ := netClient2.Do(req)
	c := cookieJar.Cookies(req.URL)
	if check200(resp.StatusCode) {
		return c[0].Value, nil
	} else {
		return "", errors.New("Failed to connect to CI\n\nProbably can't access the Concourse URL from where your bot is running")
	}
}

func concourseTrigger(team string, pipeline string, job string, flyurl string, authtoken string) error {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("POST", flyurl+"/api/v1/teams/"+team+"/pipelines/"+pipeline+"/jobs/"+job+"/builds", bytes.NewBuffer([]byte("")))
	req.Header.Add("Authorization", authtoken)
	resp, _ := netClient.Do(req)
	if check200(resp.StatusCode) {
		return nil
	} else {
		return errors.New("Failed to connect to CI\n\nEither Team, Pipeline, or Job don't exist. You should check.")
	}
}

func concoursePreCheck(team string, pipeline string, job string, flyurl string, authtoken string) error {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	statusresp := &ConcourseStatus{}
	// poll job for succeeded
	req, _ := http.NewRequest("GET", flyurl+"/api/v1/teams/"+team+"/pipelines/"+pipeline+"/jobs/"+job+"/builds/latest", nil)
	req.Header.Add("Authorization", authtoken)
	resp, _ := netClient.Do(req)
	if check200(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		if err := json.Unmarshal([]byte(body), statusresp); err != nil {
			return err
		}
		resp.Body.Close()
		if statusresp.Status == "succeeded" || statusresp.Status == "failed" || statusresp.Status == "aborted" {
			return nil
		} else {
			return errors.New("A job for `" + pipeline + "/" + job + "` is already running, try again soon")
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

func concourseStatusCheck(team string, pipeline string, job string, flyurl string, authtoken string) (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	statusresp := &ConcourseStatus{}
	// poll job for succeeded
	for {
		req, _ := http.NewRequest("GET", flyurl+"/api/v1/teams/"+team+"/pipelines/"+pipeline+"/jobs/"+job+"/builds/latest", nil)
		req.Header.Add("Authorization", authtoken)
		resp, _ := netClient.Do(req)
		if check200(resp.StatusCode) {
			body, _ := ioutil.ReadAll(resp.Body)
			if err := json.Unmarshal([]byte(body), statusresp); err != nil {
				return "", err
			}
			resp.Body.Close()
			if statusresp.Status == "succeeded" {
				buildid := strconv.Itoa(statusresp.ID)
				return buildid, nil
			}
			if statusresp.Status == "failed" {
				return "", errors.New("Job failed, see " + flyurl + "/teams/" + team + "/pipelines/" + pipeline + "/jobs/" + job + "/builds/latest")
			}
			if statusresp.Status == "aborted" {
				return "", errors.New("Job aborted, see " + flyurl + "/teams/" + team + "/pipelines/" + pipeline + "/jobs/" + job + "/builds/latest")
			}
		} else {
			return "", errors.New("Failed to establish connection to CI - Job was queued and may or may not be running still\n\nDo not run this again unless you are sure")
		}
		time.Sleep(5 * time.Second)
	}
	return "", errors.New("Something went wrong")
}

func concourseGetEventLog(pipeline string, job string, flyurl string, authtoken string, buildid string) (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	req, _ := http.NewRequest("GET", flyurl+"/api/v1/builds/"+buildid+"/events", nil)
	req.Header.Add("Authorization", authtoken)
	resp, _ := netClient.Do(req)
	if check200(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		regex := regexp.MustCompile(`\{\s*"data"\s*:\s*(.+?)\s*,\s*"event"\s*:\s*(.+?)\s*\}`)
		matches := regex.FindAllString(string(body), -1)
		output := ""
		for _, v := range matches {
			payloaddata := &ConcourseEvent{
				Data: &Data{},
			}
			err := json.Unmarshal([]byte(v), payloaddata)
			if err != nil {
			}
			output = output + payloaddata.Data.Payload
		}
		return output, nil
	} else {
		return "", errors.New("Failed to establish connection to CI - Job may or may not have finished, but failed to connect to get the results\n\nYou should check your Concourse is not broken")
	}
}
