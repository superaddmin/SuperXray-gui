# 服务器环境部署教程

> **目标读者**：准备在 VPS、云服务器或自建 Linux 主机上部署 SuperXray 的运维人员 / 开发者
> **适用版本**：`v3.0.2`
> **相关文档**：[系统架构设计](architecture.md) | [核心模块解析](modules.md) | [API 接口说明](api.md)

---

## 1. 部署前必须了解

SuperXray 是一个基于 Web 的 Xray-core 管理面板，后端使用 Go 编写，默认使用 SQLite 保存面板配置和入站数据。服务器部署时通常包含三部分：

| 组件 | 默认位置 / 端口 | 说明 |
|------|----------------|------|
| 面板程序 | `/usr/local/x-ui/x-ui` | Go 编译后的主程序 |
| 管理脚本 | `/usr/bin/x-ui` | 安装、更新、重启、证书、防火墙等菜单入口 |
| 数据库 | `/etc/x-ui/x-ui.db` | SQLite 数据库，必须备份 |
| 日志 | `/var/log/x-ui/3xui.log` | 应用日志 |
| Web 面板 | 默认 `2053` | 源码默认端口；一键安装时可能被随机化 |
| 订阅服务 | 默认 `2096` | Base64、JSON、Clash/Mihomo 订阅入口 |
| Xray API | 默认本地端口 | 由程序内部管理，不应对公网开放 |

> README 中明确提示本项目仅建议个人使用，请勿用于非法用途，也不建议直接作为高风险生产环境组件。服务器部署时请至少启用 HTTPS、强密码、随机 Web 路径、最小端口暴露和定期备份。

---

## 2. 推荐部署方案

| 方案 | 适用场景 | 优点 | 注意事项 |
|------|----------|------|----------|
| 一键脚本安装 | 大多数 Linux 服务器 | 自动安装依赖、下载 Release、配置服务、生成安全初始信息 | 需要服务器能访问 GitHub；安装过程会交互式询问端口和证书 |
| GHCR 镜像部署 | 已有容器化运维体系 | 直接拉取官方镜像，便于迁移和回滚 | 使用 `network_mode: host`，且不会自动随机化初始账号 |
| 本地 Docker Compose 构建 | 需要验证本地源码或自定义镜像 | 与仓库 `docker-compose.yml` 保持一致 | 需要本机具备 Docker Buildx，更新时需重新构建 |
| 手动构建 / 二进制部署 | 二次开发、内网构建、审计后上线 | 可完全控制 amd64 / arm64 构建产物 | 当前官方 Release 只发布 Linux `amd64` / `arm64` |

建议优先选择 **一键脚本安装**。如果服务器不能访问 GitHub，可以在可访问网络的机器上下载 Release 包和脚本后再离线分发。

---

## 3. 服务器准备

### 3.1 硬件与系统

| 项目 | 最低建议 | 推荐配置 |
|------|----------|----------|
| CPU | 1 核 | 2 核及以上 |
| 内存 | 512 MB | 1 GB 及以上 |
| 磁盘 | 1 GB 可用空间 | 5 GB 及以上，便于日志和备份 |
| 网络 | 公网 IPv4 或 IPv6 | 独立公网 IP + 稳定 DNS |
| 系统 | Debian 10+ / Ubuntu 20.04+ / RHEL 8+ / Arch / Alpine | Debian 12 或 Ubuntu 22.04 LTS |

### 3.2 域名与端口规划

生产或准生产环境建议使用域名访问面板，例如 `panel.example.com`。部署前确认：

- 域名 `A` / `AAAA` 记录已指向服务器公网 IP。
- 云厂商安全组已放行 SSH 端口、HTTP `80`、HTTPS `443`，以及你计划使用的代理端口。
- 如果直接暴露面板，需要放行面板端口；如果使用反向代理，面板可以只监听 `127.0.0.1`。
- 申请 Let's Encrypt 证书时，HTTP-01 验证通常需要公网可访问 `80/tcp`。

常见端口规划示例：

| 用途 | 示例端口 | 是否公网开放 | 说明 |
|------|----------|--------------|------|
| SSH | `22` 或自定义 | 是，建议仅允许可信 IP | 服务器管理入口 |
| ACME HTTP-01 | `80/tcp` | 是 | 申请/续期证书 |
| HTTPS 反向代理 | `443/tcp` | 是 | 推荐的面板公网入口 |
| Web 面板 | `2053/tcp` 或随机端口 | 反代时不开放 | 一键脚本可能随机生成 |
| 订阅服务 | `2096/tcp` | 按需 | 可通过反代统一到 `443` |
| Xray 入站 | 自定义 | 是 | 按协议需要放行 TCP/UDP |

### 3.3 基础依赖

Debian / Ubuntu：

