#!/bin/sh
set -eu

# Bound download retries so transient CDN failures cannot hang the build.
download() {
    curl -fsSL --retry 5 --retry-delay 3 --retry-max-time 600 \
        --connect-timeout 20 --max-time 300 "$@"
}

case ${1:-amd64} in
    amd64)
        ARCH="64"
        FNAME="amd64"
        ;;
    i386)
        ARCH="32"
        FNAME="i386"
        ;;
    armv8 | arm64 | aarch64)
        ARCH="arm64-v8a"
        FNAME="arm64"
        ;;
    armv7 | arm | arm32)
        ARCH="arm32-v7a"
        FNAME="arm32"
        ;;
    armv6)
        ARCH="arm32-v6"
        FNAME="armv6"
        ;;
    *)
        ARCH="64"
        FNAME="amd64"
        ;;
esac
mkdir -p build/bin
cd build/bin
download -O "https://github.com/XTLS/Xray-core/releases/download/v26.10.21/Xray-linux-${ARCH}.zip"
download -O "https://github.com/XTLS/Xray-core/releases/download/v26.10.21/Xray-linux-${ARCH}.zip.dgst"
expected_sha=$(awk '/^SHA2-256=/{print $2}' "Xray-linux-${ARCH}.zip.dgst")
test "${#expected_sha}" -eq 64
printf '%s  %s\n' "$expected_sha" "Xray-linux-${ARCH}.zip" | sha256sum -c -
unzip "Xray-linux-${ARCH}.zip"
rm -f "Xray-linux-${ARCH}.zip" "Xray-linux-${ARCH}.zip.dgst" geoip.dat geosite.dat
mv xray "xray-linux-${FNAME}"
download -o geoip.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
download -o geosite.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat
download -o geoip_IR.dat https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geoip.dat
download -o geosite_IR.dat https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geosite.dat
download -o geoip_RU.dat https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat
download -o geosite_RU.dat https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat
cd ../../
