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

**SuperXray** — لوحة تحكم متقدمة مفتوحة المصدر تعتمد على الويب مصممة لإدارة خادم Xray-core. توفر واجهة سهلة الاستخدام لتكوين ومراقبة بروتوكولات VPN والوكيل المختلفة.

> [!IMPORTANT]
> هذا المشروع مخصص للاستخدام الشخصي والاتصال فقط، يرجى عدم استخدامه لأغراض غير قانونية، يرجى عدم استخدامه في بيئة الإنتاج.

كمشروع محسن من مشروع X-UI الأصلي، يوفر SuperXray استقرارًا محسنًا ودعمًا أوسع للبروتوكولات وميزات إضافية.

## البدء السريع

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh)
```

لتثبيت الإصدار الحالي صراحةً:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/superaddmin/SuperXray-gui/main/install.sh) v3.0.4
```

تُنشر الحزم الرسمية حاليًا لنظام Linux بمعماريتي `amd64` و `arm64`. يعرض المثبّت في النهاية اسم المستخدم وكلمة المرور ومنفذ اللوحة و `webBasePath` التي تم إنشاؤها؛ احفظ هذه القيم. تتوفر صورة Docker على `ghcr.io/superaddmin/superxray-gui:3.0.4`. راجع [docs/deployment.md](docs/deployment.md) لتفاصيل Docker والنشر الثنائي ومتطلبات البيئة.

للحصول على الوثائق الكاملة، يرجى زيارة [ويكي المشروع](https://github.com/superaddmin/SuperXray-gui/wiki).

## شكر خاص إلى

- [alireza0](https://github.com/alireza0/)

## الاعتراف

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (الترخيص: **GPL-3.0**): _قواعد توجيه v2ray/xray و v2ray/xray-clients المحسنة مع النطاقات الإيرانية المدمجة وتركيز على الأمان وحظر الإعلانات._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (الترخيص: **GPL-3.0**): _يحتوي هذا المستودع على قواعد توجيه V2Ray محدثة تلقائيًا بناءً على بيانات النطاقات والعناوين المحظورة في روسيا._

## دعم المشروع

**إذا كان هذا المشروع مفيدًا لك، فقد ترغب في إعطائه**:star2:

<a href="https://www.buymeacoffee.com/MHSanaei" target="_blank">
<img src="./media/default-yellow.png" alt="Buy Me A Coffee" style="height: 70px !important;width: 277px !important;" >
</a>
</br>
<a href="https://nowpayments.io/donation/hsanaei" target="_blank" rel="noreferrer noopener">
   <img src="./media/donation-button-black.svg" alt="Crypto donation button by NOWPayments">
</a>

## النجوم عبر الزمن

[![Stargazers over time](https://starchart.cc/superaddmin/SuperXray-gui.svg?variant=adaptive)](https://starchart.cc/superaddmin/SuperXray-gui)
