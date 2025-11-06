package app

import (
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var log zerolog.Logger
var eventUrl string
var method string
var BearerToken string
var body string

func initEventer() {
	var cfg struct {
		Mod struct {
			Url         string `yaml:"url"`
			Method      string `yaml:"method"`
			Body        string `yaml:"body"`
			BearerToken string `yaml:"bearer_token"`
		} `yaml:"event"`
	}

	cfg.Mod.Method = "GET"

	// load config from YAML
	LoadConfig(&cfg)

	if cfg.Mod.Url == "" {
		return
	}

	eventUrl = cfg.Mod.Url
	method = cfg.Mod.Method
	body = cfg.Mod.Body
	BearerToken = cfg.Mod.BearerToken
	log = GetLogger("event")
}

func RecordEvent(ts time.Time, event, name, ip, token string) {
	if eventUrl != "" {

		log.Debug().Msgf("[event] %s %s %s", event, name, ip)

		reqUrl := strings.ReplaceAll(eventUrl, "%ip", url.QueryEscape(ip))
		reqUrl = strings.ReplaceAll(reqUrl, "%event", url.QueryEscape(event))
		reqUrl = strings.ReplaceAll(reqUrl, "%time", url.QueryEscape(ts.Format(time.DateTime)))
		reqUrl = strings.ReplaceAll(reqUrl, "%name", url.QueryEscape(name))

		var req *http.Request
		var err error

		if method == "POST" {
			reqBody := strings.ReplaceAll(body, "%ip", ip)
			reqBody = strings.ReplaceAll(reqBody, "%event", event)
			reqBody = strings.ReplaceAll(reqBody, "%time", ts.Format(time.DateTime))
			reqBody = strings.ReplaceAll(reqBody, "%name", name)

			req, err = http.NewRequest(method, reqUrl, strings.NewReader(reqBody))
		} else {
			req, err = http.NewRequest(method, reqUrl, nil)
		}

		if err != nil {
			return
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+BearerToken)
		}

		client := &http.Client{Timeout: time.Second * 3}
		res, err := client.Do(req)
		if err != nil {
			return
		}
		defer res.Body.Close()
	}
}