```bash
sudo apt-get update
sudo apt-get install -y curl tar tzdata socat ca-certificates openssl ufw
sudo timedatectl set-ntp true
```

RHEL / Rocky / AlmaLinux / Fedora：

```bash
sudo dnf install -y curl tar cronie tzdata socat ca-certificates openssl firewalld
sudo systemctl enable --now firewalld
sudo timedatectl set-ntp true
```

Arch Linux：

```bash
sudo pacman -Syu --noconfirm curl tar cronie tzdata socat ca-certificates openssl
sudo systemctl enable --now cronie
sudo timedatectl set-ntp true
```

Alpine Linux：

```bash
apk update
apk add dcron curl tar tzdata socat ca-certificates openssl
rc-update add crond
rc-service crond start
```

Docker / Compose：

```bash
docker version
docker compose version
```

建议使用 Docker Engine 24+ 和 Compose plugin。容器部署默认使用宿主机网络，需要在宿主机安全组和防火墙中放行面板、订阅和 Xray 入站端口。拉取官方镜像时还需要服务器能访问 `ghcr.io`。

### 3.4 确认服务器 CPU 架构

本项目的 Linux Release 包和 Xray 二进制需要与服务器 CPU 架构一致。当前官方 Release 默认发布 `x-ui-linux-amd64.tar.gz` 和 `x-ui-linux-arm64.tar.gz` 两个包。Ubuntu 服务器部署前先确认架构：

```bash
uname -m
dpkg --print-architecture 2>/dev/null || true
```

| `uname -m` 输出 | Debian / Ubuntu 架构名 | 本项目 Release / 二进制后缀 |
|-----------------|-------------------------|------------------------------|
| `x86_64` | `amd64` | `linux-amd64` |
| `aarch64` / `arm64` | `arm64` | `linux-arm64` |

---

## 4. 一键脚本安装（推荐）

### 4.1 执行安装

建议先进入 root shell，避免 `sudo` 与 Bash 进程替换语法产生权限问题：

```bash
sudo -i
apt-get update
apt-get install -y curl tar tzdata socat ca-certificates openssl
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh)
```

如需指定二进制 Release 下载源，可在命令前设置 `XUI_RELEASE_REPO=仓库所有者/仓库名`。默认下载 `superaddmin/SuperXray-gui` 的正式 Release。
如 GitHub latest 正式版接口暂时不可用，也可以显式指定版本号安装：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh) v3.0.2
```

安装脚本会自动把 `x86_64` / `amd64` 映射为 `amd64`，把 `aarch64` / `arm64` 映射为 `arm64`，并下载对应的 `x-ui-linux-<arch>.tar.gz`。

脚本会执行以下操作：

1. 检测系统发行版和 CPU 架构。
2. 安装基础依赖：`cron` / `cronie`、`curl`、`tar`、`tzdata`、`socat`、`ca-certificates`、`openssl`。
3. 从 GitHub Release 下载对应架构的 `x-ui-linux-<arch>.tar.gz`。
4. 安装主程序到 `/usr/local/x-ui/`。
5. 安装管理脚本到 `/usr/bin/x-ui`。
6. 创建 `/var/log/x-ui` 日志目录。
7. 安装 systemd unit 或 Alpine OpenRC 服务。
8. 初始化数据库并执行迁移。
9. 检测默认凭据和过短的 `webBasePath`，必要时生成随机用户名、密码、端口和 Web 路径。
10. 引导配置 SSL 证书。

### 4.2 安装过程中的关键选择

脚本检测到默认账号 `admin/admin` 或默认路径 `/` 时，会自动加强初始安全配置：

- 生成随机用户名。
- 生成随机密码。
- 生成随机 `webBasePath`。
- 询问是否自定义面板端口；如果不自定义，会随机选择 `1024-62000` 内的端口。
- 引导配置 SSL 证书。

安装完成时终端会输出类似信息：

```text
Username:    <随机用户名>
Password:    <随机密码>
Port:        <面板端口>
WebBasePath: <随机路径>
Access URL:  https://<域名或IP>:<端口>/<随机路径>
```

这段输出只出现一次，务必保存到密码管理器。如果忘记，可以通过 `x-ui` 管理菜单重置用户名、密码和 Web 路径。

### 4.3 SSL 证书选择

安装脚本支持三类证书方式：

| 方式 | 适用场景 | 前置条件 |
|------|----------|----------|
| 域名证书 | 推荐方式 | 域名已解析到服务器，`80/tcp` 可公网访问 |
| IP 证书 | 没有域名时临时使用 | `80/tcp` 可公网访问；证书有效期较短，依赖自动续期 |
| 自定义证书 | 已有证书或内网 CA | 准备好证书文件和私钥文件 |

域名证书默认安装路径：

```text
/root/cert/<domain>/fullchain.pem
/root/cert/<domain>/privkey.pem
```

脚本会把证书路径写入面板配置，并在证书续期后重启 `x-ui`。

### 4.4 安装后检查

```bash
x-ui status
x-ui settings
systemctl status x-ui --no-pager
journalctl -u x-ui -n 100 --no-pager
tail -n 100 /var/log/x-ui/3xui.log
```

直接查看面板内部配置可以调用主程序：

```bash
/usr/local/x-ui/x-ui setting -show true
/usr/local/x-ui/x-ui setting -getListen true
/usr/local/x-ui/x-ui setting -getCert true
```

> `x-ui` 是管理脚本，支持 `x-ui start`、`x-ui restart`、`x-ui settings` 等运维子命令；面板二进制的设置命令请使用 `/usr/local/x-ui/x-ui setting ...`。

### 4.5 常用管理命令

| 命令 | 中文说明 |
|------|----------|
| `x-ui` | 打开交互式管理菜单 |
| `x-ui start` | 启动面板服务 |
| `x-ui stop` | 停止面板服务 |
| `x-ui restart` | 重启面板服务 |
| `x-ui restart-xray` | 仅重启 Xray 内核 |
| `x-ui status` | 查看当前运行状态 |
| `x-ui settings` | 查看当前配置和访问地址 |
| `x-ui enable` | 开启开机自启 |
| `x-ui disable` | 关闭开机自启 |
| `x-ui log` | 查看面板日志 |
| `x-ui banlog` | 查看 fail2ban 封禁日志 |
| `x-ui update` | 更新面板程序 |
| `x-ui update-all-geofiles` | 更新全部 GeoIP/GeoSite 数据文件 |
| `x-ui legacy` | 安装指定历史版本 |
| `x-ui install` | 安装面板 |
| `x-ui uninstall` | 卸载面板 |

常用二进制设置命令：

```bash
/usr/local/x-ui/x-ui setting -username '<新用户名>' -password '<新密码>'
/usr/local/x-ui/x-ui setting -port 2053
/usr/local/x-ui/x-ui setting -webBasePath '/panel-secret/'
/usr/local/x-ui/x-ui setting -listenIP 127.0.0.1
/usr/local/x-ui/x-ui setting -resetTwoFactor
/usr/local/x-ui/x-ui cert -webCert /root/cert/example.com/fullchain.pem -webCertKey /root/cert/example.com/privkey.pem
/usr/local/x-ui/x-ui cert -reset
systemctl restart x-ui
```

---

## 5. 防火墙与云安全组

### 5.1 Ubuntu / Debian 使用 UFW

反向代理部署只需要对公网开放 SSH、`80`、`443` 和 Xray 入站端口：

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow <Xray端口>/tcp
sudo ufw allow <Xray端口>/udp
sudo ufw enable
sudo ufw status verbose
```

