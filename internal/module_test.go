package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestModuleInfo(t *testing.T) {
	m := NewModule()
	if m.Info().ID != "ai-subtitles" {
		t.Fatalf("id = %s", m.Info().ID)
	}
}

func TestGenerateHTTP(t *testing.T) {
	m := New(Config{GRPCAddr: "127.0.0.1:0", HTTPAddr: "127.0.0.1:0"})
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Stop(ctx) })
	waitHealth(t, m.HTTPListenAddr())

	resp, err := http.Post("http://"+m.HTTPListenAddr()+"/v1/generate", "application/json",
		strings.NewReader(`{"media_id":"m1","language":"en","duration_sec":4,"transcript_hint":"Hi. Bye."}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var got GenerateResult
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.SRT, "Hi") {
		t.Fatalf("srt = %s", got.SRT)
	}
}

func waitHealth(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("health timeout")
}
