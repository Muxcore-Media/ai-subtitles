package internal

import (
	"fmt"
	"math"
	"strings"

	"github.com/Muxcore-Media/contracts-ai/infer"
	mediaevents "github.com/Muxcore-Media/contracts-media/events"
)

// GenerateInput is enough to produce a sidecar without a cloud ASR key.
type GenerateInput struct {
	MediaID        string  `json:"media_id"`
	MediaFileID    string  `json:"media_file_id"`
	Language       string  `json:"language"`
	DurationSec    float64 `json:"duration_sec"`
	TranscriptHint string  `json:"transcript_hint"`
	FilePath       string  `json:"file_path"`
}

// GenerateResult is a generated SRT job.
type GenerateResult struct {
	JobID    string                `json:"job_id"`
	MediaID  string                `json:"media_id"`
	Language string                `json:"language"`
	SRT      string                `json:"srt"`
	Cues     []infer.TranscriptCue `json:"cues"`
}

func generateSubtitles(in GenerateInput) (GenerateResult, error) {
	hint := strings.TrimSpace(in.TranscriptHint)
	if hint == "" && in.FilePath != "" {
		hint = "Spoken dialogue for " + in.FilePath
	}
	if hint == "" {
		hint = "Generated dialogue for media " + firstNonEmpty(in.MediaID, in.MediaFileID, "unknown")
	}
	lang := in.Language
	if lang == "" {
		lang = "en"
	}
	cues := heuristicCues(hint, in.DurationSec)
	return GenerateResult{
		MediaID:  in.MediaID,
		Language: lang,
		SRT:      formatSRT(cues),
		Cues:     cues,
	}, nil
}

func heuristicCues(hint string, durationSec float64) []infer.TranscriptCue {
	return splitHint(hint, durationSec)
}

func splitHint(hint string, durationSec float64) []infer.TranscriptCue {
	parts := strings.FieldsFunc(hint, func(r rune) bool { return r == '.' || r == '!' || r == '?' || r == '\n' })
	var lines []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			lines = append(lines, p)
		}
	}
	if len(lines) == 0 {
		lines = []string{strings.TrimSpace(hint)}
	}
	if durationSec <= 0 {
		durationSec = float64(len(lines)) * 2.5
	}
	slot := durationSec * 1000 / float64(len(lines))
	cues := make([]infer.TranscriptCue, 0, len(lines))
	for i, line := range lines {
		start := int64(float64(i) * slot)
		end := int64(float64(i+1) * slot)
		cues = append(cues, infer.TranscriptCue{StartMs: start, EndMs: end, Text: line})
	}
	return cues
}

// SyncInput aligns a foreign subtitle file to media duration / speech start.
type SyncInput struct {
	MediaID         string `json:"media_id"`
	SubtitleSRT     string `json:"subtitle_srt"`
	MediaDurationMs int64  `json:"media_duration_ms"`
	SpeechStartMs   int64  `json:"speech_start_ms"`
}

// SyncResult is the rewritten SRT plus how it was aligned.
type SyncResult struct {
	JobID    string                `json:"job_id"`
	MediaID  string                `json:"media_id"`
	OffsetMs int64                 `json:"offset_ms"`
	Scale    float64               `json:"scale"`
	Mode     string                `json:"mode"`
	SRT      string                `json:"srt"`
	Cues     []infer.TranscriptCue `json:"cues"`
}

func syncSubtitles(in SyncInput) (SyncResult, error) {
	cues, err := parseSRT(in.SubtitleSRT)
	if err != nil {
		return SyncResult{}, err
	}
	offset := in.SpeechStartMs - cues[0].StartMs
	shifted := shiftCues(cues, offset)
	scale := 1.0
	mode := "offset"
	if in.MediaDurationMs > 0 {
		last := lastCueEnd(shifted)
		if last > 0 {
			ratio := float64(in.MediaDurationMs) / float64(last)
			if math.Abs(ratio-1) > 0.03 {
				scale = ratio
				shifted = scaleCues(shifted, scale)
				mode = "offset+scale"
			}
		}
	}
	return SyncResult{
		MediaID:  in.MediaID,
		OffsetMs: offset,
		Scale:    scale,
		Mode:     mode,
		SRT:      formatSRT(shifted),
		Cues:     shifted,
	}, nil
}

func generateFromSearchFailed(p mediaevents.SubtitleSearchFailedPayload, durationSec float64, hint string) (GenerateResult, error) {
	if strings.TrimSpace(hint) == "" {
		hint = fmt.Sprintf("Dialogue for %s", firstNonEmpty(p.Query, p.MediaFileID, "untitled"))
	}
	return generateSubtitles(GenerateInput{
		MediaID: p.MediaFileID, MediaFileID: p.MediaFileID,
		Language: p.Language, DurationSec: durationSec,
		TranscriptHint: hint, FilePath: p.FilePath,
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
