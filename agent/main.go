package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// Wrapper class that takes care of all boilerplate required for agent session.
type Client struct {
	baseURL string
	authHeader string
	h http.Client
	SupportedTaskTypes map[string]bool
}

func NewClient(baseURL string) Client {
	return Client{baseURL: baseURL, h: http.Client{}}
}

func (c *Client) RegisterAgent(user, pass string) error {
	// It's not necessary to do GET /agents_selfreg, server will reject request
	// anyway if registration is disabled.
	req, err := http.NewRequest("POST", c.baseURL + "/agents?user=" + url.QueryEscape(user) + "&pass=" + url.QueryEscape(pass), nil)
	if err != nil {
		return fmt.Errorf("request create: %v", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode / 100 != 2 { // check for non 2xx code, not just 200.
		return errors.New(errorMessage(resp))
	}
	return nil
}

func (c *Client) UseAccount(user, pass string) {
	c.authHeader = user + ":" + pass
}

// PollTasks requests first task from server's queue.
//
// It may block for up to 26 seconds. And also note that it returns error for tasks
// with type not in SupportedTaskTypes (if SupportTaskTypes is not nil).
func (c *Client) PollTasks() (id int, type_ string, body map[string]interface{}, err error) {
	req, err := http.NewRequest("GET", c.baseURL + "/tasks", nil)
	if err != nil {
		return -1, "", nil, fmt.Errorf("request create: %v", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	resp, err := c.h.Do(req)
	if err != nil {
		return -1, "", nil, err
	}
	if resp.StatusCode / 100 != 2 { // check for non 2xx code, not just 200.
		return -1, "", nil, errors.New(errorMessage(resp))
	}

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, "", nil, fmt.Errorf("response body read: %v", err)
	}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return -1, "", nil, fmt.Errorf("response body parse: %v", err)
	}

	if body["id"] == nil {
		return -1, "", body, errors.New("missing id field in response")
	}
	var ok bool
	id, ok = body["id"].(int)
	if !ok {
		return -1, "", body, errors.New("non-numeric task ID")
	}

	if body["type"] == nil {
		return id, "", body, errors.New("missing task type in response")
	}
	type_, ok = body["type"].(string)
	if !ok {
		return id, "", body, errors.New("non-string task type")
	}

	if c.SupportedTaskTypes != nil {
		if _, prs := c.SupportedTaskTypes[type_]; !prs {
			return id, type_, body, errors.New("unsupported task type")
		}
	}

	return
}

func (c *Client) SendTaskResult(taskID int, result map[string]interface{}) error {
	resJson, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("json format: %v", err)
	}

	if _, prs := result["error"]; !prs {
		result["error"] = false
	}

	req, err := http.NewRequest("POST", c.baseURL + "/task_result?id=" + strconv.Itoa(taskID), bytes.NewReader(resJson))
	if err != nil {
		return fmt.Errorf("request create: %v", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode / 100 != 2 { // check for non 2xx code, not just 200.
		return errors.New(errorMessage(resp))
	}
	return nil
}

func errorMessage(resp *http.Response) string {
	// We have two cases to handle:
	// - Error at intermediate level (nginx)
	//   This probably means dutserver is down or something. We have only status code in this case.
	// - Error at dutserver level
	//   We have error message in JSON.

	if resp.Header.Get("Content-Type") == "application/json" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Failed to read response body:", err)
			return "unknown error"
		}

		jsonObj := make(map[string]interface{})
		if err := json.Unmarshal(body, &jsonObj); err != nil {
			log.Println("Failed to parse JSON in response body:", err)
			return "unknown error"
		}

		msg := jsonObj["msg"]
		if msg == nil {
			log.Println("Missing error message in non-2xx response")
			return "unknown error"
		}

		msgStr, ok := msg.(string)
		if !ok {
			log.Println("Non-string msg field in JSON body:", msg)
			return fmt.Sprint(msg)
		}

		return msgStr
	} else {
		return "HTTP: " + resp.Status
	}
}
