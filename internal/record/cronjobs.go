package record

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"sync"

	"github.com/AlexxIT/go2rtc/internal/app"
)

var finalizeMu sync.Mutex

// FFmpegFinalizeRecordings runs the configured finalizer executable.
func FFmpegFinalizeRecordings() {
	if !finalizeMu.TryLock() {
		return
	}
	defer finalizeMu.Unlock()

	var cfg struct {
		Record map[string]any `yaml:"record"`
	}
	app.LoadConfig(&cfg)

	recordingsBasePath, err := recordingRoot(cfg.Record["basePath"].(string))
	if err != nil {
		log.Error().Err(err).Msg("failed to find recordings root")
		return
	}
	segmentDuration, _, err := parseSegmentDuration(cfg.Record["segmentDuration"])
	if err != nil {
		log.Error().Err(err).Msg("invalid segment duration")
		return
	}
	args := []string{recordingsBasePath, strconv.FormatFloat(segmentDuration.Seconds(), 'f', -1, 64)}
	for _, recording := range recordings {
		args = append(args, recording.activeFiles()...)
	}

	cmd := exec.Command(cfg.Record["finalizeScriptPath"].(string), args...)
	log.Debug().Msgf("cron job 'FFmpegFinalizeRecordings' will execute: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Msgf("cron job 'FFmpegFinalizeRecordings' has failed: %s", output)
		return
	}
}

// RemoveDanglingRecodings is a cron job that removes dangling recordings
// older than the configured window, e.g. caused by a crash.
func RemoveDanglingRecodings() {
	var cfg struct {
		Record map[string]any `yaml:"record"`
	}
	app.LoadConfig(&cfg)
	numSegments, ok := cfg.Record["numSegments"].(int)
	if !ok || numSegments < 1 {
		return
	}

	recordingsBasePath, err := recordingRoot(cfg.Record["basePath"].(string))
	if err != nil {
		log.Error().Err(err).Msg("failed to find recordings root")
		return
	}
	segmentDuration, _, err := parseSegmentDuration(cfg.Record["segmentDuration"])
	if err != nil {
		log.Error().Err(err).Msg("invalid segment duration")
		return
	}
	lifespanMins := math.Ceil(segmentDuration.Minutes()*float64(numSegments)) + 5

	cmd := exec.Command(
		"find",
		recordingsBasePath,
		"-type",
		"f",
		"-mmin",
		fmt.Sprintf("+%.0f", lifespanMins),
		"-delete",
	)
	log.Debug().Msgf("cron job 'RemoveDanglingRecodings' will execute: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Msgf("cron job 'RemoveDanglingRecodings' has failed: %s", output)
		return
	}
}
