package webassets

import "embed"

// FS 嵌入 Vue 3 构建产物（web/dist）。构建：cd web/ui && npm run build
//
//go:embed dist/index.html dist/assets/*
var FS embed.FS
