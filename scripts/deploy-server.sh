#!/usr/bin/env bash
# FlowAgent 服务器一键部署（Linux，需 root 或 docker 组权限）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Windows 打包的 tar 常带 CRLF，会导致 bash\r 报错；解压后先执行一次即可
fix_crlf() {
  local d
  d="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  shopt -s nullglob
  for f in "$d"/*.sh; do
    sed -i 's/\r$//' "$f" 2>/dev/null || true
  done
}
fix_crlf

echo "==> FlowAgent 服务器部署"
echo "    项目目录: $ROOT"

# 1. Docker
if ! command -v docker >/dev/null 2>&1; then
  echo "==> 安装 Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker
  systemctl start docker
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "==> 未检测到 Docker Compose v2，尝试安装 docker-compose-plugin..."
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq
    apt-get install -y docker-compose-plugin
  elif command -v yum >/dev/null 2>&1; then
    yum install -y docker-compose-plugin
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y docker-compose-plugin
  fi
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "错误: 需要 Docker Compose v2（命令: docker compose）"
  echo "  Ubuntu/Debian: apt-get install -y docker-compose-plugin"
  echo "  或重新安装 Docker: curl -fsSL https://get.docker.com | sh"
  exit 1
fi

echo "==> Docker Compose: $(docker compose version --short 2>/dev/null || docker compose version)"

# 2. 环境变量
if [[ ! -f .env ]]; then
  cp .env.example .env
  SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 32)"
  if grep -q '^FLOWAGENT_JWT_SECRET=change-me-in-production' .env; then
    sed -i "s/^FLOWAGENT_JWT_SECRET=.*/FLOWAGENT_JWT_SECRET=${SECRET}/" .env
  fi
  echo "==> 已生成 .env（JWT 密钥已随机化）"
else
  echo "==> 使用已有 .env"
fi

# 3. 数据目录与 API 密钥
mkdir -p data/config data/runs data/series
PROV="$ROOT/data/config/providers.local.yaml"
if [[ ! -f "$PROV" ]]; then
  cp config/providers.local.yaml.example "$PROV"
  echo "==> 已创建 $PROV"
  echo "    *** 请编辑该文件填入 DeepSeek / 百炼 / 火山等 API Key 后再生成视频 ***"
fi

# 4. 构建并启动
echo "==> 构建镜像（首次约 5–15 分钟）..."
docker compose build

echo "==> 启动服务..."
docker compose up -d

echo ""
echo "==> 部署完成"
PORT="$(grep -E '^FLOWAGENT_PORT=' .env 2>/dev/null | cut -d= -f2 || echo 8080)"
PORT="${PORT:-8080}"
IP="$(curl -fsS --max-time 3 ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo '<服务器公网IP>')"
echo ""
echo "  访问地址: http://${IP}:${PORT}"
echo "  健康检查: curl http://127.0.0.1:${PORT}/api/health"
echo ""
echo "  默认登录（dev 短信）: 任意手机号 + 验证码 123456"
echo ""
echo "  常用命令:"
echo "    docker compose logs -f          # 查看日志"
echo "    docker compose restart          # 重启"
echo "    docker compose down             # 停止"
echo "    编辑 API 密钥: nano data/config/providers.local.yaml"
echo ""
