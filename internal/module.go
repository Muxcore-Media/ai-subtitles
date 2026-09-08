package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Muxcore-Media/core/pkg/contracts"
	"google.golang.org/grpc"
)

const moduleVersion = "0.1.0"

type Module struct {
	id, grpcAddr, httpAddr string
	autoGenerate           bool
	cfgMu                  sync.RWMutex
	jobs                   []GenerateResult
	syncs                  []SyncResult

	grpcSrv *grpc.Server
	lis     net.Listener
	httpSrv *http.Server
}

type Config struct {
	ID, GRPCAddr, HTTPAddr string
	AutoGenerate           bool
}

func NewModule() *Module { return New(Config{AutoGenerate: true}) }

func New(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "ai-subtitles"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = "127.0.0.1:9762"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = "127.0.0.1:9763"
	}
	if v := os.Getenv("AI_SUBTITLES_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv("MUXCORE_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("AI_SUBTITLES_AUTO"); v != "" {
		cfg.AutoGenerate = v == "1" || v == "true"
	}
	return &Module{
		id: cfg.ID, grpcAddr: cfg.GRPCAddr, httpAddr: cfg.HTTPAddr,
		autoGenerate: cfg.AutoGenerate,
	}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID: m.id, Name: "AI Subtitles", Version: moduleVersion,
		Roles:          []string{"ai"},
		Description:    "AI subtitle generation, sync, and translation when catalog search fails",
		Author:         "Muxcore-Media",
		Capabilities:   []string{"ai.subtitles", "settings"},
		MinCoreVersion: MinCoreVersion,
		HTTPAddr:       m.grpcAddr,
	}
}

func (m *Module) Init(context.Context) error { return nil }

func (m *Module) Start(ctx context.Context) error {
	return serveModule(ctx, m, &m.grpcAddr, &m.httpAddr, &m.lis, &m.grpcSrv, &m.httpSrv, m.registerHTTP)
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
	}
	if m.httpSrv != nil {
		return m.httpSrv.Shutdown(ctx)
	}
	return nil
}

func (m *Module) Health(context.Context) error { return nil }
func (m *Module) GRPCListenAddr() string       { return m.grpcAddr }
func (m *Module) HTTPListenAddr() string       { return m.httpAddr }

func (m *Module) Settings() []contracts.SettingDef {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return []contracts.SettingDef{{
		Key: "auto_generate", Label: "Auto-generate on catalog miss", Type: contracts.SettingTypeBool,
		Value: fmt.Sprintf("%t", m.autoGenerate), Default: "true", Group: "AI",
	}}
}

func (m *Module) UpdateSetting(key, value string) error {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	if key != "auto_generate" {
		return fmt.Errorf("unknown setting %q", key)
	}
	m.autoGenerate = value == "1" || value == "true" || value == "TRUE"
	return nil
}

func (m *Module) rememberGenerate(r GenerateResult) {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	r.JobID = fmt.Sprintf("gen-%d", len(m.jobs)+1)
	m.jobs = append(m.jobs, r)
}

func (m *Module) rememberSync(r SyncResult) {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	r.JobID = fmt.Sprintf("sync-%d", len(m.syncs)+1)
	m.syncs = append(m.syncs, r)
}

func serveModule(
	ctx context.Context,
	m meshSettings,
	grpcAddr, httpAddr *string,
	lis *net.Listener,
	grpcSrv **grpc.Server,
	httpSrv **http.Server,
	register func(*http.ServeMux),
) error {
	var lc net.ListenConfig
	gLis, err := lc.Listen(ctx, "tcp", *grpcAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC %s: %w", *grpcAddr, err)
	}
	*lis = gLis
	*grpcAddr = gLis.Addr().String()
	*grpcSrv = grpc.NewServer()
	registerAIMesh(*grpcSrv, m.meshID(), m)
	go func() {
		slog.Info("gRPC listening", "addr", *grpcAddr)
		if serveErr := (*grpcSrv).Serve(gLis); serveErr != nil {
			slog.Error("gRPC serve", "error", serveErr)
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	register(mux)
	hLis, err := lc.Listen(ctx, "tcp", *httpAddr)
	if err != nil {
		return fmt.Errorf("listen HTTP %s: %w", *httpAddr, err)
	}
	*httpAddr = hLis.Addr().String()
	*httpSrv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		slog.Info("HTTP listening", "addr", *httpAddr)
		if serveErr := (*httpSrv).Serve(hLis); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error("HTTP serve", "error", serveErr)
		}
	}()
	return nil
}

type meshSettings interface {
	meshID() string
	Settings() []contracts.SettingDef
	UpdateSetting(key, value string) error
	handleMesh(ctx context.Context, method string, payload []byte) ([]byte, error)
}

func (m *Module) meshID() string { return m.id }
