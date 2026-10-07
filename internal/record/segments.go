package record

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/AlexxIT/go2rtc/internal/streams"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/mp4"
	"github.com/AlexxIT/go2rtc/pkg/mpegts"
)

const dateFormat = "2006-01-02_15_04_05"

var mp4MagicNumber = []byte{0, 0, 0, 28, 102, 116, 121, 112}
var tsHeader = []byte{0x47, 0x40, 0x00, 0x10}

type Segments struct {
	segmentDuration time.Duration
	numSegments     int
	pathTemplate    *template.Template
	nameTemplate    *template.Template
	pathData        pathData
	filenameTZ      *time.Location

	files        []*os.File
	current      int
	nextNumber   uint64
	nameCounters map[string]int
	mu           sync.Mutex

	streamName string
	stream     *streams.Stream
	medias     []*core.Media
	cons       *mp4.Consumer
	tsCons     *mpegts.Consumer
	ts         bool
}

func NewSegments(
	segmentDuration time.Duration, durationText string, numSegments int,
	basePath, filename string, filenameTZ *time.Location, streamName, streamURL, deviceName string,
) (segments *Segments, err error) {
	pathTemplate, err := newPathTemplate(basePath, streamName)
	if err != nil {
		return nil, err
	}
	var nameTemplate *template.Template
	if filename != "" {
		nameTemplate, err = template.New("filename").Parse(filename)
		if err != nil {
			return nil, err
		}
	}
	fileSlots := numSegments
	if fileSlots < 2 {
		fileSlots = 2
	}
	segments = &Segments{
		segmentDuration: segmentDuration,
		numSegments:     numSegments,
		pathTemplate:    pathTemplate,
		nameTemplate:    nameTemplate,
		pathData:        pathData{ID: streamName, URL: streamURL, DEVICE_NAME: deviceName, DURATION: durationText},
		filenameTZ:      filenameTZ,
		files:           make([]*os.File, fileSlots),
		nextNumber:      1,
		nameCounters:    make(map[string]int),
		streamName:      streamName,
		stream:          streams.Get(streamName),
		medias:          mp4.ParseQuery(map[string][]string{"src": {streamName}, "mp4": {"all"}}),
	}
	path, err := segments.pathFor(time.Now().In(filenameTZ))
	if err != nil {
		return nil, err
	}
	segmentFilename, err := segments.filenameFor(time.Now().In(filenameTZ))
	if err != nil {
		return nil, err
	}
	segments.ts = strings.HasSuffix(segmentFilename, ".ts")
	err = os.MkdirAll(path, 0750)
	if err != nil {
		return nil, err
	}

	return
}

func (s *Segments) Write(b []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ts && bytes.HasPrefix(b, tsHeader) || !s.ts && bytes.HasPrefix(b, mp4MagicNumber) {
		s.switchFile()
	}
	return s.files[s.current].Write(b)
}

func (s *Segments) Record() {
	for {
		if err := s.prepareNextFile(); err == nil {
			break
		} else {
			log.Error().Err(err).Msg("failed to open new segment file, retrying...")
		}
		time.Sleep(30 * time.Second)
	}

	var cons core.Consumer
	if s.ts {
		s.tsCons = mpegts.NewConsumer()
		cons = s.tsCons
	} else {
		s.cons = mp4.NewConsumer(s.medias)
		cons = s.cons
	}

	for {
		err := s.stream.AddConsumer(cons)
		if err == nil {
			break
		}
		log.Error().Err(err).Msgf("failed to add a recording consumer (%s), retrying...", s.streamName)
		time.Sleep(30 * time.Second)
	}
	go func() {
		if s.ts {
			_, _ = s.tsCons.WriteTo(s) // blocks
		} else {
			_, _ = s.cons.WriteTo(s) // blocks
		}
	}()

	s.scheduleSwitch()
}

func (s *Segments) switchFile() {
	prev := s.current
	prevFile := s.files[prev]
	s.current++
	if s.current == len(s.files) {
		s.current = 0
	}
	go func() {
		if prevFile != nil {
			_ = prevFile.Close()
			if s.numSegments == 1 {
				removeSegment(prevFile.Name())
			}
		}
	}()
}

func (s *Segments) prepareNextFile() error {
	now := time.Now().In(s.filenameTZ)
	path, err := s.pathFor(now)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0750); err != nil {
		return err
	}
	finalName, err := s.filenameFor(now)
	if err != nil {
		return err
	}
	if strings.HasSuffix(finalName, ".ts") != s.ts {
		return fmt.Errorf("record.filename extension changed: %q", finalName)
	}
	newFile, err := s.openNextFile(path, finalName)
	if err != nil {
		return err
	}

	s.mu.Lock()
	next := s.current + 1
	if next == len(s.files) {
		next = 0
	}
	oldFile := s.files[next]
	s.files[next] = newFile
	s.nextNumber++
	s.mu.Unlock()
	if oldFile != nil {
		go func() {
			info, err := oldFile.Stat()
			_ = oldFile.Close()
			if err == nil && info.Size() == 0 {
				_ = os.Remove(oldFile.Name())
			} else if s.numSegments > 1 {
				removeSegment(oldFile.Name())
			}
		}()
	}
	return nil
}

func (s *Segments) filenameFor(now time.Time) (string, error) {
	if s.nameTemplate == nil {
		return fmt.Sprintf("%s_%s.mp4", now.Format(dateFormat), now.Add(s.segmentDuration).Format(dateFormat)), nil
	}
	var name strings.Builder
	if err := s.nameTemplate.Execute(&name, s.templateDataFor(now)); err != nil {
		return "", err
	}
	filename := name.String()
	if filename == "" || strings.ContainsAny(filename, `/\`) {
		return "", fmt.Errorf("record.filename must be an mp4 or ts file name: %q", filename)
	}
	return filename, nil
}

func (s *Segments) openNextFile(path, filename string) (*os.File, error) {
	finalizeMu.Lock()
	defer finalizeMu.Unlock()

	ext := ".mp4"
	if s.ts {
		ext = ".ts"
	}
	stem := strings.TrimSuffix(filename, ext)
	key := filepath.Join(path, stem)
	for number := s.nameCounters[key]; ; number++ {
		candidate := stem
		if number > 0 {
			candidate = fmt.Sprintf("%s_%d", stem, number)
		}
		finalPath := filepath.Join(path, candidate+ext)
		if _, err := os.Stat(finalPath); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		rawPath := filepath.Join(path, candidate)
		if s.ts {
			rawPath = finalPath
		}
		file, err := os.OpenFile(rawPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		s.nameCounters[key] = number + 1
		return file, nil
	}
}

func removeSegment(rawPath string) {
	finalizeMu.Lock()
	defer finalizeMu.Unlock()
	for _, path := range []string{rawPath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Error().Err(err).Msg("failed to remove old segment file")
		}
	}
}

func (s *Segments) activeFiles() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var paths []string
	for _, file := range s.files {
		if file != nil {
			paths = append(paths, file.Name())
		}
	}
	return paths
}

func (s *Segments) scheduleSwitch() {
	for range time.NewTicker(s.segmentDuration).C {
		if err := s.prepareNextFile(); err != nil {
			log.Error().Err(err).Msg("failed to open new segment file")
			continue
		}
		if s.ts {
			if err := s.tsCons.WriteHeader(); err != nil {
				log.Error().Err(err).Msg("failed to write segment header")
			}
		} else {
			s.cons.ResetMuxer() // trigger the muxer to send mp4 magic number
		}
	}
}
