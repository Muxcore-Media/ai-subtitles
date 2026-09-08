package internal

import (
	"strings"
	"testing"

	mediaevents "github.com/Muxcore-Media/contracts-media/events"
)

func TestGenerateSubtitlesFromHint(t *testing.T) {
	got, err := generateSubtitles(GenerateInput{
		MediaID: "m1", Language: "en", DurationSec: 6,
		TranscriptHint: "Hello there. Welcome aboard.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.SRT, "Hello there") || !strings.Contains(got.SRT, "Welcome aboard") {
		t.Fatalf("srt = %s", got.SRT)
	}
	if len(got.Cues) != 2 || got.Cues[1].EndMs != 6000 {
		t.Fatalf("cues = %#v", got.Cues)
	}
}

func TestSyncSubtitlesOffsetAndScale(t *testing.T) {
	src := formatSRT(splitHint("One. Two.", 10))
	got, err := syncSubtitles(SyncInput{
		MediaID: "m1", SubtitleSRT: src,
		MediaDurationMs: 20000, SpeechStartMs: 2000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.OffsetMs != 2000 {
		t.Fatalf("offset = %d", got.OffsetMs)
	}
	if got.Scale < 1.6 || got.Scale > 1.7 {
		t.Fatalf("scale = %f mode=%s", got.Scale, got.Mode)
	}
	if got.Cues[0].StartMs < 3000 {
		t.Fatalf("first start = %d (expected scaled after +2s)", got.Cues[0].StartMs)
	}
}

func TestGenerateFromSearchFailed(t *testing.T) {
	got, err := generateFromSearchFailed(mediaevents.SubtitleSearchFailedPayload{
		MediaFileID: "f1", Query: "Arrival", Language: "en", Reason: "empty_results",
	}, 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.SRT, "Arrival") {
		t.Fatalf("srt = %s", got.SRT)
	}
}

func TestParseRoundTripSRT(t *testing.T) {
	raw := "1\n00:00:01,000 --> 00:00:02,500\nHi\n\n2\n00:00:03,000 --> 00:00:04,000\nThere\n"
	cues, err := parseSRT(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0].Text != "Hi" || cues[0].StartMs != 1000 {
		t.Fatalf("cues = %#v", cues)
	}
	out := formatSRT(cues)
	again, err := parseSRT(out)
	if err != nil || len(again) != 2 {
		t.Fatalf("roundtrip %q err=%v", out, err)
	}
}
