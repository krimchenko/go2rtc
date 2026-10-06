package cronjobs

import (
	"fmt"

	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/AlexxIT/go2rtc/internal/record"
	"github.com/robfig/cron"
)

func Init() {
	var cfg struct {
		Record map[string]any `yaml:"record"`
	}
	app.LoadConfig(&cfg)

	segmentDuration, ok := cfg.Record["segmentDuration"].(string)
	if !ok || segmentDuration == "" {
		return
	}

	c := cron.New()
	if finalizeScriptPath, ok := cfg.Record["finalizeScriptPath"].(string); ok && finalizeScriptPath != "" {
		c.AddFunc(fmt.Sprintf("@every %s", segmentDuration), record.FFmpegFinalizeRecordings)
	}
	c.AddFunc("* */5 * * * *", record.RemoveDanglingRecodings)
	c.Start()
}
