package record

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type pathData struct {
	URL, DEVICE_NAME, ID string
	DATETIME, DATE, TIME string
	YEAR, MONTH, DAY     string
	HOUR, MINUTE, SECOND string
	NOW                  time.Time
	DURATION, NUMBER     string
	TIMESTAMP            int64
}

func newPathTemplate(basePath, streamID string) (*template.Template, error) {
	if strings.TrimSpace(basePath) == "" {
		return nil, fmt.Errorf("record.basePath is empty")
	}
	if !strings.Contains(basePath, "{{") {
		basePath = filepath.Join(basePath, strings.ReplaceAll(streamID, "/", "-"))
	}
	return template.New("basePath").Parse(basePath)
}

func recordingRoot(basePath string) (string, error) {
	if strings.TrimSpace(basePath) == "" {
		return "", fmt.Errorf("record.basePath is empty")
	}
	prefix, _, hasTemplate := strings.Cut(basePath, "{{")
	if !hasTemplate {
		return filepath.Clean(basePath), nil
	}
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix = filepath.Dir(prefix)
	}
	root := filepath.Clean(prefix)
	if root == "." || root == string(filepath.Separator) {
		return "", fmt.Errorf("record.basePath needs a fixed directory prefix for background jobs")
	}
	return root, nil
}

func (s *Segments) pathFor(now time.Time) (string, error) {
	var path strings.Builder
	if err := s.pathTemplate.Execute(&path, s.templateDataFor(now)); err != nil {
		return "", err
	}
	return filepath.Clean(path.String()), nil
}

func (s *Segments) templateDataFor(now time.Time) pathData {
	data := s.pathData
	data.NOW = now
	if data.DURATION == "" {
		data.DURATION = s.segmentDuration.String()
	}
	data.NUMBER = fmt.Sprintf("%08d", s.nextNumber)
	data.TIMESTAMP = now.Unix()
	data.DATETIME = now.Format("2006-01-02_15-04-05")
	data.DATE = now.Format("2006-01-02")
	data.TIME = now.Format("15-04-05")
	data.YEAR = now.Format("2006")
	data.MONTH = now.Format("01")
	data.DAY = now.Format("02")
	data.HOUR = now.Format("15")
	data.MINUTE = now.Format("04")
	data.SECOND = now.Format("05")

	return data
}
