# 服务器部署指南（Docker）

适用于阿里云 / 腾讯云 / 华为云等 Linux 云服务器（推荐 Ubuntu 22.04+，2 核 4G 起，需能访问外网 API）。

## 一、准备服务器

1. 购买云服务器，开放安全组端口：
   - **8080**（直接访问 Web）
   - 或 **80 / 443**（若用 Nginx + 域名）
2. SSH 登录：

```bash
ssh root@你的服务器IP
```

3. 安装 Git（若从仓库拉代码）：

```bash
apt update && apt install -y git
```

## 二、上传代码

**方式 A：Git 克隆**

```bash
cd /opt
git clone <你的仓库地址> flow-agent
cd flow-agent
```

**方式 B：本机打包上传**

在 Windows 开发机项目根目录（**推荐**，会自动把 shell 脚本转为 LF 换行）：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/package-for-linux.ps1
scp flow-agent.tar.gz root@39.102.136.31:/opt/
```

或手动打包（若出现 `bash\r: No such file or directory`，见下方修复）：

```powershell
tar -czf flow-agent.tar.gz --exclude=runs --exclude=node_modules --exclude=.git .
scp flow-agent.tar.gz root@39.102.136.31:/opt/
```

在服务器：

```bash
mkdir -p /opt/flow-agent && cd /opt/flow-agent
tar -xzf /opt/flow-agent.tar.gz
```

## 三、一键部署

若 `./scripts/deploy-server.sh` 报 **`/usr/bin/env: 'bash\r': No such file or directory`**，是 Windows 打包带了 CRLF，先执行：

```bash
sed -i 's/\r$//' scripts/*.sh
```

然后部署（**用 bash 调用**，不要依赖 shebang）：

```bash
cd /opt/flow-agent
bash scripts/deploy-server.sh
```

脚本会自动：

- 安装 Docker（若无）
- 生成 `.env` 与随机 JWT 密钥
- 创建 `data/` 目录与 API 密钥模板
- `docker compose build && docker compose up -d`

## 四、配置 API 密钥（必做）

编辑服务器上的密钥文件：

```bash
nano data/config/providers.local.yaml
```

至少填入你使用的栈所需 Key（如 DeepSeek、百炼 dashscope、火山 volcengine 等），保存后重启：

```bash
docker compose restart
```

## 五、访问与登录

浏览器打开：

```
http://<服务器公网IP>:8080
```

- 首次需 **手机号 + 验证码** 登录
- 默认 dev 短信模式：验证码固定 **123456**（不真发短信）
- 登录后可 **创作台** 生成视频，**历史** 查看过往任务

## 六、数据在哪里

所有数据在宿主机 `./data/`（已挂载进容器 `/data`）：

| 路径 | 说明 |
|------|------|
| `data/runs/` | 每次生成的视频与分镜产物 |
| `data/app.db` | 用户账号 |
| `data/config/providers.local.yaml` | API 密钥 |

**备份**：定期打包 `data/` 目录即可。

## 七、域名 + HTTPS（推荐）

1. 域名 A 记录指向服务器 IP
2. 参考 [deploy/nginx/flowagent.conf](../deploy/nginx/flowagent.conf) 配置 Nginx
3. `certbot --nginx -d 你的域名.com`
4. docker-compose 可改 `FLOWAGENT_PORT=8080` 仅本机监听，对外只走 443

## 八、常用运维命令

```bash
cd /opt/flow-agent

docker compose ps              # 状态
docker compose logs -f         # 日志
docker compose restart         # 重启
docker compose down            # 停止
docker compose up -d --build   # 更新代码后重新构建
```

## 九、防火墙

**Ubuntu ufw：**

```bash
ufw allow 8080/tcp
ufw allow 22/tcp
ufw enable
```

**云控制台**：安全组入站规则同样放行 8080（或 80/443）。

## 十、生产短信（可选）

编辑 `.env`：

```env
FLOWAGENT_SMS_PROVIDER=aliyun
FLOWAGENT_SMS_ACCESS_KEY_ID=...
FLOWAGENT_SMS_ACCESS_KEY_SECRET=...
FLOWAGENT_SMS_SIGN_NAME=...
FLOWAGENT_SMS_TEMPLATE_CODE=...
```

然后 `docker compose up -d`。

---

本地调试（不登录）：`FLOWAGENT_AUTH_ENABLED=false flowagent serve`