如果你不使用反向代理，而是直接访问面板端口，还需要开放面板端口：

```bash
sudo ufw allow <面板端口>/tcp
```

订阅服务如果不走反向代理，也需要开放：

```bash
sudo ufw allow 2096/tcp
```

### 5.2 RHEL 系使用 firewalld

```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --add-port=<Xray端口>/tcp
sudo firewall-cmd --permanent --add-port=<Xray端口>/udp
sudo firewall-cmd --reload
sudo firewall-cmd --list-all
```

### 5.3 使用脚本菜单管理防火墙

也可以运行：

```bash
x-ui
```

选择 `22. Firewall Management`。该菜单主要围绕 UFW 工作，适合 Debian / Ubuntu 系服务器；RHEL 系建议直接使用 `firewall-cmd`。

---

## 6. HTTPS 与反向代理

### 6.1 推荐拓扑

推荐让 Nginx 或 Caddy 监听公网 `443`，面板仅监听本机：

```text
Internet -> Nginx/Caddy :443 -> 127.0.0.1:<面板端口> -> SuperXray
Internet -> Nginx/Caddy :443/sub/ -> 127.0.0.1:2096/sub/ -> 订阅服务
Internet -> <Xray端口> -> Xray 入站
```

把面板限制到本机监听：

```bash
/usr/local/x-ui/x-ui setting -listenIP 127.0.0.1
/usr/local/x-ui/x-ui setting -webBasePath '/panel-secret/'
systemctl restart x-ui
```

如果已在面板内配置证书，但计划由 Nginx/Caddy 统一终止 TLS，可以清空面板证书：

```bash
/usr/local/x-ui/x-ui cert -reset
systemctl restart x-ui
```

### 6.2 Nginx 示例

以下示例假设：

