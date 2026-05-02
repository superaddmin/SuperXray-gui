[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/superxray.svg">
    <img alt="SuperXray" src="./media/superxray.svg">
  </picture>
</p>

[![Release](https://img.shields.io/github/v/release/superaddmin/SuperXray-gui.svg)](https://github.com/superaddmin/SuperXray-gui/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/superaddmin/SuperXray-gui/release.yml.svg)](https://github.com/superaddmin/SuperXray-gui/actions)
[![GO Version](https://img.shields.io/github/go-mod/go-version/superaddmin/SuperXray-gui.svg)](#)
[![Downloads](https://img.shields.io/github/downloads/superaddmin/SuperXray-gui/total.svg)](https://github.com/superaddmin/SuperXray-gui/releases/latest)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)
[![Go Reference](https://pkg.go.dev/badge/github.com/superaddmin/SuperXray-gui/v2.svg)](https://pkg.go.dev/github.com/superaddmin/SuperXray-gui/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/superaddmin/SuperXray-gui/v2)](https://goreportcard.com/report/github.com/superaddmin/SuperXray-gui/v2)

**SuperXray** — advanced, open-source web-based control panel designed for managing Xray-core server. It offers a user-friendly interface for configuring and monitoring various VPN and proxy protocols.

> [!IMPORTANT]
> This project is only for personal usage, please do not use it for illegal purposes, and please do not use it in a production environment.

As an enhanced fork of the original X-UI project, SuperXray provides improved stability, broader protocol support, and additional features.

## Quick Start

### One-click install (Linux)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh)
```

To pin the current release:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh) v3.0.3
```

The installer prepares the required Linux packages (`cron`/`cronie`, `curl`, `tar`, `tzdata`, `socat`, `ca-certificates`, `openssl`), downloads the matching GitHub Release asset, and prints the generated username, password, panel port, and `webBasePath` at the end. Official binary assets are currently published for Linux `amd64` and `arm64`.

### Docker / GHCR

```bash
docker run -d --name superxray-gui --network host --restart unless-stopped \
  -v $PWD/db:/etc/x-ui \
  -v $PWD/cert:/root/cert \
  -e XRAY_VMESS_AEAD_FORCED=false \
  -e XUI_ENABLE_FAIL2BAN=true \
  ghcr.io/superaddmin/superxray-gui:3.0.3
```

The repository `docker-compose.yml` builds the image from local source. For image-based container deployment, you can switch Compose to `image: ghcr.io/superaddmin/superxray-gui:3.0.3`. Docker startup does not run the one-click installer's random security initialization, so change the default credentials, panel port, and `webBasePath` immediately after the first start.

For deployment details, see [docs/deployment.md](docs/deployment.md). For development and release workflow notes, see [docs/development.md](docs/development.md). The project Wiki remains available at <https://github.com/superaddmin/SuperXray-gui/wiki>.

## A Special Thanks to

- [alireza0](https://github.com/alireza0/)

## Acknowledgment

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (License: **GPL-3.0**): _Enhanced v2ray/xray and v2ray/xray-clients routing rules with built-in Iranian domains and a focus on security and adblocking._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (License: **GPL-3.0**): _This repository contains automatically updated V2Ray routing rules based on data on blocked domains and addresses in Russia._

## Support project

**If this project is helpful to you, you may wish to give it a**:star2:

<a href="https://www.buymeacoffee.com/MHSanaei" target="_blank">
<img src="./media/default-yellow.png" alt="Buy Me A Coffee" style="height: 70px !important;width: 277px !important;" >
</a>

</br>
<a href="https://nowpayments.io/donation/hsanaei" target="_blank" rel="noreferrer noopener">
   <img src="./media/donation-button-black.svg" alt="Crypto donation button by NOWPayments">
</a>

## Stargazers over Time

[![Stargazers over time](https://starchart.cc/superaddmin/SuperXray-gui.svg?variant=adaptive)](https://starchart.cc/superaddmin/SuperXray-gui)
