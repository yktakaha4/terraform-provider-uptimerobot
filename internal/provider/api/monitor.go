package uptimerobotapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var monitorType = map[string]int{
	"http":    1,
	"keyword": 2,
	"ping":    3,
	"port":    4,
}
var MonitorType = mapKeys(monitorType)

var monitorSubType = map[string]int{
	"http":   1,
	"https":  2,
	"ftp":    3,
	"smtp":   4,
	"pop3":   5,
	"imap":   6,
	"custom": 99,
}
var MonitorSubType = mapKeys(monitorSubType)

var monitorStatus = map[string]int{
	"paused":          0,
	"not checked yet": 1,
	"up":              2,
	"seems down":      8,
	"down":            9,
}

var monitorKeywordType = map[string]int{
	"exists":     1,
	"not exists": 2,
}
var MonitorKeywordType = mapKeys(monitorKeywordType)

var monitorHTTPAuthType = map[string]int{
	"basic":  1,
	"digest": 2,
}
var MonitorHTTPAuthType = mapKeys(monitorHTTPAuthType)

var monitorHTTPMethod = map[string]int{
	"HEAD":    1,
	"GET":     2,
	"POST":    3,
	"PUT":     4,
	"PATCH":   5,
	"DELETE":  6,
	"OPTIONS": 7,
}
var MonitorHTTPMethod = mapKeys(monitorHTTPMethod)

type MonitorAlertContact struct {
	ID         string `json:"id"`
	Recurrence int    `json:"recurrence"`
	Threshold  int    `json:"threshold"`
}

type MonitorCustomHTTPStatuses struct {
	Up   []int
	Down []int
}