- 面板端口为 `2053`。
- 面板 `webBasePath` 为 `/panel-secret/`。
- 订阅服务保持默认端口 `2096`。
- Nginx 负责公网 HTTPS。

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name panel.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name panel.example.com;

    ssl_certificate /etc/letsencrypt/live/panel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/panel.example.com/privkey.pem;

    location /panel-secret/ {
        proxy_pass http://127.0.0.1:2053/panel-secret/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /sub/ {
        proxy_pass http://127.0.0.1:2096/sub/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /json/ {
        proxy_pass http://127.0.0.1:2096/json/;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /clash/ {
        proxy_pass http://127.0.0.1:2096/clash/;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

启用配置：

```bash
sudo nginx -t
sudo systemctl reload nginx
```

### 6.3 Caddy 示例

```caddyfile
panel.example.com {
    handle /panel-secret/* {
        reverse_proxy 127.0.0.1:2053
    }

    handle /sub/* {
        reverse_proxy 127.0.0.1:2096
    }

    handle /json/* {
        reverse_proxy 127.0.0.1:2096
    }

    handle /clash/* {
        reverse_proxy 127.0.0.1:2096
    }
}
```

校验并重载：

```bash
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy
```

### 6.4 Cloudflare 注意事项

- 面板站点 SSL 模式建议使用 `Full` 或 `Full (strict)`。
- WebSocket 需要保持开启。
- 如果代理 Xray 流量，只能选择 Cloudflare 支持的端口和传输方式。
- Reality、裸 TCP、UDP、Hysteria 等协议通常不应直接走普通 Cloudflare HTTP 代理。
- 面板路径不要使用简单的 `/`，建议使用随机 `webBasePath`。

---

## 7. Docker / GHCR 部署

### 7.1 方案 A：直接使用 GHCR 镜像

适合只想运行官方容器镜像的服务器。当前镜像发布到 GitHub Container Registry：

```text
ghcr.io/superaddmin/superxray-gui:3.0.2
ghcr.io/superaddmin/superxray-gui:latest
```

快速启动：

```bash
mkdir -p db cert
docker run -d --name superxray-gui --network host --restart unless-stopped \
  -v $PWD/db:/etc/x-ui \
  -v $PWD/cert:/root/cert \
  -e XRAY_VMESS_AEAD_FORCED=false \
  -e XUI_ENABLE_FAIL2BAN=true \
  ghcr.io/superaddmin/superxray-gui:3.0.2
```

使用 Compose 时建议写成：

```yaml
services:
  3xui:
    image: ghcr.io/superaddmin/superxray-gui:3.0.2
    container_name: 3xui_app
    volumes:
      - $PWD/db/:/etc/x-ui/
      - $PWD/cert/:/root/cert/
    environment:
      XRAY_VMESS_AEAD_FORCED: "false"
      XUI_ENABLE_FAIL2BAN: "true"
    tty: true
    network_mode: host
    restart: unless-stopped
```

启动和查看日志：

```bash
docker compose up -d
docker compose ps
docker compose logs -f 3xui
```

### 7.2 方案 B：使用仓库 Compose 本地构建

仓库自带的 [`docker-compose.yml`](../docker-compose.yml) 当前使用 `build: .` 从本地源码构建镜像，适合验证本地修改或自定义镜像：

```bash
git clone https://github.com/superaddmin/SuperXray-gui.git
cd SuperXray-gui
docker compose up -d --build
```

如果想复用官方 GHCR 镜像，把 Compose 中的 `build:` 段替换为：

```yaml
image: ghcr.io/superaddmin/superxray-gui:3.0.2
```

### 7.3 数据目录

容器部署默认挂载：

```yaml
volumes:
  - $PWD/db/:/etc/x-ui/
  - $PWD/cert/:/root/cert/
network_mode: host
```

需要备份的目录：

```text
./db/x-ui.db
./cert/
```

### 7.4 Docker 初始安全设置

Docker 入口不会执行一键脚本里的随机化配置。首次启动后，源码默认账号仍可能是 `admin/admin`、面板端口仍是 `2053`、路径仍是 `/`。请立即修改：

以下命令按 Compose 示例的 `container_name: 3xui_app` 编写；如果使用上面的 `docker run --name superxray-gui`，请把 `3xui_app` 替换为 `superxray-gui`。

```bash
docker exec -it 3xui_app /app/x-ui setting -show true
docker exec -it 3xui_app /app/x-ui setting -username '<新用户名>' -password '<强密码>' -webBasePath '/panel-secret/'
docker exec -it 3xui_app /app/x-ui setting -port 2053
docker compose restart 3xui
```

如果使用 `docker run` 而不是 Compose，下面涉及 `docker compose restart 3xui` 的步骤都改为：

```bash
docker restart superxray-gui
```

配置证书：

```bash
mkdir -p cert
cp /path/to/fullchain.pem cert/fullchain.pem
cp /path/to/privkey.pem cert/privkey.pem
docker exec -it 3xui_app /app/x-ui cert -webCert /root/cert/fullchain.pem -webCertKey /root/cert/privkey.pem
docker compose restart 3xui
```

### 7.5 Docker 更新与备份

更新前先备份数据库和证书：

```bash
mkdir -p backup
cp -a db "backup/db-$(date +%F-%H%M%S)"
cp -a cert "backup/cert-$(date +%F-%H%M%S)"
```

GHCR 镜像部署：

```bash
docker compose pull
docker compose up -d
```

本地源码构建部署：

```bash
git pull
docker compose up -d --build
```

---

## 8. Ubuntu 手动安装与部署（amd64 / arm64）

手动部署适合服务器无法直接运行一键脚本、需要审计 Release 包、需要上传内网构建产物，或需要在目标 Ubuntu 服务器上自行编译的场景。下面命令默认以 Ubuntu 20.04+ / 22.04+ / 24.04+、systemd、安装目录 `/usr/local/x-ui` 为准，并只覆盖 Linux `amd64` 与 `arm64` 两种服务器架构。

### 8.1 进入 root shell 并准备基础环境

后续命令建议在同一个 root shell 中执行，避免环境变量丢失：

```bash
sudo -i
apt-get update
apt-get install -y curl tar unzip ca-certificates openssl git build-essential pkg-config
timedatectl set-ntp true
```

设置本次部署使用的架构变量：

```bash
case "$(uname -m)" in
  x86_64|amd64)
    XUI_ARCH="amd64"
    GO_ARCH="amd64"
    ;;
  aarch64|arm64)
    XUI_ARCH="arm64"
    GO_ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

echo "Deploy architecture: ${XUI_ARCH}"
```

### 8.2 方案 A：手动下载 GitHub Release 包安装

如果只需要部署官方构建产物，优先使用 Release 包。当前官方 Release 包名为 `x-ui-linux-amd64.tar.gz` 和 `x-ui-linux-arm64.tar.gz`，已经包含 `x-ui`、`x-ui.sh`、systemd unit、Xray 二进制和规则数据文件。

```bash
TAG="$(curl -fsSL https://api.github.com/repos/superaddmin/SuperXray-gui/releases/latest \
  | grep '"tag_name":' \
  | sed -E 's/.*"([^"]+)".*/\1/')"

if [ -z "${TAG}" ]; then
  echo "Failed to resolve latest release tag. Set TAG manually, for example: TAG=v2.x.x" >&2
  exit 1
fi

curl -fL -o "/tmp/x-ui-linux-${XUI_ARCH}.tar.gz" \
  "https://github.com/superaddmin/SuperXray-gui/releases/download/${TAG}/x-ui-linux-${XUI_ARCH}.tar.gz"

rm -rf /tmp/x-ui
tar -xzf "/tmp/x-ui-linux-${XUI_ARCH}.tar.gz" -C /tmp
```

安装到系统目录：

```bash
systemctl stop x-ui 2>/dev/null || true

install -d -m 0755 /usr/local/x-ui
install -d -m 0700 /etc/x-ui
install -d -m 0750 /var/log/x-ui

rm -rf /usr/local/x-ui/*
cp -a /tmp/x-ui/. /usr/local/x-ui/

chmod 0755 /usr/local/x-ui/x-ui
chmod 0755 "/usr/local/x-ui/bin/xray-linux-${XUI_ARCH}"
install -m 0755 /usr/local/x-ui/x-ui.sh /usr/bin/x-ui
install -m 0644 /usr/local/x-ui/x-ui.service.debian /etc/systemd/system/x-ui.service
```

完成复制后跳到 [8.5 写入环境配置并启动服务](#85-写入环境配置并启动服务)。

### 8.3 方案 B：在 Ubuntu 目标服务器上源码构建

源码构建适合你已经修改代码、需要在目标架构上原生编译，或不想直接使用 Release 包的场景。由于项目使用 CGO 和 SQLite，建议在目标服务器本机编译；跨架构构建需要额外交叉编译工具链。

```bash
cd /opt
git clone https://github.com/superaddmin/SuperXray-gui.git
cd SuperXray-gui

GO_VERSION="$(awk '/^go / {print $2}' go.mod)"

if ! command -v go >/dev/null 2>&1 || ! go version | grep -q "go${GO_VERSION}"; then
  curl -fL -o "/tmp/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" \
    "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
fi

export PATH="/usr/local/go/bin:${PATH}"
go version
go mod download
CGO_ENABLED=1 GOOS=linux GOARCH="${GO_ARCH}" go build -ldflags "-w -s" -o build/x-ui main.go
sh ./DockerInit.sh "${XUI_ARCH}"
```

`DockerInit.sh` 会下载对应架构的 Xray-core，并把 `geoip.dat`、`geosite.dat`、`geoip_IR.dat`、`geosite_IR.dat`、`geoip_RU.dat`、`geosite_RU.dat` 放入 `build/bin/`。

安装源码构建产物：

```bash
systemctl stop x-ui 2>/dev/null || true

install -d -m 0755 /usr/local/x-ui
install -d -m 0755 /usr/local/x-ui/bin
install -d -m 0700 /etc/x-ui
install -d -m 0750 /var/log/x-ui

rm -rf /usr/local/x-ui/*
install -m 0755 build/x-ui /usr/local/x-ui/x-ui
cp -a build/bin /usr/local/x-ui/
chmod 0755 "/usr/local/x-ui/bin/xray-linux-${XUI_ARCH}"
install -m 0755 x-ui.sh /usr/bin/x-ui
install -m 0644 x-ui.service.debian /etc/systemd/system/x-ui.service
```

### 8.4 方案 C：上传本地或 CI 构建产物

如果已经在本地或 CI 中得到 `x-ui-linux-amd64.tar.gz` / `x-ui-linux-arm64.tar.gz`，直接上传对应架构的包到服务器，然后复用 8.2 的解压和安装步骤：

```bash
# amd64 服务器
scp x-ui-linux-amd64.tar.gz root@<server>:/tmp/

# arm64 服务器
scp x-ui-linux-arm64.tar.gz root@<server>:/tmp/
```

如果只有面板主程序二进制，例如 `dist/SuperXray-gui-linux-amd64` 或 `dist/SuperXray-gui-linux-arm64`，还需要同时上传服务脚本和 Xray 下载脚本：

```bash
# amd64 服务器
scp dist/SuperXray-gui-linux-amd64 root@<server>:/tmp/SuperXray-gui-linux-amd64

# arm64 服务器
scp dist/SuperXray-gui-linux-arm64 root@<server>:/tmp/SuperXray-gui-linux-arm64

# 两种架构都需要上传这些部署脚本
scp x-ui.sh x-ui.service.debian DockerInit.sh root@<server>:/tmp/
```

在服务器 root shell 中安装上传的主程序：

```bash
case "${XUI_ARCH}" in
  amd64)
    PANEL_BIN="/tmp/SuperXray-gui-linux-amd64"
    ;;
  arm64)
    PANEL_BIN="/tmp/SuperXray-gui-linux-arm64"
    ;;
esac

systemctl stop x-ui 2>/dev/null || true

install -d -m 0755 /usr/local/x-ui
install -d -m 0755 /usr/local/x-ui/bin
install -d -m 0700 /etc/x-ui
install -d -m 0750 /var/log/x-ui

rm -rf /usr/local/x-ui/*
install -m 0755 "${PANEL_BIN}" /usr/local/x-ui/x-ui
install -d -m 0755 /usr/local/x-ui/bin
install -m 0755 /tmp/x-ui.sh /usr/bin/x-ui
install -m 0644 /tmp/x-ui.service.debian /etc/systemd/system/x-ui.service

cd /tmp
rm -rf /tmp/build
sh /tmp/DockerInit.sh "${XUI_ARCH}"
cp -a /tmp/build/bin/. /usr/local/x-ui/bin/
chmod 0755 "/usr/local/x-ui/bin/xray-linux-${XUI_ARCH}"
```

### 8.5 写入环境配置并启动服务

三种手动部署方式都建议显式写入 `/etc/default/x-ui`，与 `x-ui.service.debian` 中的 `EnvironmentFile=-/etc/default/x-ui` 对齐：

```bash
cat >/etc/default/x-ui <<'EOF'
XRAY_VMESS_AEAD_FORCED=false
XUI_LOG_LEVEL=info
XUI_DB_FOLDER=/etc/x-ui
XUI_LOG_FOLDER=/var/log/x-ui
XUI_BIN_FOLDER=bin
EOF
chmod 0644 /etc/default/x-ui
```

初始化数据库和面板安全信息。请替换成自己的用户名、强密码和随机 Web 路径：

```bash
/usr/local/x-ui/x-ui migrate
/usr/local/x-ui/x-ui setting \
  -username '<新用户名>' \
  -password '<强密码>' \
  -port 2053 \
  -webBasePath '/panel-secret/'

systemctl daemon-reload
systemctl enable --now x-ui
systemctl status x-ui --no-pager
```

如果要让面板只通过 Nginx / Caddy 反向代理访问，可以在启动前增加：

```bash
/usr/local/x-ui/x-ui setting -listenIP 127.0.0.1
systemctl restart x-ui
```

### 8.6 验证与卸载

验证服务状态、版本、配置和日志：

```bash
x-ui status
x-ui settings
/usr/local/x-ui/x-ui -v
/usr/local/x-ui/x-ui setting -show true
systemctl is-active x-ui
journalctl -u x-ui -n 100 --no-pager
tail -n 100 /var/log/x-ui/3xui.log
ss -tulpen | grep -E 'x-ui|xray|:2053|:2096' || true
```

卸载 systemd 部署时，先备份数据库和证书，再停止并删除服务文件：

```bash
BACKUP_DIR="/root/x-ui-backup-$(date +%F-%H%M%S)"
mkdir -p "${BACKUP_DIR}"
cp -a /etc/x-ui "${BACKUP_DIR}/" 2>/dev/null || true

systemctl disable --now x-ui 2>/dev/null || true
rm -f /etc/systemd/system/x-ui.service /usr/bin/x-ui
rm -rf /usr/local/x-ui
systemctl daemon-reload
```

---

## 9. 面板首次配置

登录面板后建议按顺序完成：

1. **确认账号安全**：修改用户名、强密码，并启用双因素认证。
2. **确认访问路径**：`webBasePath` 不要使用 `/`，使用随机路径。
3. **确认 HTTPS**：使用脚本证书、反向代理证书或自定义证书。
4. **确认监听地址**：反向代理场景建议 Web 面板监听 `127.0.0.1`。
5. **设置订阅服务**：确认订阅端口、路径、是否启用 JSON / Clash 输出。
6. **创建 Xray 入站**：按协议需要开放端口，TCP/UDP 不要漏放行。
7. **检查 Xray 状态**：面板首页应显示 Xray Running。
8. **导出一个客户端配置测试连通性**。

常用默认设置参考：

| 设置 | 默认值 | 建议 |
|------|--------|------|
| Web 端口 | `2053` | 可随机或仅本机监听 |
| Web 路径 | `/` | 改为随机路径 |
| 订阅端口 | `2096` | 可通过反代收敛到 `443` |
| 订阅路径 | `/sub/` | 可按需修改 |
| JSON 路径 | `/json/` | 如不用可关闭 |
| Clash 路径 | `/clash/` | 如不用可关闭 |
| Session 过期 | `360` 分钟 | 按安全要求缩短 |
| Telegram Bot | 关闭 | 需要通知/备份时再启用 |
| LDAP | 关闭 | 企业统一认证时再启用 |

---

## 10. 备份与恢复

### 10.1 systemd 部署备份

最重要的文件是 `/etc/x-ui/x-ui.db`。建议在停止服务后复制，避免 SQLite 写入中途备份不一致：

```bash
sudo install -d -m 0700 /backup/SuperXray
sudo systemctl stop x-ui
sudo cp -a /etc/x-ui/x-ui.db "/backup/SuperXray/x-ui-$(date +%F-%H%M%S).db"
sudo systemctl start x-ui
```

如果使用了面板证书，也备份证书目录：

```bash
sudo cp -a /root/cert "/backup/SuperXray/cert-$(date +%F-%H%M%S)"
```

### 10.2 systemd 部署恢复

```bash
sudo systemctl stop x-ui
sudo cp -a /backup/SuperXray/x-ui-YYYY-MM-DD-HHMMSS.db /etc/x-ui/x-ui.db
sudo chown root:root /etc/x-ui/x-ui.db
sudo chmod 0600 /etc/x-ui/x-ui.db
sudo /usr/local/x-ui/x-ui migrate
sudo systemctl start x-ui
```

恢复后检查：

```bash
x-ui status
/usr/local/x-ui/x-ui setting -show true
```

### 10.3 Docker 部署备份

```bash
docker compose stop 3xui
mkdir -p backup
cp -a db "backup/db-$(date +%F-%H%M%S)"
cp -a cert "backup/cert-$(date +%F-%H%M%S)"
docker compose start 3xui
```

恢复时停止容器，把备份目录复制回 `db/` 和 `cert/`，再启动容器。

---

## 11. 更新与维护

### 11.1 一键脚本部署更新

```bash
x-ui update
x-ui restart
```

更新 GeoIP/GeoSite 数据：

```bash
x-ui update-all-geofiles
```

### 11.2 Docker 部署更新

```bash
cp -a db "backup/db-$(date +%F-%H%M%S)"
git pull
docker compose up -d --build
docker compose logs -f 3xui
```

### 11.3 日常巡检

```bash
systemctl is-active x-ui
x-ui status
journalctl -u x-ui -n 100 --no-pager
tail -n 100 /var/log/x-ui/3xui.log
ss -tulpen | grep -E 'x-ui|xray|:2053|:2096'
df -h
free -h
```

建议定期检查：

- 证书是否快过期。
- 面板是否仍在公网直接暴露。
- 代理入站端口是否与防火墙规则一致。
- 数据库备份是否可恢复。
- 日志是否异常增长。

---

## 12. 常见故障排查

### 12.1 访问不到面板

先确认服务和端口：

```bash
x-ui status
/usr/local/x-ui/x-ui setting -show true
ss -tulpen | grep x-ui
journalctl -u x-ui -n 100 --no-pager
```

常见原因：

- 访问 URL 漏了随机 `webBasePath`。
- 面板端口被随机化，不再是 `2053`。
- 防火墙或云安全组未放行端口。
- 已设置 `listenIP=127.0.0.1`，但仍在公网直接访问面板端口。
- 证书配置错误导致 HTTPS 握手失败。

### 12.2 忘记用户名、密码或 Web 路径

运行交互式菜单：

```bash
x-ui
```

使用：

- `6. Reset Username & Password`
- `7. Reset Web Base Path`
- `10. View Current Settings`

也可以直接调用二进制：

```bash
/usr/local/x-ui/x-ui setting -username '<新用户名>' -password '<新密码>' -resetTwoFactor
/usr/local/x-ui/x-ui setting -webBasePath '/new-secret-path/'
systemctl restart x-ui
```

### 12.3 证书申请失败

检查：

```bash
dig +short panel.example.com
curl -I http://panel.example.com
ss -tulpen | grep ':80'
journalctl -u x-ui -n 100 --no-pager
```

常见原因：

- 域名没有解析到当前服务器。
- 云安全组或本机防火墙未开放 `80/tcp`。
- Nginx/Caddy 已占用 `80`，但脚本使用 standalone 模式申请证书。
- IPv6 AAAA 记录指向错误地址。
- CDN 代理影响 ACME 验证。

如果 Nginx/Caddy 已经负责证书，推荐在反向代理层申请和续期证书，面板内部使用 HTTP 监听 `127.0.0.1`。

### 12.4 Xray 未运行

```bash
x-ui restart-xray
journalctl -u x-ui -n 200 --no-pager
tail -n 200 /var/log/x-ui/3xui.log
ss -tulpen | grep <Xray端口>
```

常见原因：

- 入站端口冲突。
- 防火墙只放行 TCP，但协议需要 UDP。
- Xray 配置 JSON 不合法。
- Xray 二进制缺失或权限不正确。
- GeoIP/GeoSite 文件缺失。

可以通过面板的 Xray 配置页面查看和回滚配置，或运行：

```bash
x-ui update-all-geofiles
x-ui restart-xray
```

### 12.5 GitHub 下载失败

一键脚本和构建流程都需要访问 GitHub Release。可选处理：

- 如果提示 `获取 SuperXray Release 版本失败`，先打开 `https://github.com/superaddmin/SuperXray-gui/releases` 确认是否存在可下载的 Release。
- 如果仓库只有预发布版本，使用显式版本安装命令，例如：`bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh) v3.0.2`。
- 换用可访问 GitHub 的网络环境。
- 手动下载 Release 包后上传到服务器。
- 使用企业内网制品库缓存 Release 包。
- 手动构建并复制 `/usr/local/x-ui`、`/usr/bin/x-ui`、systemd unit 和 `bin/` 目录。
- Docker 拉取失败时，确认服务器能访问 `ghcr.io/superaddmin/superxray-gui`，或改用本地 `docker compose up -d --build` 构建。

### 12.6 Docker 容器启动后仍是默认账号

这是正常现象：Docker 入口不会执行一键脚本的随机化安全初始化。立即执行：

```bash
docker exec -it 3xui_app /app/x-ui setting -username '<新用户名>' -password '<强密码>' -webBasePath '/panel-secret/'
docker compose restart 3xui
```

---

## 13. 安全加固清单

- 使用随机用户名和强密码，避免 `admin/admin`。
- 使用随机 `webBasePath`，不要把面板放在 `/`。
- 启用 HTTPS；优先使用域名证书。
- 反向代理场景下让面板监听 `127.0.0.1`。
- 只开放必要端口；代理协议需要 UDP 时才开放 UDP。
- 云安全组与系统防火墙同时检查，避免规则不一致。
- 启用双因素认证。
- 按需配置 fail2ban / IP Limit。
- 不在日志、Issue、截图中泄露订阅链接、Token、Cookie、证书私钥。
- 定期备份 `/etc/x-ui/x-ui.db` 和证书目录。
- 更新前先备份，更新后检查 `x-ui status`、Xray 状态和客户端连通性。

---

## 14. 最小上线验证步骤

完成部署后，至少做一次完整验证：

1. `x-ui status` 显示面板运行中。
2. `systemctl status x-ui --no-pager` 无连续重启。
3. `ss -tulpen` 能看到面板端口、订阅端口和 Xray 入站端口。
4. 浏览器访问 `https://<域名>/<webBasePath>` 可以登录。
5. 登录后立即确认用户名、双因素认证、证书和订阅设置。
6. 创建一个测试入站和客户端。
7. 客户端导入配置后可以连通。
8. 面板首页能看到 Xray Running 和流量变化。
9. 备份 `/etc/x-ui/x-ui.db` 并记录恢复步骤。

做到这里，服务器部署就算稳稳落地了。后续变更端口、证书、反向代理或入站协议时，按“先备份、再改动、后验证”的顺序执行。
