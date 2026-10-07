package app

import (
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger
var eventURLTemplate *template.Template
var eventBodyTemplate *template.Template
var method string
var BearerToken string

type eventData struct {
	IP, EVENT, TIME, NAME string
}

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

	log = GetLogger("event")
	var err error
	eventURLTemplate, err = template.New("event.url").Parse(cfg.Mod.Url)
	if err != nil {
		log.Error().Err(err).Msg("invalid event.url template")
		return
	}
	if cfg.Mod.Method == "POST" {
		eventBodyTemplate, err = template.New("event.body").Parse(cfg.Mod.Body)
		if err != nil {
			log.Error().Err(err).Msg("invalid event.body template")
			eventURLTemplate = nil
			return
		}
	}
	method = cfg.Mod.Method
	BearerToken = cfg.Mod.BearerToken
}

func RecordEvent(ts time.Time, event, name, ip, token string) {
	if eventURLTemplate != nil {

		log.Debug().Msgf("[event] %s %s %s", event, name, ip)

		data := eventData{ip, event, ts.Format(time.DateTime), name}
		urlData := eventData{
			url.QueryEscape(data.IP),
			url.QueryEscape(data.EVENT),
			url.QueryEscape(data.TIME),
			url.QueryEscape(data.NAME),
		}
		var reqURL strings.Builder
		if err := eventURLTemplate.Execute(&reqURL, urlData); err != nil {
			log.Error().Err(err).Msg("failed to render event.url template")
			return
		}

		var req *http.Request
		var err error

		if method == "POST" {
			var reqBody strings.Builder
			if err := eventBodyTemplate.Execute(&reqBody, data); err != nil {
				log.Error().Err(err).Msg("failed to render event.body template")
				return
			}

			req, err = http.NewRequest(method, reqURL.String(), strings.NewReader(reqBody.String()))
		} else {
			req, err = http.NewRequest(method, reqURL.String(), nil)
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