type Monitor struct {
	ID           int    `json:"id"`
	FriendlyName string `json:"friendly_name"`
	URL          string `json:"url"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	Interval     int    `json:"interval"`

	SubType string `json:"sub_type"`
	Port    int    `json:"port"`

	KeywordType  string `json:"keyword_type"`
	KeywordValue string `json:"keyword_value"`

	HTTPMethod   string `json:"http_method"`
	HTTPUsername string `json:"http_username"`
	HTTPPassword string `json:"http_password"`
	HTTPAuthType string `json:"http_auth_type"`

	IgnoreSSLErrors bool `json:"ignore_ssl_errors"`

	CustomHTTPHeaders  map[string]string
	CustomHTTPStatuses MonitorCustomHTTPStatuses

	AlertContacts []MonitorAlertContact
}

func (client UptimeRobotApiClient) GetMonitor(id int) (m Monitor, err error) {
	data := url.Values{}
	data.Add("monitors", fmt.Sprintf("%d", id))
	data.Add("ssl", fmt.Sprintf("%d", 1))
	data.Add("custom_http_headers", fmt.Sprintf("%d", 1))
	data.Add("custom_http_statuses", fmt.Sprintf("%d", 1))
	data.Add("alert_contacts", fmt.Sprintf("%d", 1))

	body, err := client.MakeCall(
		"getMonitors",
		data.Encode(),
	)
	if err != nil {
		return
	}

	monitors, ok := body["monitors"].([]interface{})
	if !ok {
		j, _ := json.Marshal(body)
		err = errors.New("Unknown response from the server: " + string(j))
		return
	}

	if len(monitors) < 1 {
		err = fmt.Errorf("Monitor not found: %d", id)
		return
	}

	monitor := monitors[0].(map[string]interface{})

	m.ID = id

	m.FriendlyName = monitor["friendly_name"].(string)
	m.URL = monitor["url"].(string)
	m.Type = intToString(monitorType, int(monitor["type"].(float64)))
	m.Status = intToString(monitorStatus, int(monitor["status"].(float64)))
	m.Interval = int(monitor["interval"].(float64))

	switch m.Type {
	case "port":
		m.SubType = intToString(monitorSubType, int(monitor["sub_type"].(float64)))
		if m.SubType != "custom" {
			m.Port = 0
		} else {
			m.Port = int(monitor["port"].(float64))
		}
		break
	case "keyword":
		m.KeywordType = intToString(monitorKeywordType, int(monitor["keyword_type"].(float64)))
		m.KeywordValue = monitor["keyword_value"].(string)

		if val := monitor["http_auth_type"]; val != nil {
			// PS: There seems to be a bug in the UR api as it never returns this value
			m.HTTPAuthType = intToString(monitorHTTPAuthType, int(val.(float64)))
		}
		// TODO(mnaser): The UptimeRobot API does not return `http_method` when
		//               the monitor type is `http`. I've sent them an email, we
		//               should comment this out once that's fixed.
		//
		// m.HTTPMethod = intToString(monitorHTTPMethod, int(monitor["http_method"].(float64)))
		m.HTTPUsername = monitor["http_username"].(string)
		m.HTTPPassword = monitor["http_password"].(string)
		break
	case "http":
		if val := monitor["http_auth_type"]; val != nil {
			// PS: There seems to be a bug in the UR api as it never returns this value
			m.HTTPAuthType = intToString(monitorHTTPAuthType, int(val.(float64)))
		}
		// TODO(mnaser): The UptimeRobot API does not return `http_method` when
		//               the monitor type is `http`. I've sent them an email, we
		//               should comment this out once that's fixed.
		//
		// m.HTTPMethod = intToString(monitorHTTPMethod, int(monitor["http_method"].(float64)))
		m.HTTPUsername = monitor["http_username"].(string)
		m.HTTPPassword = monitor["http_password"].(string)
		break
	}

	ignoreSSLErrors := int(monitor["ssl"].(map[string]interface{})["ignore_errors"].(float64))
	if ignoreSSLErrors == 1 {
		m.IgnoreSSLErrors = true
	} else {
		m.IgnoreSSLErrors = false
	}

	customHTTPHeaders := make(map[string]string)
	for k, v := range monitor["custom_http_headers"].(map[string]interface{}) {
		customHTTPHeaders[k] = v.(string)
	}
	m.CustomHTTPHeaders = customHTTPHeaders

	if monitor["custom_http_statuses"] != nil {
		statuses := monitor["custom_http_statuses"].(map[string]interface{})
		m.CustomHTTPStatuses = MonitorCustomHTTPStatuses{
			Up:   statuses["up"].([]int),
			Down: statuses["down"].([]int),
		}
	}

	if contacts := monitor["alert_contacts"].([]interface{}); contacts != nil {
		m.AlertContacts = make([]MonitorAlertContact, len(contacts))
		for k, v := range contacts {
			contact := v.(map[string]interface{})
			var ac MonitorAlertContact
			ac.ID = contact["id"].(string)
			ac.Recurrence = int(contact["recurrence"].(float64))
			ac.Threshold = int(contact["threshold"].(float64))
			m.AlertContacts[k] = ac
		}
	}

	return
}

type MonitorRequestAlertContact struct {
	ID         string
	Threshold  int
	Recurrence int
}
type MonitorCreateRequest struct {
	FriendlyName string
	URL          string
	Type         string
	Interval     int

	SubType string
	Port    int

	KeywordType  string
	KeywordValue string

	HTTPMethod   string
	HTTPUsername string
	HTTPPassword string
	HTTPAuthType string

	IgnoreSSLErrors bool

	AlertContacts []MonitorRequestAlertContact

	CustomHTTPHeaders  map[string]string
	CustomHTTPStatuses MonitorCustomHTTPStatuses
}

func (statuses MonitorCustomHTTPStatuses) toRequestData() string {
	// must be sent as 404:0_200:1 to accept 404 as down and 200 as up

	upStatuses := make([]string, len(statuses.Up))
	for i, v := range statuses.Up {
		upStatuses[i] = fmt.Sprintf("%03d:1", v)
	}

	downStatuses := make([]string, len(statuses.Down))
	for i, v := range statuses.Down {
		downStatuses[i] = fmt.Sprintf("%03d:0", v)
	}

	allStatuses := append(upStatuses, downStatuses...)

	return strings.Join(allStatuses, "_")
}

func (client UptimeRobotApiClient) CreateMonitor(req MonitorCreateRequest) (m Monitor, err error) {
	data := url.Values{}
	data.Add("friendly_name", req.FriendlyName)
	data.Add("url", req.URL)
	data.Add("type", fmt.Sprintf("%d", monitorType[req.Type]))
	data.Add("interval", fmt.Sprintf("%d", req.Interval))
	switch req.Type {
	case "port":
		data.Add("sub_type", fmt.Sprintf("%d", monitorSubType[req.SubType]))
		data.Add("port", fmt.Sprintf("%d", req.Port))
		break
	case "keyword":
		data.Add("keyword_type", fmt.Sprintf("%d", monitorKeywordType[req.KeywordType]))
		data.Add("keyword_value", req.KeywordValue)

		data.Add("http_method", fmt.Sprintf("%d", monitorHTTPMethod[req.HTTPMethod]))
		data.Add("http_auth_type", fmt.Sprintf("%d", monitorHTTPAuthType[req.HTTPAuthType]))
		data.Add("http_username", req.HTTPUsername)
		data.Add("http_password", req.HTTPPassword)
		break
	case "http":
		data.Add("http_method", fmt.Sprintf("%d", monitorHTTPMethod[req.HTTPMethod]))
		data.Add("http_auth_type", fmt.Sprintf("%d", monitorHTTPAuthType[req.HTTPAuthType]))
		data.Add("http_username", req.HTTPUsername)
		data.Add("http_password", req.HTTPPassword)
		if req.HTTPMethod == "POST" {
			data.Add("post_type", "1")
			data.Add("post_content_type", "0")
			data.Add("post_value", "{}")
		}
		break
	}

	if req.IgnoreSSLErrors {
		data.Add("ignore_ssl_errors", "1")
	} else {
		data.Add("ignore_ssl_errors", "0")
	}

	acStrings := make([]string, len(req.AlertContacts))
	for k, v := range req.AlertContacts {
		acStrings[k] = fmt.Sprintf("%s_%d_%d", v.ID, v.Threshold, v.Recurrence)
	}
	data.Add("alert_contacts", strings.Join(acStrings, "-"))

	// custom http headers
	if len(req.CustomHTTPHeaders) > 0 {
		jsonData, err := json.Marshal(req.CustomHTTPHeaders)
		if err == nil {
			data.Add("custom_http_headers", string(jsonData))
		}
	}

	// custom http statuses
	if len(req.CustomHTTPStatuses.Up) > 0 || len(req.CustomHTTPStatuses.Down) > 0 {
		data.Add("custom_http_statuses", req.CustomHTTPStatuses.toRequestData())
	}

	body, err := client.MakeCall(
		"newMonitor",
		data.Encode(),
	)
	if err != nil {
		return
	}

	monitor := body["monitor"].(map[string]interface{})
	id := int(monitor["id"].(float64))

	return client.GetMonitor(id)
}

type MonitorUpdateRequest struct {
	ID           int
	FriendlyName string
	URL          string
	Type         string
	Interval     int

	SubType string
	Port    int

	KeywordType  string
	KeywordValue string

	HTTPMethod   string
	HTTPUsername string
	HTTPPassword string
	HTTPAuthType string

	IgnoreSSLErrors bool

	AlertContacts []MonitorRequestAlertContact

	CustomHTTPHeaders  map[string]string
	CustomHTTPStatuses MonitorCustomHTTPStatuses
}

func (client UptimeRobotApiClient) UpdateMonitor(req MonitorUpdateRequest) (m Monitor, err error) {
	data := url.Values{}
	data.Add("id", fmt.Sprintf("%d", req.ID))
	data.Add("friendly_name", req.FriendlyName)
	data.Add("url", req.URL)
	data.Add("type", fmt.Sprintf("%d", monitorType[req.Type]))
	data.Add("interval", fmt.Sprintf("%d", req.Interval))
	switch req.Type {
	case "port":
		data.Add("sub_type", fmt.Sprintf("%d", monitorSubType[req.SubType]))
		data.Add("port", fmt.Sprintf("%d", req.Port))
		break
	case "keyword":
		data.Add("keyword_type", fmt.Sprintf("%d", monitorKeywordType[req.KeywordType]))
		data.Add("keyword_value", req.KeywordValue)

		data.Add("http_method", fmt.Sprintf("%d", monitorHTTPMethod[req.HTTPMethod]))
		data.Add("http_auth_type", fmt.Sprintf("%d", monitorHTTPAuthType[req.HTTPAuthType]))
		data.Add("http_username", req.HTTPUsername)
		data.Add("http_password", req.HTTPPassword)
		break
	case "http":
		data.Add("http_method", fmt.Sprintf("%d", monitorHTTPMethod[req.HTTPMethod]))
		data.Add("http_auth_type", fmt.Sprintf("%d", monitorHTTPAuthType[req.HTTPAuthType]))
		data.Add("http_username", req.HTTPUsername)
		data.Add("http_password", req.HTTPPassword)
		break
	}

	if req.IgnoreSSLErrors {
		data.Add("ignore_ssl_errors", "1")
	} else {
		data.Add("ignore_ssl_errors", "0")
	}

	acStrings := make([]string, len(req.AlertContacts))
	for k, v := range req.AlertContacts {
		acStrings[k] = fmt.Sprintf("%s_%d_%d", v.ID, v.Threshold, v.Recurrence)
	}
	data.Add("alert_contacts", strings.Join(acStrings, "-"))

	// custom http headers
	if len(req.CustomHTTPHeaders) > 0 {
		jsonData, err := json.Marshal(req.CustomHTTPHeaders)
		if err == nil {
			data.Add("custom_http_headers", string(jsonData))
		}
	} else {
		//delete custom http headers when it is empty
		data.Add("custom_http_headers", "{}")
	}

	// custom http statuses
	if len(req.CustomHTTPStatuses.Up) > 0 || len(req.CustomHTTPStatuses.Down) > 0 {
		data.Add("custom_http_statuses", req.CustomHTTPStatuses.toRequestData())
	}

	_, err = client.MakeCall(
		"editMonitor",
		data.Encode(),
	)
	if err != nil {
		return
	}

	return client.GetMonitor(req.ID)
}

func (client UptimeRobotApiClient) DeleteMonitor(id int) (err error) {
	data := url.Values{}
	data.Add("id", fmt.Sprintf("%d", id))

	_, err = client.MakeCall(
		"deleteMonitor",
		data.Encode(),
	)
	if err != nil {
		return
	}
	return
}
