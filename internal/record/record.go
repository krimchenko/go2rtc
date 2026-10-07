package record

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/rs/zerolog"
)

var log zerolog.Logger
var recordings = map[string]*Segments{}

func Init() {
	log = app.GetLogger("record")

	var cfg struct {
		Record  map[string]any `yaml:"record"`
		Streams map[string]any `yaml:"streams"`
	}

	// todo defaults

	app.LoadConfig(&cfg)

	basePath, ok := cfg.Record["basePath"].(string)
	if !ok {
		return
		//log.Fatal().Msg("record.basePath is invalid")
	}

	segmentDuration, segmentDurationStr, err := parseSegmentDuration(cfg.Record["segmentDuration"])
	if err != nil {
		log.Fatal().Msg("record.segmentDuration is invalid")
	}

	timezone := time.Local
	if timezoneStr, ok := cfg.Record["timezone"].(string); ok && strings.TrimSpace(timezoneStr) != "" {
		if location, err := time.LoadLocation(timezoneStr); err == nil {
			timezone = location
		} else {
			log.Warn().Err(err).Msg("record.timezone is invalid, using system timezone")
		}
	}

	numSegments, ok := cfg.Record["numSegments"].(int)
	if cfg.Record["numSegments"] != nil && !ok {
		log.Fatal().Msg("record.numSegments is invalid")
	}
	filename, ok := cfg.Record["filename"].(string)
	if cfg.Record["filename"] != nil && !ok {
		log.Fatal().Msg("record.filename is invalid")
	}

	for streamName, streamConfig := range cfg.Streams {
		var streamURL, deviceName string
		switch stream := streamConfig.(type) {
		case string:
			streamURL = stream
		case map[string]any:
			streamURL, _ = stream["url"].(string)
			deviceName, _ = stream["device_name"].(string)
		}
		seg, err := NewSegments(
			segmentDuration,
			segmentDurationStr,
			numSegments,
			basePath,
			filename,
			timezone,
			streamName,
			streamURL,
			deviceName,
		)

		if err != nil {
			log.Fatal().Err(err).Msg("failed to create segments")
		}
		recordings[streamName] = seg

		go seg.Record()
		time.Sleep(time.Second * 2) // sleep couple seconds so streams won't switch segments all at the same time
	}
}

func parseSegmentDuration(value any) (time.Duration, string, error) {
	var text string
	switch v := value.(type) {
	case string:
		text = strings.TrimSpace(v)
	case int, int64, float64:
		text = fmt.Sprint(v)
	default:
		return 0, "", fmt.Errorf("invalid segment duration: %v", value)
	}

	parseText := text
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		parseText += "s"
	}
	duration, err := time.ParseDuration(parseText)
	return duration, text, err
}
