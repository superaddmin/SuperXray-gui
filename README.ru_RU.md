[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/SuperXray.png">
    <img alt="SuperXray" src="./media/SuperXray.png">
  </picture>
</p>

[![Release](https://img.shields.io/github/v/release/superaddmin/SuperXray-gui.svg)](https://github.com/superaddmin/SuperXray-gui/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/superaddmin/SuperXray-gui/release.yml.svg)](https://github.com/superaddmin/SuperXray-gui/actions)
[![GO Version](https://img.shields.io/github/go-mod/go-version/superaddmin/SuperXray-gui.svg)](#)
[![Downloads](https://img.shields.io/github/downloads/superaddmin/SuperXray-gui/total.svg)](https://github.com/superaddmin/SuperXray-gui/releases/latest)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)
[![Go Reference](https://pkg.go.dev/badge/github.com/superaddmin/SuperXray-gui/v2.svg)](https://pkg.go.dev/github.com/superaddmin/SuperXray-gui/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/superaddmin/SuperXray-gui/v2)](https://goreportcard.com/report/github.com/superaddmin/SuperXray-gui/v2)

**SuperXray** — продвинутая панель управления с открытым исходным кодом на основе веб-интерфейса, разработанная для управления сервером Xray-core. Предоставляет удобный интерфейс для настройки и мониторинга различных VPN и прокси-протоколов.

> [!IMPORTANT]
> Этот проект предназначен только для личного использования, пожалуйста, не используйте его в незаконных целях и в производственной среде.

Как улучшенная версия оригинального проекта X-UI, SuperXray обеспечивает повышенную стабильность, более широкую поддержку протоколов и дополнительные функции.

## Быстрый старт

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh)
```

Чтобы установить текущий релиз явно:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh) v3.0.1
```

Официальные бинарные релизы сейчас публикуются для Linux `amd64` и `arm64`. В конце установки скрипт выводит сгенерированные имя пользователя, пароль, порт панели и `webBasePath`; сохраните эти данные. Docker-образ доступен как `ghcr.io/superaddmin/superxray-gui:3.0.1`. Подробности по Docker, бинарному развертыванию и зависимостям см. в [docs/deployment.md](docs/deployment.md).

Полную документацию смотрите в [вики проекта](https://github.com/superaddmin/SuperXray-gui/wiki).

## Особая благодарность

- [alireza0](https://github.com/alireza0/)

## Благодарности

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (Лицензия: **GPL-3.0**): _Улучшенные правила маршрутизации для v2ray/xray и v2ray/xray-clients со встроенными иранскими доменами и фокусом на безопасность и блокировку рекламы._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (Лицензия: **GPL-3.0**): _Этот репозиторий содержит автоматически обновляемые правила маршрутизации V2Ray на основе данных о заблокированных доменах и адресах в России._

## Поддержка проекта

**Если этот проект полезен для вас, вы можете поставить ему**:star2:

<a href="https://www.buymeacoffee.com/MHSanaei" target="_blank">
<img src="./media/default-yellow.png" alt="Buy Me A Coffee" style="height: 70px !important;width: 277px !important;" >
</a>

</br>
<a href="https://nowpayments.io/donation/hsanaei" target="_blank" rel="noreferrer noopener">
   <img src="./media/donation-button-black.svg" alt="Crypto donation button by NOWPayments">
</a>

## Звезды с течением времени

[![Stargazers over time](https://starchart.cc/superaddmin/SuperXray-gui.svg?variant=adaptive)](https://starchart.cc/superaddmin/SuperXray-gui)
