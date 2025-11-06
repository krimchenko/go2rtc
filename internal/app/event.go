package app

import (
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var log zerolog.Logger
var Url string
var Method string
var BearerToken string
var Body string

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

	Url = cfg.Mod.Url
	Method = cfg.Mod.Method
	Body = cfg.Mod.Body
	BearerToken = cfg.Mod.BearerToken
	log = GetLogger("event")
}

func RecordEvent(ts time.Time, event, name, ip, token string) {
	if Url != "" {

		log.Debug().Msgf("[event] %s %s %s", event, name, ip)

		reqUrl := strings.ReplaceAll(Url, "%ip", url.QueryEscape(ip))
		reqUrl = strings.ReplaceAll(reqUrl, "%event", url.QueryEscape(event))
		reqUrl = strings.ReplaceAll(reqUrl, "%time", url.QueryEscape(ts.Format(time.DateTime)))
		reqUrl = strings.ReplaceAll(reqUrl, "%name", url.QueryEscape(name))

		if Method == "POST" {
			//request := struct {
			//	Names string `json:"names"`
			//}{
			//	Names: name,
			//}
			//data, err := json.Marshal(request)
			//if err != nil {
			//	return
			//}

			reqBody := strings.ReplaceAll(Body, "%ip", ip)
			reqBody = strings.ReplaceAll(reqBody, "%event", event)
			reqBody = strings.ReplaceAll(reqBody, "%time", ts.Format(time.DateTime))
			reqBody = strings.ReplaceAll(reqBody, "%name", name)

			req, err := http.NewRequest(Method, reqUrl, strings.NewReader(reqBody))
			if err != nil {
				return
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			} else if BearerToken != "" {
				req.Header.Set("Authorization", "Bearer "+BearerToken)
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: time.Second * 3}
			res, err := client.Do(req)
			if err != nil {
				return
			}
			defer res.Body.Close()
		} else {
			req, err := http.NewRequest(Method, reqUrl, nil)
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
}
