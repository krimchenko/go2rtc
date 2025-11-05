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
    
    fmt.Println("[test] cronjob init")

    if cfg.Record == nil {
        fmt.Println("[test] cronjob return")
        return
    }
    
    fmt.Println("[test] cronjob if")
    
    if cfg.Record["finalizeScriptPath"] != "" && cfg.Record["segmentDuration"] != "" {
        fmt.Println("[test] cronjob cron new")
        c := cron.New()
        c.Start()
        fmt.Println("[test] cronjob cron if finalizeScriptPath")
        if cfg.Record["finalizeScriptPath"] != "" {
            fmt.Println("[test] cronjob cron add finalizeScriptPath")
            c.AddFunc(
                fmt.Sprintf("@every %s", cfg.Record["segmentDuration"].(string)),
                record.FFmpegFinalizeRecordings,
            )
        }
        
        fmt.Println("[test] cronjob cron if segmentDuration")
        if cfg.Record["segmentDuration"] != "" {
            fmt.Println("[test] cronjob cron add segmentDuration")
            c.AddFunc("* */5 * * * *", record.RemoveDanglingRecodings)
        }
    }
}
