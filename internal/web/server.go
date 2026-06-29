package web

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/auth"
	webassets "github.com/flow-agent/flow-agent/web"
)

// Server 本地 Agent Studio HTTP 服务。
type Server struct {
	Root        string
	Addr        string
	DesktopMode bool
	AuthCfg     auth.Config
	authStore   *auth.Store
	authSvc     *auth.AuthService
	srv         *http.Server
}

// New 创建 Web 服务；root 为项目根目录。
func New(root, addr string) *Server {
	return &Server{Root: root, Addr: addr, AuthCfg: auth.LoadConfigFromEnv()}
}

// InitAuth 初始化认证（Docker / 云端模式）。
func (s *Server) InitAuth() error {
	if !s.AuthCfg.Enabled {
		return nil
	}
	dataDir := auth.DataDir(s.Root)
	store, err := auth.OpenStore(dataDir)
	if err != nil {
		return fmt.Errorf("auth store: %w", err)
	}
	s.authStore = store
	s.authSvc = auth.NewAuthService(store, auth.NewSMSClient(s.AuthCfg.SMS), s.AuthCfg)
	slog.Info("auth enabled", "data_dir", dataDir, "sms_provider", s.AuthCfg.SMS.Provider)
	return nil
}

// Close 释放资源。
func (s *Server) Close() {
	if s.authStore != nil {
		_ = s.authStore.Close()
	}
}

// ListenAndServe 阻塞监听；addr 为空时使用 s.Addr。
func (s *Server) ListenAndServe() error {
	if err := s.InitAuth(); err != nil {
		return err
	}
	defer s.Close()
	mux := s.Handler()
	s.srv = &http.Server{Addr: s.Addr, Handler: mux}
	fmt.Fprintf(os.Stdout, "FlowAgent Studio: http://%s/\n", s.Addr)
	if s.AuthCfg.Enabled {
		fmt.Fprintf(os.Stdout, "Auth: SMS login enabled (provider=%s)\n", s.AuthCfg.SMS.Provider)
	}
	return s.srv.ListenAndServe()
}

// Start 在 goroutine 中启动服务。
func (s *Server) Start() error {
	if err := s.InitAuth(); err != nil {
		return err
	}
	mux := s.Handler()
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.Addr = ln.Addr().String()
	s.srv = &http.Server{Handler: mux}
	go func() {
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("web server stopped", "err", err)
		}
	}()
	return nil
}

// Shutdown 优雅关闭。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	err := s.srv.Shutdown(ctx)
	s.Close()
	return err
}

// BaseURL 返回服务根 URL。
func (s *Server) BaseURL() string {
	return "http://" + s.Addr + "/"
}

// Handler 返回 HTTP 路由。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	api := &apiHandler{
		root:        s.Root,
		desktopMode: s.DesktopMode,
		authCfg:     s.AuthCfg,
		authStore:   s.authStore,
		authSvc:     s.authSvc,
		smsSendLimit:  auth.NewRateLimiter(8, time.Hour),
		smsLoginLimit: auth.NewRateLimiter(30, 15*time.Minute),
	}

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/auth/status", api.handleAuthStatus)
	mux.HandleFunc("/api/auth/me", api.handleAuthMe)
	mux.HandleFunc("/api/auth/sms/send", api.handleAuthSendSMS)
	mux.HandleFunc("/api/auth/sms/login", api.handleAuthLogin)

	mux.HandleFunc("/api/shot-sizes", api.handleShotSizes)
	mux.HandleFunc("/api/voices", api.handleVoices)
	mux.HandleFunc("/api/themes", api.handleThemes)
	mux.HandleFunc("/api/config/status", api.handleConfigStatus)
	mux.HandleFunc("/api/config/stacks", api.handleConfigStacks)
	mux.HandleFunc("/api/config/media-adapters", api.handleMediaAdapters)
	mux.HandleFunc("/api/user/prefs", api.handleUserPrefs)
	mux.HandleFunc("/api/utils/validate-dir", api.handleValidateDir)
	mux.HandleFunc("/api/utils/open-dir", api.handleOpenDir)
	mux.HandleFunc("/api/dialog/pick-folder", api.handlePickFolder)
	mux.Handle("/api/micro-movie", api.withAuth(api.handleCreateMicroMovie))
	mux.HandleFunc("/api/runs", api.handleListRuns)
	mux.Handle("/api/runs/", api.withAuth(http.HandlerFunc(api.handleRuns)))

	static, err := fs.Sub(webassets.FS, "dist")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", spaHandler(static))
	return corsMiddleware(mux, s.DesktopMode)
}

func (h *apiHandler) withAuth(next http.HandlerFunc) http.Handler {
	return auth.Middleware(h.authCfg, http.HandlerFunc(next))
}

func spaHandler(static fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			serveSPAIndex(w, static)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if f, err := static.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(path, ".") {
			serveSPAIndex(w, static)
			return
		}
		http.NotFound(w, r)
	})
}

func serveSPAIndex(w http.ResponseWriter, static fs.FS) {
	data, err := fs.ReadFile(static, "index.html")
	if err != nil {
		http.Error(w, "frontend not built: run `cd web/ui && npm install && npm run build`", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// PickListenAddr 选择可用本地地址；preferred 为空时用 127.0.0.1:0。
func PickListenAddr(preferred string) (string, error) {
	if preferred == "" {
		preferred = "127.0.0.1:0"
	}
	ln, err := net.Listen("tcp", preferred)
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr, nil
}

// OpenRunDir 在资源管理器中打开 run 目录（Windows/macOS/Linux）。
func OpenRunDir(runDir string) error {
	runDir = filepath.Clean(runDir)
	switch {
	case fileExists(runDir):
	default:
		return fmt.Errorf("run dir not found: %s", runDir)
	}
	return openFolder(runDir)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
