package config

import (
	"os"
	"path/filepath"
	"strings"
)

// App 聚合运行时路径与已加载的配置。
type App struct {
	Root         string    // 项目根（含 go.mod）
	ConfigDir    string    // config/
	WorkflowsDir string    // docs/workflows/
	SeriesDir    string    // series/
	RunsDir      string    // runs/
	DataDir      string    // 持久化数据根（Docker volume）
	Providers    Providers // 各云 API 密钥
	Stack        *Stack    // 可选：standard-tier 等
}

// ApplyDataDir 若设置 FLOWAGENT_DATA_DIR，将 runs/series 指向该目录。
func ApplyDataDir(app *App, root string) {
	dataDir := strings.TrimSpace(os.Getenv("FLOWAGENT_DATA_DIR"))
	if dataDir == "" {
		dataDir = root
	}
	app.DataDir = dataDir
	if dataDir != root {
		app.RunsDir = filepath.Join(dataDir, "runs")
		app.SeriesDir = filepath.Join(dataDir, "series")
	}
}

// Load 根据项目根构建 App。stackName 为空时不加载 stack YAML。
func Load(root, stackName string) (*App, error) {
	app := &App{
		Root:         root,
		ConfigDir:    filepath.Join(root, "config"),
		WorkflowsDir: filepath.Join(root, "docs", "workflows"),
		SeriesDir:    filepath.Join(root, "series"),
		RunsDir:      filepath.Join(root, "runs"),
	}
	ApplyDataDir(app, root)

	providersPath := filepath.Join(app.ConfigDir, "providers.local.yaml")
	if dataProviders := filepath.Join(app.DataDir, "config", "providers.local.yaml"); app.DataDir != root {
		if _, err := os.Stat(dataProviders); err == nil {
			providersPath = dataProviders
		}
	}
	providers, err := LoadProviders(providersPath)
	if err != nil {
		providers, err = LoadProviders(filepath.Join(app.ConfigDir, "providers.local.yaml.example"))
		if err != nil {
			providers = Providers{}
		}
	}
	app.Providers = providers

	if stackName != "" {
		stack, err := LoadStack(filepath.Join(app.ConfigDir, "stacks", stackName+".yaml"))
		if err != nil {
			return nil, err
		}
		app.Stack = stack
	}
	return app, nil
}
