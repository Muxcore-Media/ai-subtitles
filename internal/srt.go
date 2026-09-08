package internal

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Muxcore-Media/contracts-ai/infer"
)

var (
	srtIndex = regexp.MustCompile(`^\d+$`)
	srtArrow = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2}[,.]\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}[,.]\d{3})`)
)

func parseSRT(raw string) ([]infer.TranscriptCue, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	blocks := strings.Split(raw, "\n\n")
	var cues []infer.TranscriptCue
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 2 {
			continue
		}
		i := 0
		if srtIndex.MatchString(lines[0]) {
			i = 1
		}
		if i >= len(lines) {
			continue
		}
		m := srtArrow.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		start, err := parseSRTTime(m[1])
		if err != nil {
			return nil, err
		}
		end, err := parseSRTTime(m[2])
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
		cues = append(cues, infer.TranscriptCue{StartMs: start, EndMs: end, Text: text})
	}
	if len(cues) == 0 {
		return nil, fmt.Errorf("no subtitle cues")
	}
	return cues, nil
}

func formatSRT(cues []infer.TranscriptCue) string {
	var b strings.Builder
	for i, c := range cues {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n", i+1, formatSRTTime(c.StartMs), formatSRTTime(c.EndMs), c.Text)
	}
	return b.String()
}

func parseSRTTime(s string) (int64, error) {
	s = strings.ReplaceAll(s, ",", ".")
	t, err := time.Parse("15:04:05.000", s)
	if err != nil {
		return 0, err
	}
	dur := time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second +
		time.Duration(t.Nanosecond())
	return dur.Milliseconds(), nil
}

func formatSRTTime(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	h := ms / 3_600_000
	ms %= 3_600_000
	m := ms / 60_000
	ms %= 60_000
	s := ms / 1000
	frac := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, frac)
}

func shiftCues(cues []infer.TranscriptCue, offsetMs int64) []infer.TranscriptCue {
	out := make([]infer.TranscriptCue, len(cues))
	for i, c := range cues {
		out[i] = infer.TranscriptCue{StartMs: c.StartMs + offsetMs, EndMs: c.EndMs + offsetMs, Text: c.Text}
	}
	return out
}

func scaleCues(cues []infer.TranscriptCue, scale float64) []infer.TranscriptCue {
	if scale <= 0 {
		return cues
	}
	out := make([]infer.TranscriptCue, len(cues))
	for i, c := range cues {
		out[i] = infer.TranscriptCue{
			StartMs: int64(float64(c.StartMs) * scale),
			EndMs:   int64(float64(c.EndMs) * scale),
			Text:    c.Text,
		}
	}
	return out
}

func lastCueEnd(cues []infer.TranscriptCue) int64 {
	var max int64
	for _, c := range cues {
		if c.EndMs > max {
			max = c.EndMs
		}
	}
	return max
}
