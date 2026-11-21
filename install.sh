#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

xui_folder="${XUI_MAIN_FOLDER:=/usr/local/x-ui}"
xui_service="${XUI_SERVICE:=/etc/systemd/system}"
script_repo="${XUI_SCRIPT_REPO:=superaddmin/SuperXray-gui}"
release_repo="${XUI_RELEASE_REPO:=superaddmin/SuperXray-gui}"
raw_base="https://raw.githubusercontent.com/${script_repo}/main"
release_base="https://github.com/${release_repo}"
release_api="https://api.github.com/repos/${release_repo}"
RELEASE_TAG=""
RELEASE_TAG_SOURCE=""

# check root
[[ $EUID -ne 0 ]] && echo -e "${red}致命错误：${plain}请使用 root 权限运行此脚本 \n " && exit 1

# Check OS and set release variable
if [[ -f /etc/os-release ]]; then
    # The target system supplies its release metadata at runtime.
    # shellcheck source=/dev/null
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    # shellcheck source=/dev/null
    source /usr/lib/os-release
    release=$ID
else
    echo "无法检测系统发行版，请联系维护者！" >&2
    exit 1
fi
echo "当前系统发行版：$release"

arch() {
    case "$(uname -m)" in
        x86_64 | x64 | amd64) echo 'amd64' ;;
        armv8* | arm64 | aarch64) echo 'arm64' ;;
        *) echo -e "${red}不支持的 CPU 架构：$(uname -m)。SuperXray 服务器安装包仅发布 amd64 和 arm64。${plain}" >&2; return 1 ;;
    esac
}

package_arch=$(arch) || exit 1
echo "CPU 架构：$package_arch"

# Simple helpers
is_ipv4() {
    [[ "$1" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] && return 0 || return 1
}
is_ipv6() {
    [[ "$1" =~ : ]] && return 0 || return 1
}
is_domain() {
    [[ "$1" =~ ^([A-Za-z0-9](-*[A-Za-z0-9])*\.)+(xn--[a-z0-9]{2,}|[A-Za-z]{2,})$ ]] && return 0 || return 1
}

extract_release_tag() {
    grep -m1 '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

fetch_latest_release_tag() {
    local ip_opt="$1"
    local curl_args=(-fsSLH "Accept: application/vnd.github+json")
    [[ -n "${ip_opt}" ]] && curl_args=("${ip_opt}" "${curl_args[@]}")
    curl "${curl_args[@]}" "${release_api}/releases/latest" 2>/dev/null | extract_release_tag
}

fetch_first_release_tag() {
    local ip_opt="$1"
    local curl_args=(-fsSLH "Accept: application/vnd.github+json")
    [[ -n "${ip_opt}" ]] && curl_args=("${ip_opt}" "${curl_args[@]}")
    curl "${curl_args[@]}" "${release_api}/releases?per_page=10" 2>/dev/null | extract_release_tag
}

resolve_release_tag() {
    local tag=""

    tag=$(fetch_latest_release_tag "")
    if [[ -n "${tag}" ]]; then
        RELEASE_TAG="${tag}"
        RELEASE_TAG_SOURCE="latest"
        return 0
    fi

    echo -e "${yellow}未找到 GitHub latest 正式版，正在尝试获取最新公开 Release...${plain}" >&2
    tag=$(fetch_first_release_tag "")
    if [[ -n "${tag}" ]]; then
        RELEASE_TAG="${tag}"
        RELEASE_TAG_SOURCE="release-list"
        return 0
    fi

    echo -e "${yellow}正在尝试通过 IPv4 获取版本信息...${plain}" >&2
    tag=$(fetch_latest_release_tag "-4")
    if [[ -n "${tag}" ]]; then
        RELEASE_TAG="${tag}"
        RELEASE_TAG_SOURCE="latest-ipv4"
        return 0
    fi

    tag=$(fetch_first_release_tag "-4")
    if [[ -n "${tag}" ]]; then
        RELEASE_TAG="${tag}"
        RELEASE_TAG_SOURCE="release-list-ipv4"
        return 0
    fi

    return 1
}

# Port helpers
is_port_in_use() {
    local port="$1"
    if command -v ss >/dev/null 2>&1; then
        ss -ltn 2>/dev/null | awk -v p=":${port}$" '$4 ~ p {found = 1; exit} END {exit !found}'
        return
    fi
    if command -v netstat >/dev/null 2>&1; then
        netstat -lnt 2>/dev/null | awk -v p=":${port}$" '$4 ~ p {found = 1; exit} END {exit !found}'
        return
    fi
    if command -v lsof >/dev/null 2>&1; then
        lsof -nP "-iTCP:$port" -sTCP:LISTEN >/dev/null 2>&1 && return 0
    fi
    return 1
}

install_base() {
    case "${release}" in
        ubuntu | debian | armbian)
            apt-get update && apt-get install -y -q cron curl tar tzdata socat ca-certificates openssl
        ;;
        fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
            dnf -y update && dnf install -y -q cronie curl tar tzdata socat ca-certificates openssl
        ;;
        centos)
            if [[ "${VERSION_ID}" =~ ^7 ]]; then
                yum -y update && yum install -y cronie curl tar tzdata socat ca-certificates openssl
            else
                dnf -y update && dnf install -y -q cronie curl tar tzdata socat ca-certificates openssl
            fi
        ;;
        arch | manjaro | parch)
            pacman -Syu && pacman -Syu --noconfirm cronie curl tar tzdata socat ca-certificates openssl
        ;;
        opensuse-tumbleweed | opensuse-leap)
            zypper refresh && zypper -q install -y cron curl tar timezone socat ca-certificates openssl
        ;;
        alpine)
            apk update && apk add dcron curl tar tzdata socat ca-certificates openssl
        ;;
        *)
            apt-get update && apt-get install -y -q cron curl tar tzdata socat ca-certificates openssl
        ;;
    esac
}

gen_random_string() {
    local length="$1"
    openssl rand -base64 $(( length * 2 )) \
        | tr -dc 'a-zA-Z0-9' \
        | head -c "$length"
}

install_acme() {
    echo -e "${green}正在安装 acme.sh，用于管理 SSL 证书...${plain}"
    cd ~ || return 1
    if ! curl -s https://get.acme.sh | sh >/dev/null 2>&1; then
        echo -e "${red}acme.sh 安装失败${plain}"
        return 1
    else
        echo -e "${green}acme.sh 安装成功${plain}"
    fi
    return 0
}


# Issue Let's Encrypt IP certificate with shortlived profile (~6 days validity)
# Requires acme.sh and port 80 open for HTTP-01 challenge
setup_ip_certificate() {
    local ipv4="$1"
    local ipv6="$2"  # optional

    echo -e "${green}正在配置 Let's Encrypt IP 证书（短周期证书）...${plain}"
    echo -e "${yellow}说明：IP 证书有效期约 6 天，并会自动续期。${plain}"
    echo -e "${yellow}默认监听 80 端口；如果使用其他端口，请确保外部 80 端口已转发到该端口。${plain}"

    # Check for acme.sh
    if ! command -v ~/.acme.sh/acme.sh &>/dev/null; then
        if ! install_acme; then
            echo -e "${red}acme.sh 安装失败${plain}"
            return 1
        fi
    fi

    # Validate IP address
    if [[ -z "$ipv4" ]]; then
        echo -e "${red}必须提供 IPv4 地址${plain}"
        return 1
    fi

    if ! is_ipv4 "$ipv4"; then
        echo -e "${red}IPv4 地址无效：$ipv4${plain}"
        return 1
    fi

    # Create certificate directory
    local certDir="/root/cert/ip"
    mkdir -p "$certDir"

    # Build domain arguments
    local domain_args=(-d "$ipv4")
    if [[ -n "$ipv6" ]] && is_ipv6 "$ipv6"; then
        domain_args+=(-d "$ipv6")
        echo -e "${green}已包含 IPv6 地址：${ipv6}${plain}"
    fi

    # Set reload command for auto-renewal (add || true so it doesn't fail during first install)
    local reloadCmd="systemctl restart x-ui 2>/dev/null || rc-service x-ui restart 2>/dev/null || true"

    # Choose port for HTTP-01 listener (default 80, prompt override)
    local WebPort=""
    read -rp "请输入 ACME HTTP-01 监听端口（默认 80）：" WebPort
    WebPort="${WebPort:-80}"
    if ! [[ "${WebPort}" =~ ^[0-9]+$ ]] || ((WebPort < 1 || WebPort > 65535)); then
        echo -e "${red}端口无效，将使用默认端口 80。${plain}"
        WebPort=80
    fi
    echo -e "${green}将使用端口 ${WebPort} 进行 standalone 校验。${plain}"
    if [[ "${WebPort}" -ne 80 ]]; then
        echo -e "${yellow}提醒：Let's Encrypt 仍会访问外部 80 端口，请将外部 80 端口转发到 ${WebPort}。${plain}"
    fi

    # Ensure chosen port is available
    while true; do
        if is_port_in_use "${WebPort}"; then
            echo -e "${yellow}端口 ${WebPort} 已被占用。${plain}"

            local alt_port=""
            read -rp "请输入 acme.sh standalone 监听的其他端口（留空则中止）：" alt_port
            alt_port="${alt_port// /}"
            if [[ -z "${alt_port}" ]]; then
                echo -e "${red}端口 ${WebPort} 被占用，无法继续。${plain}"
                return 1
            fi
            if ! [[ "${alt_port}" =~ ^[0-9]+$ ]] || ((alt_port < 1 || alt_port > 65535)); then
                echo -e "${red}端口无效。${plain}"
                return 1
            fi
            WebPort="${alt_port}"
            continue
        else
            echo -e "${green}端口 ${WebPort} 可用，可用于 standalone 校验。${plain}"
            break
        fi
    done

    # Issue certificate with shortlived profile
    echo -e "${green}正在为 ${ipv4} 签发 IP 证书...${plain}"
    ~/.acme.sh/acme.sh --set-default-ca --server letsencrypt --force >/dev/null 2>&1
    
    if ! ~/.acme.sh/acme.sh --issue \
        "${domain_args[@]}" \
        --standalone \
        --server letsencrypt \
        --certificate-profile shortlived \
        --days 6 \
        --httpport "${WebPort}" \
        --force; then
        echo -e "${red}IP 证书签发失败${plain}"
        echo -e "${yellow}请确认端口 ${WebPort} 可访问（或已从外部 80 端口转发）${plain}"
        # Cleanup acme.sh data for both IPv4 and IPv6 if specified
        rm -rf "$HOME/.acme.sh/${ipv4}" 2>/dev/null
        [[ -n "$ipv6" ]] && rm -rf "$HOME/.acme.sh/${ipv6}" 2>/dev/null
        rm -rf "$certDir" 2>/dev/null
        return 1
    fi

    echo -e "${green}证书签发成功，正在安装...${plain}"

    # Install certificate
    # Note: acme.sh may report "Reload error" and exit non-zero if reloadcmd fails,
    # but the cert files are still installed. We check for files instead of exit code.
    ~/.acme.sh/acme.sh --installcert -d "${ipv4}" \
        --key-file "${certDir}/privkey.pem" \
        --fullchain-file "${certDir}/fullchain.pem" \
        --reloadcmd "${reloadCmd}" 2>&1 || true

    # Verify certificate files exist (don't rely on exit code - reloadcmd failure causes non-zero)
    if [[ ! -f "${certDir}/fullchain.pem" || ! -f "${certDir}/privkey.pem" ]]; then
        echo -e "${red}安装后未找到证书文件${plain}"
        # Cleanup acme.sh data for both IPv4 and IPv6 if specified
        rm -rf "$HOME/.acme.sh/${ipv4}" 2>/dev/null
        [[ -n "$ipv6" ]] && rm -rf "$HOME/.acme.sh/${ipv6}" 2>/dev/null
        rm -rf "$certDir" 2>/dev/null
        return 1
    fi
    
    echo -e "${green}证书文件安装成功${plain}"

    # Enable auto-upgrade for acme.sh (ensures cron job runs)
    ~/.acme.sh/acme.sh --upgrade --auto-upgrade >/dev/null 2>&1

    # Secure permissions: private key readable only by owner
    chmod 600 "$certDir/privkey.pem" 2>/dev/null
    chmod 644 "$certDir/fullchain.pem" 2>/dev/null

    # Configure panel to use the certificate
    echo -e "${green}正在为面板设置证书路径...${plain}"
    if ! "${xui_folder}/x-ui" cert -webCert "${certDir}/fullchain.pem" -webCertKey "${certDir}/privkey.pem"; then
        echo -e "${yellow}警告：无法自动设置证书路径${plain}"
        echo -e "${yellow}证书文件位置：${plain}"
        echo -e "  证书：${certDir}/fullchain.pem"
        echo -e "  私钥：${certDir}/privkey.pem"
    else
        echo -e "${green}证书路径配置成功${plain}"
    fi

    echo -e "${green}IP 证书已安装并配置成功！${plain}"
    echo -e "${green}证书有效期约 6 天，会通过 acme.sh 定时任务自动续期。${plain}"
    echo -e "${yellow}acme.sh 会在证书到期前自动续期并重载 x-ui。${plain}"
    return 0
}

# Comprehensive manual SSL certificate issuance via acme.sh
ssl_cert_issue() {
    local existing_port=$1 existing_webBasePath=$2
    
    # check for acme.sh first
    if ! command -v ~/.acme.sh/acme.sh &>/dev/null; then
        echo "未找到 acme.sh，正在安装..."
        cd ~ || return 1
        if ! curl -s https://get.acme.sh | sh; then
            echo -e "${red}acme.sh 安装失败${plain}"
            return 1
        else
            echo -e "${green}acme.sh 安装成功${plain}"
        fi
    fi

    # get the domain here, and we need to verify it
    local domain=""
    while true; do
        read -rp "请输入域名：" domain
        domain="${domain// /}"  # Trim whitespace
        
        if [[ -z "$domain" ]]; then
            echo -e "${red}域名不能为空，请重试。${plain}"
            continue
        fi
        
        if ! is_domain "$domain"; then
            echo -e "${red}域名格式无效：${domain}。请输入有效域名。${plain}"
            continue
        fi
        
        break
    done
    echo -e "${green}当前域名：${domain}，正在检查...${plain}"
    SSL_ISSUED_DOMAIN="${domain}"

    # detect existing certificate and reuse it if present
    local cert_exists=0
    if ~/.acme.sh/acme.sh --list 2>/dev/null | awk '{print $1}' | grep -Fxq "${domain}"; then
        cert_exists=1
        local certInfo
        certInfo=$(~/.acme.sh/acme.sh --list 2>/dev/null | grep -F "${domain}")
        echo -e "${yellow}已找到 ${domain} 的现有证书，将直接复用。${plain}"
        [[ -n "${certInfo}" ]] && echo "$certInfo"
    else
        echo -e "${green}域名检查通过，可以开始签发证书...${plain}"
    fi

    # create a directory for the certificate
    certPath="/root/cert/${domain}"
    if [ ! -d "$certPath" ]; then
        mkdir -p "$certPath"
    else
        rm -rf "$certPath"
        mkdir -p "$certPath"
    fi

    # get the port number for the standalone server
    local WebPort=80
    read -rp "请选择用于签发证书的端口（默认 80）：" WebPort
    if [[ ${WebPort} -gt 65535 || ${WebPort} -lt 1 ]]; then
        echo -e "${yellow}输入的端口 ${WebPort} 无效，将使用默认端口 80。${plain}"
        WebPort=80
    fi
    echo -e "${green}将使用端口 ${WebPort} 签发证书，请确认该端口已开放。${plain}"

    # Stop panel temporarily
    echo -e "${yellow}正在临时停止面板...${plain}"
    systemctl stop x-ui 2>/dev/null || rc-service x-ui stop 2>/dev/null

    if [[ ${cert_exists} -eq 0 ]]; then
        # issue the certificate
        ~/.acme.sh/acme.sh --set-default-ca --server letsencrypt --force
        if ! ~/.acme.sh/acme.sh --issue -d "${domain}" --listen-v6 --standalone --httpport "${WebPort}" --force; then
            echo -e "${red}证书签发失败，请检查日志。${plain}"
            rm -rf "$HOME/.acme.sh/${domain}"
            systemctl start x-ui 2>/dev/null || rc-service x-ui start 2>/dev/null
            return 1
        else
            echo -e "${green}证书签发成功，正在安装证书...${plain}"
        fi
    else
        echo -e "${green}正在使用现有证书并安装到面板...${plain}"
    fi

    # Setup reload command
    reloadCmd="systemctl restart x-ui || rc-service x-ui restart"
    echo -e "${green}ACME 默认 --reloadcmd 为：${yellow}systemctl restart x-ui || rc-service x-ui restart${plain}"
    echo -e "${green}该命令会在每次证书签发和续期时执行。${plain}"
    read -rp "是否要修改 ACME 的 --reloadcmd？(y/n): " setReloadcmd
    if [[ "$setReloadcmd" == "y" || "$setReloadcmd" == "Y" ]]; then
        echo -e "\n${green}\t1.${plain} 预设：systemctl reload nginx ; systemctl restart x-ui"
        echo -e "${green}\t2.${plain} 输入自定义命令"
        echo -e "${green}\t0.${plain} 保持默认 reloadcmd"
        read -rp "请选择一个选项：" choice
        case "$choice" in
        1)
            echo -e "${green}Reloadcmd 为：systemctl reload nginx ; systemctl restart x-ui${plain}"
            reloadCmd="systemctl reload nginx ; systemctl restart x-ui"
            ;;
        2)
            echo -e "${yellow}建议将 x-ui restart 放在命令末尾${plain}"
            read -rp "请输入自定义 reloadcmd：" reloadCmd
            echo -e "${green}Reloadcmd 为：${reloadCmd}${plain}"
            ;;
        *)
            echo -e "${green}保持默认 reloadcmd${plain}"
            ;;
        esac
    fi

    # install the certificate
    local installOutput=""
    installOutput=$(~/.acme.sh/acme.sh --installcert -d "${domain}" \
        --key-file "/root/cert/${domain}/privkey.pem" \
        --fullchain-file "/root/cert/${domain}/fullchain.pem" --reloadcmd "${reloadCmd}" 2>&1)
    local installRc=$?
    echo "${installOutput}"

    local installWroteFiles=0
    if echo "${installOutput}" | grep -q "Installing key to:" && echo "${installOutput}" | grep -q "Installing full chain to:"; then
        installWroteFiles=1
    fi

    if [[ -f "/root/cert/${domain}/privkey.pem" && -f "/root/cert/${domain}/fullchain.pem" && ( ${installRc} -eq 0 || ${installWroteFiles} -eq 1 ) ]]; then
        echo -e "${green}证书安装成功，正在启用自动续期...${plain}"
    else
        echo -e "${red}证书安装失败，正在退出。${plain}"
        if [[ ${cert_exists} -eq 0 ]]; then
            rm -rf "$HOME/.acme.sh/${domain}"
        fi
        systemctl start x-ui 2>/dev/null || rc-service x-ui start 2>/dev/null
        return 1
    fi

    # enable auto-renew
    if ! ~/.acme.sh/acme.sh --upgrade --auto-upgrade; then
        echo -e "${yellow}自动续期配置可能存在问题，证书详情如下：${plain}"
        ls -lah "/root/cert/${domain}/"
        # Secure permissions: private key readable only by owner
        chmod 600 "$certPath/privkey.pem" 2>/dev/null
        chmod 644 "$certPath/fullchain.pem" 2>/dev/null
    else
        echo -e "${green}自动续期已启用，证书详情如下：${plain}"
        ls -lah "/root/cert/${domain}/"
        # Secure permissions: private key readable only by owner
        chmod 600 "$certPath/privkey.pem" 2>/dev/null
        chmod 644 "$certPath/fullchain.pem" 2>/dev/null
    fi

    # start panel
    systemctl start x-ui 2>/dev/null || rc-service x-ui start 2>/dev/null

    # Prompt user to set panel paths after successful certificate installation
    read -rp "是否将此证书设置到面板？(y/n): " setPanel
    if [[ "$setPanel" == "y" || "$setPanel" == "Y" ]]; then
        local webCertFile="/root/cert/${domain}/fullchain.pem"
        local webKeyFile="/root/cert/${domain}/privkey.pem"

        if [[ -f "$webCertFile" && -f "$webKeyFile" ]]; then
            "${xui_folder}/x-ui" cert -webCert "$webCertFile" -webCertKey "$webKeyFile"
            echo -e "${green}已为面板设置证书路径${plain}"
            echo -e "${green}证书文件：$webCertFile${plain}"
            echo -e "${green}私钥文件：$webKeyFile${plain}"
            echo ""
            echo -e "${green}访问地址：https://${domain}:${existing_port}/${existing_webBasePath}${plain}"
            echo -e "${yellow}面板将重启以应用 SSL 证书...${plain}"
            systemctl restart x-ui 2>/dev/null || rc-service x-ui restart 2>/dev/null
        else
            echo -e "${red}错误：未找到域名 $domain 对应的证书或私钥文件。${plain}"
        fi
    else
        echo -e "${yellow}已跳过面板证书路径设置。${plain}"
    fi
    
    return 0
}

# Reusable interactive SSL setup (domain or IP)
# Sets global `SSL_HOST` to the chosen domain/IP for Access URL usage
prompt_and_setup_ssl() {
    local server_ip="$1"

    local ssl_choice=""

    echo -e "${yellow}请选择 SSL 证书配置方式：${plain}"
    echo -e "${green}1.${plain} Let's Encrypt 域名证书（有效期 90 天，自动续期）"
    echo -e "${green}2.${plain} Let's Encrypt IP 证书（有效期约 6 天，自动续期）"
    echo -e "${green}3.${plain} 自定义 SSL 证书（使用已有证书文件路径）"
    echo -e "${blue}说明：${plain}选项 1 和 2 需要开放 80 端口；选项 3 需要手动输入证书路径。"
    read -rp "请选择一个选项（默认 2，使用 IP 证书）：" ssl_choice
    ssl_choice="${ssl_choice// /}"  # Trim whitespace
    
    # Default to 2 (IP cert) if input is empty or invalid (not 1 or 3)
    if [[ "$ssl_choice" != "1" && "$ssl_choice" != "3" ]]; then
        ssl_choice="2"
    fi

    case "$ssl_choice" in
    1)
        # User chose Let's Encrypt domain option
        echo -e "${green}正在使用 Let's Encrypt 申请域名证书...${plain}"
        if ssl_cert_issue "$2" "$3"; then
            local cert_domain="${SSL_ISSUED_DOMAIN}"
            if [[ -z "${cert_domain}" ]]; then
                cert_domain=$(~/.acme.sh/acme.sh --list 2>/dev/null | tail -1 | awk '{print $1}')
            fi

            if [[ -n "${cert_domain}" ]]; then
                SSL_HOST="${cert_domain}"
                echo -e "${green}[成功] SSL 证书已使用域名 ${cert_domain} 配置完成${plain}"
            else
                echo -e "${yellow}SSL 可能已配置完成，但未能自动识别域名${plain}"
                SSL_HOST="${server_ip}"
            fi
        else
            echo -e "${red}域名模式 SSL 证书配置失败。${plain}"
            SSL_HOST="${server_ip}"
        fi
        ;;
    2)
        # User chose Let's Encrypt IP certificate option
        echo -e "${green}正在使用 Let's Encrypt 申请 IP 证书（短周期证书）...${plain}"
        
        # Ask for optional IPv6
        local ipv6_addr=""
        read -rp "是否需要同时包含 IPv6 地址？（留空则跳过）：" ipv6_addr
        ipv6_addr="${ipv6_addr// /}"  # Trim whitespace
        
        # Stop panel if running (port 80 needed)
        if [[ $release == "alpine" ]]; then
            rc-service x-ui stop >/dev/null 2>&1
        else
            systemctl stop x-ui >/dev/null 2>&1
        fi
        
        if setup_ip_certificate "${server_ip}" "${ipv6_addr}"; then
            SSL_HOST="${server_ip}"
            echo -e "${green}[成功] Let's Encrypt IP 证书配置完成${plain}"
        else
            echo -e "${red}[失败] IP 证书配置失败，请检查 80 端口是否开放。${plain}"
            SSL_HOST="${server_ip}"
        fi
        ;;
    3)
        # User chose Custom Paths (User Provided) option
        echo -e "${green}正在使用自定义已有证书...${plain}"
        local custom_cert=""
        local custom_key=""
        local custom_domain=""

        # 3.1 Request Domain to compose Panel URL later
        read -rp "请输入证书对应的域名：" custom_domain
        custom_domain="${custom_domain// /}" # Remove spaces

        # 3.2 Loop for Certificate Path
        while true; do
            read -rp "请输入证书文件路径（关键词：.crt / fullchain）：" custom_cert
            # Strip quotes if present
            custom_cert=$(echo "$custom_cert" | tr -d '"' | tr -d "'")

            if [[ -f "$custom_cert" && -r "$custom_cert" && -s "$custom_cert" ]]; then
                break
            elif [[ ! -f "$custom_cert" ]]; then
                echo -e "${red}错误：文件不存在，请重试。${plain}"
            elif [[ ! -r "$custom_cert" ]]; then
                echo -e "${red}错误：文件存在但不可读，请检查权限。${plain}"
            else
                echo -e "${red}错误：文件为空。${plain}"
            fi
        done

        # 3.3 Loop for Private Key Path
        while true; do
            read -rp "请输入私钥文件路径（关键词：.key / privatekey）：" custom_key
            # Strip quotes if present
            custom_key=$(echo "$custom_key" | tr -d '"' | tr -d "'")

            if [[ -f "$custom_key" && -r "$custom_key" && -s "$custom_key" ]]; then
                break
            elif [[ ! -f "$custom_key" ]]; then
                echo -e "${red}错误：文件不存在，请重试。${plain}"
            elif [[ ! -r "$custom_key" ]]; then
                echo -e "${red}错误：文件存在但不可读，请检查权限。${plain}"
            else
                echo -e "${red}错误：文件为空。${plain}"
            fi
        done

        # 3.4 Apply Settings via x-ui binary
        "${xui_folder}/x-ui" cert -webCert "$custom_cert" -webCertKey "$custom_key" >/dev/null 2>&1
        
        # Set SSL_HOST for composing Panel URL
        if [[ -n "$custom_domain" ]]; then
            SSL_HOST="$custom_domain"
        else
            SSL_HOST="${server_ip}"
        fi

        echo -e "${green}[成功] 自定义证书路径已应用。${plain}"
        echo -e "${yellow}说明：请自行负责这些证书文件的续期。${plain}"

        systemctl restart x-ui >/dev/null 2>&1 || rc-service x-ui restart >/dev/null 2>&1
        ;;
    *)
        echo -e "${red}选项无效，已跳过 SSL 配置。${plain}"
        SSL_HOST="${server_ip}"
        ;;
    esac
}

read_panel_settings() {
    local settings line value initialize_path=auto path_found=false cert_found=false
    settings=$("${xui_folder}/x-ui" setting -show -getCert) || return 1
    existing_port=""
    existing_hasDefaultCredential=""
    while IFS= read -r line; do
        value=${line#*:}
        value=${value# }
        case "$line" in
            port:*) existing_port=$value ;;
            hasDefaultCredential:*) existing_hasDefaultCredential=$value ;;
            webBasePath:*) existing_webBasePath=${value#/}; path_found=true ;;
            initializeWebBasePath:*) initialize_path=$value ;;
            cert:*) existing_cert=$value; cert_found=true ;;
            'get '*' failed, error info:'*) return 1 ;;
        esac
    done <<< "$settings"
    [[ $path_found == true && $cert_found == true ]] || return 1
    [[ $existing_hasDefaultCredential == true || $existing_hasDefaultCredential == false ]] || return 1
    [[ $existing_port =~ ^[1-9][0-9]{0,4}$ ]] && (( existing_port <= 65535 )) || return 1
    case "$initialize_path" in
        true) existing_webBasePath="" ;;
        false|auto) ;; # Older releases only report the effective path.
        *) return 1 ;;
    esac
}

config_after_install() {
    local existing_hasDefaultCredential existing_webBasePath existing_port existing_cert
    if ! read_panel_settings; then
        echo '无法读取完整的面板设置，已停止安装。' >&2
        return 1
    fi
    local URL_lists=(
        "https://api4.ipify.org"
        "https://ipv4.icanhazip.com"
        "https://v4.api.ipinfo.io/ip"
        "https://ipv4.myexternalip.com/raw"
        "https://4.ident.me"
        "https://check-host.net/ip"
    )
    local server_ip=""
    for ip_address in "${URL_lists[@]}"; do
        local response http_code ip_result
        response=$(curl -s -w "\n%{http_code}" --max-time 3 "${ip_address}" 2>/dev/null)
        http_code=$(echo "$response" | tail -n1)
        ip_result=$(echo "$response" | head -n-1 | tr -d '[:space:]')
        if [[ "${http_code}" == "200" && -n "${ip_result}" ]]; then
            server_ip="${ip_result}"
            break
        fi
    done
    
    if [[ -z $existing_webBasePath ]]; then
        if [[ "$existing_hasDefaultCredential" == "true" ]]; then
            local config_webBasePath config_username config_password
            config_webBasePath=$(gen_random_string 18)
            config_username=$(gen_random_string 10)
            config_password=$(gen_random_string 10)
            
            read -rp "是否自定义面板端口？（选择 n 将自动生成随机端口）[y/n]: " config_confirm
            if [[ "${config_confirm}" == "y" || "${config_confirm}" == "Y" ]]; then
                read -rp "请设置面板端口：" config_port
                echo -e "${yellow}面板端口：${config_port}${plain}"
            else
                local config_port
                config_port=$(shuf -i 1024-62000 -n 1)
                echo -e "${yellow}已生成随机端口：${config_port}${plain}"
            fi
            
            "${xui_folder}/x-ui" setting -username "${config_username}" -password "${config_password}" -port "${config_port}" -webBasePath "${config_webBasePath}" || return 1
            
            echo ""
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${green}        SSL 证书配置（必须）        ${plain}"
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${yellow}出于安全考虑，面板必须配置 SSL 证书。${plain}"
            echo -e "${yellow}Let's Encrypt 目前支持域名和 IP 地址证书。${plain}"
            echo ""

            prompt_and_setup_ssl "${server_ip}" "${config_port}" "${config_webBasePath}"
            
            # Display final credentials and access information
            echo ""
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${green}          面板安装完成！          ${plain}"
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${green}用户名：      ${config_username}${plain}"
            echo -e "${green}密码：        ${config_password}${plain}"
            echo -e "${green}端口：        ${config_port}${plain}"
            echo -e "${green}WebBasePath: ${config_webBasePath}${plain}"
            echo -e "${green}访问地址：    https://${SSL_HOST}:${config_port}/${config_webBasePath}${plain}"
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${yellow}[重要] 请妥善保存以上登录凭据！${plain}"
            echo -e "${yellow}[SSL] 证书已启用并完成配置${plain}"
        else
            local config_webBasePath
            config_webBasePath=$(gen_random_string 18)
            echo -e "${yellow}WebBasePath 为空或根路径，正在生成新的路径...${plain}"
            "${xui_folder}/x-ui" setting -webBasePath "${config_webBasePath}" || return 1
            echo -e "${green}新的 WebBasePath：${config_webBasePath}${plain}"

            # If the panel is already installed but no certificate is configured, prompt for SSL now
            if [[ -z "${existing_cert}" ]]; then
                echo ""
                echo -e "${green}═══════════════════════════════════════════${plain}"
                echo -e "${green}        SSL 证书配置（推荐）        ${plain}"
                echo -e "${green}═══════════════════════════════════════════${plain}"
                echo -e "${yellow}Let's Encrypt 目前支持域名和 IP 地址证书。${plain}"
                echo ""
                prompt_and_setup_ssl "${server_ip}" "${existing_port}" "${config_webBasePath}"
                echo -e "${green}访问地址：  https://${SSL_HOST}:${existing_port}/${config_webBasePath}${plain}"
            else
                # If a cert already exists, just show the access URL
                echo -e "${green}访问地址：https://${server_ip}:${existing_port}/${config_webBasePath}${plain}"
            fi
        fi
    else
        if [[ "$existing_hasDefaultCredential" == "true" ]]; then
            local config_username config_password
            config_username=$(gen_random_string 10)
            config_password=$(gen_random_string 10)
            
            echo -e "${yellow}检测到默认凭据，需要进行安全更新...${plain}"
            "${xui_folder}/x-ui" setting -username "${config_username}" -password "${config_password}" || return 1
            echo -e "已生成新的随机登录凭据："
            echo -e "###############################################"
            echo -e "${green}用户名：${config_username}${plain}"
            echo -e "${green}密码：${config_password}${plain}"
            echo -e "###############################################"
        else
            echo -e "${green}用户名、密码和 WebBasePath 已正确设置。${plain}"
        fi

        # Existing install: if no cert configured, prompt user for SSL setup
        if [[ -z "$existing_cert" ]]; then
            echo ""
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${green}        SSL 证书配置（推荐）        ${plain}"
            echo -e "${green}═══════════════════════════════════════════${plain}"
            echo -e "${yellow}Let's Encrypt 目前支持域名和 IP 地址证书。${plain}"
            echo ""
            prompt_and_setup_ssl "${server_ip}" "${existing_port}" "${existing_webBasePath}"
            echo -e "${green}访问地址：  https://${SSL_HOST}:${existing_port}/${existing_webBasePath}${plain}"
        else
            echo -e "${green}SSL 证书已配置，无需操作。${plain}"
        fi
    fi
    
    "${xui_folder}/x-ui" migrate
}

create_release_stage() {
    xui_folder=$(readlink -f -- "$xui_folder") || return 1
    [[ -n "$xui_folder" && "$xui_folder" != / ]] || return 1
    if [[ -d "$xui_folder" && ! -f "$xui_folder/x-ui" && -n "$(ls -A -- "$xui_folder")" ]]; then
        echo "Installation directory contains unrelated files: $xui_folder" >&2
        return 1
    fi
    release_stage=$(mktemp -d "${xui_folder}.stage.XXXXXX") || return 1
    trap finish_release EXIT
    release_switched=false
    release_was_running=false
    release_unit="${xui_service}/x-ui.service"
    [[ $release != alpine ]] || release_unit=/etc/init.d/x-ui
    if [[ -e "$release_unit" ]]; then cp -a -- "$release_unit" "$release_stage/old-unit" || return 1; fi
    if [[ -e /usr/bin/x-ui ]]; then cp -a -- /usr/bin/x-ui "$release_stage/old-cli" || return 1; fi
}

extract_release() {
    mkdir -- "$release_stage/unpacked" || return 1
    tar -xzf "$release_stage/release.tar.gz" -C "$release_stage/unpacked" || return 1
    local root="$release_stage/unpacked/x-ui"
    [[ -d "$root" && ! -L "$root" && ! -L "$root/bin" ]] || return 1
    local member
    for member in "$root/x-ui" "$root/bin/xray-linux-$package_arch"; do
        [[ -f "$member" && ! -L "$member" && -s "$member" ]] || return 1
        chmod +x -- "$member" || return 1
    done
}

switch_release() {
    if [[ -e "$xui_folder" ]]; then
        local next="$release_stage/unpacked/x-ui"
        local packaged_generic_unit=false
        [[ ! -f "$next/x-ui.service" ]] || packaged_generic_unit=true
        cp -an -- "$xui_folder/." "$next/" || return 1
        if [[ $packaged_generic_unit == false ]]; then
            rm -f -- "$next/x-ui.service" || return 1
        fi
        mv -T -- "$xui_folder" "$release_stage/previous" || return 1
    fi
    mv -T -- "$release_stage/unpacked/x-ui" "$xui_folder" || return 1
    release_switched=true
}

finish_release() {
    local status=$?
    if (( status != 0 )); then
        if [[ $release_switched == true ]]; then
            if [[ $release == alpine ]]; then rc-service x-ui stop; else systemctl stop x-ui; fi
            mv -T -- "$xui_folder" "$release_stage/failed" || return 1
            local old destination
            for old in old-cli old-unit; do
                if [[ $old == old-cli ]]; then destination=/usr/bin/x-ui; else destination=$release_unit; fi
                if [[ -e "$release_stage/$old" ]]; then
                    cp -a -- "$release_stage/$old" "$destination" || return 1
                else
                    rm -f -- "$destination" || return 1
                fi
            done
            [[ $release == alpine ]] || systemctl daemon-reload
        fi
        if [[ -d "$release_stage/previous" ]]; then
            mv -T -- "$release_stage/previous" "$xui_folder" || return 1
        fi
        if [[ $release_was_running == true ]]; then
            if [[ $release == alpine ]]; then rc-service x-ui start; else systemctl start x-ui; fi
        fi
    fi
    rm -rf -- "${release_stage:?}"
}

configure_unit_path() {
    [[ $xui_folder != /usr/local/x-ui ]] || return 0
    local line dir_value exec_value shell_value
    dir_value=$(printf '%s' "$xui_folder" | sed 's/%/%%/g')
    exec_value=$(printf '%s' "$dir_value" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\$/$$/g')
    shell_value=$(printf '%s' "$xui_folder" | sed "s/'/'\\\\''/g")
    while IFS= read -r line || [[ -n "$line" ]]; do
        case "$release:$line" in
            alpine:command=*) printf "command='%s/x-ui'\n" "$shell_value" ;;
            alpine:*'cd /usr/local/x-ui'*) printf "    cd '%s' || return 1\n" "$shell_value" ;;
            *:WorkingDirectory=*) printf 'WorkingDirectory=%s\n' "$dir_value" ;;
            *:ExecStart=*) printf 'ExecStart="%s/x-ui"\n' "$exec_value" ;;
            *) printf '%s\n' "$line" ;;
        esac
    done < "$release_unit" > "$release_stage/rendered-unit" || return 1
    cat "$release_stage/rendered-unit" > "$release_unit"
}

install_release_menu() {
    local target=/usr/bin/x-ui
    if [[ $xui_folder == /usr/local/x-ui ]]; then
        install -m 755 "$release_stage/menu.sh" "$target"
        return
    fi
    install -m 755 "$release_stage/menu.sh" "$xui_folder/x-ui.sh" || return 1
    {
        printf '#!/bin/bash\nexport XUI_MAIN_FOLDER=%q\n' "$xui_folder"
        printf 'export XUI_MENU_TARGET=%q\n' "$xui_folder/x-ui.sh"
        printf 'exec %q "$@"\n' "$xui_folder/x-ui.sh"
    } > "$release_stage/launcher" || return 1
    install -m 755 "$release_stage/launcher" "$target"
}

stop_release_service() {
    if [[ $release == alpine ]]; then
        if rc-service x-ui status >/dev/null 2>&1; then
            release_was_running=true
            rc-service x-ui stop || return 1
        fi
    elif systemctl is-active --quiet x-ui; then
        release_was_running=true
        systemctl stop x-ui || return 1
    fi
}

install_x-ui() {
    create_release_stage || exit 1
    
    # Download resources
    if [ $# == 0 ]; then
        if ! resolve_release_tag || [[ -z "${RELEASE_TAG}" ]]; then
            echo -e "${red}获取 SuperXray Release 版本失败，GitHub 未返回可用版本。${plain}"
            echo -e "${yellow}请检查 ${release_base}/releases，或手动指定版本，例如：bash install.sh v2.9.10${plain}"
            exit 1
        fi
        tag_version="${RELEASE_TAG}"
        if [[ "${RELEASE_TAG_SOURCE}" == release-list* ]]; then
            echo -e "${yellow}已回退到最新公开 Release：${tag_version}。建议将稳定版本标记为正式 Release。${plain}"
        fi
        echo -e "已获取 SuperXray 版本：${tag_version}，开始安装..."
        if ! curl -4fLRo "$release_stage/release.tar.gz" "${release_base}/releases/download/${tag_version}/x-ui-linux-${package_arch}.tar.gz"; then
            echo -e "${red}下载 x-ui 失败，请确认服务器可以访问 GitHub ${plain}"
            exit 1
        fi
    else
        tag_version=$1
        tag_version_numeric=${tag_version#v}
        min_version="2.3.5"
        
        if [[ "$(printf '%s\n' "$min_version" "$tag_version_numeric" | sort -V | head -n1)" != "$min_version" ]]; then
            echo -e "${red}请使用较新的版本（至少 v2.3.5），安装已退出。${plain}"
            exit 1
        fi
        
        url="${release_base}/releases/download/${tag_version}/x-ui-linux-${package_arch}.tar.gz"
        echo -e "开始安装 x-ui $1"
        if ! curl -4fLRo "$release_stage/release.tar.gz" "$url"; then
            echo -e "${red}下载 x-ui $1 失败，请检查该版本是否存在 ${plain}"
            exit 1
        fi
    fi
    if ! extract_release; then
        echo "安装包无效，已取消安装。" >&2
        exit 1
    fi
    raw_base="https://raw.githubusercontent.com/${script_repo}/${tag_version}"
    if ! curl -4fLRo "$release_stage/menu.sh" "${raw_base}/x-ui.sh"; then
        echo -e "${red}下载 x-ui.sh 失败${plain}"
        exit 1
    fi
    
    if [[ -f "$xui_folder/x-ui" ]]; then
        stop_release_service || exit 1
    fi
    switch_release || exit 1
    cd "$xui_folder" || exit 1
    install_release_menu || exit 1
    mkdir -p /var/log/x-ui
    config_after_install || exit 1

    # Etckeeper compatibility
    if [ -d "/etc/.git" ]; then
        if [ -f "/etc/.gitignore" ]; then
            if ! grep -q "x-ui/x-ui.db" "/etc/.gitignore"; then
                echo "" >> "/etc/.gitignore"
                echo "x-ui/x-ui.db" >> "/etc/.gitignore"
            echo -e "${green}已将 x-ui.db 添加到 /etc/.gitignore 以兼容 etckeeper${plain}"
            fi
        else
            echo "x-ui/x-ui.db" > "/etc/.gitignore"
            echo -e "${green}已创建 /etc/.gitignore，并添加 x-ui.db 以兼容 etckeeper${plain}"
        fi
    fi
    
    if [[ $release == "alpine" ]]; then
        if ! curl -4fLRo /etc/init.d/x-ui "${raw_base}/x-ui.rc"; then
            echo -e "${red}下载 x-ui.rc 失败${plain}"
            exit 1
        fi
        configure_unit_path || exit 1
        chmod +x /etc/init.d/x-ui
        rc-update add x-ui || exit 1
        rc-service x-ui start || exit 1
    else
        # Install systemd service file
        service_installed=false
        
        if [ -f "x-ui.service" ]; then
            echo -e "${green}已在解压文件中找到 x-ui.service，正在安装...${plain}"
            if cp -f x-ui.service "${xui_service}/" >/dev/null 2>&1; then
                service_installed=true
            fi
        fi
        
        if [ "$service_installed" = false ]; then
            case "${release}" in
                ubuntu | debian | armbian)
                    if [ -f "x-ui.service.debian" ]; then
                        echo -e "${green}已在解压文件中找到 x-ui.service.debian，正在安装...${plain}"
                        if cp -f x-ui.service.debian "${xui_service}/x-ui.service" >/dev/null 2>&1; then
                            service_installed=true
                        fi
                    fi
                ;;
                arch | manjaro | parch)
                    if [ -f "x-ui.service.arch" ]; then
                        echo -e "${green}已在解压文件中找到 x-ui.service.arch，正在安装...${plain}"
                        if cp -f x-ui.service.arch "${xui_service}/x-ui.service" >/dev/null 2>&1; then
                            service_installed=true
                        fi
                    fi
                ;;
                *)
                    if [ -f "x-ui.service.rhel" ]; then
                        echo -e "${green}已在解压文件中找到 x-ui.service.rhel，正在安装...${plain}"
                        if cp -f x-ui.service.rhel "${xui_service}/x-ui.service" >/dev/null 2>&1; then
                            service_installed=true
                        fi
                    fi
                ;;
            esac
        fi
        
        # If service file not found in tar.gz, download from GitHub
        if [ "$service_installed" = false ]; then
            echo -e "${yellow}压缩包中未找到服务文件，正在从 GitHub 下载...${plain}"
            local service_suffix
            case "$release" in
                ubuntu | debian | armbian) service_suffix=debian ;;
                arch | manjaro | parch) service_suffix=arch ;;
                *) service_suffix=rhel ;;
            esac
            if ! curl -4fLRo "$xui_service/x-ui.service" "$raw_base/x-ui.service.$service_suffix" >/dev/null 2>&1; then
                echo -e "${red}从 GitHub 安装 x-ui.service 失败${plain}"
                exit 1
            fi
            service_installed=true
        fi
        
        if [ "$service_installed" = true ]; then
            echo -e "${green}正在配置 systemd 服务...${plain}"
            configure_unit_path || exit 1
            chown root:root "${xui_service}/x-ui.service" >/dev/null 2>&1
            chmod 644 "${xui_service}/x-ui.service" >/dev/null 2>&1
            systemctl daemon-reload || exit 1
            systemctl enable x-ui || exit 1
            systemctl start x-ui || exit 1
        else
            echo -e "${red}安装 x-ui.service 文件失败${plain}"
            exit 1
        fi
    fi
    
    echo -e "${green}x-ui ${tag_version}${plain} 安装完成，当前正在运行..."
    echo -e ""
    echo -e "┌────────────────────────────────────────────────────────────┐
│  ${blue}SuperXray管理命令（子命令）:${plain}                              │
│                                                            │
│  ${blue}x-ui${plain}                       - 打开交互式管理菜单       │
│  ${blue}x-ui start${plain}                 - 启动面板服务             │
│  ${blue}x-ui stop${plain}                  - 停止面板服务             │
│  ${blue}x-ui restart${plain}               - 重启面板服务             │
│  ${blue}x-ui restart-xray${plain}          - 仅重启 Xray 内核         │
│  ${blue}x-ui status${plain}                - 查看运行状态             │
│  ${blue}x-ui settings${plain}              - 查看当前配置             │
│  ${blue}x-ui enable${plain}                - 开启开机自启             │
│  ${blue}x-ui disable${plain}               - 关闭开机自启             │
│  ${blue}x-ui log${plain}                   - 查看面板日志             │
│  ${blue}x-ui banlog${plain}                - 查看 fail2ban 封禁日志   │
│  ${blue}x-ui update${plain}                - 更新面板程序             │
│  ${blue}x-ui update-all-geofiles${plain}   - 更新 GeoIP/GeoSite 数据  │
│  ${blue}x-ui legacy${plain}                - 安装指定历史版本         │
│  ${blue}x-ui install${plain}               - 安装面板                 │
│  ${blue}x-ui uninstall${plain}             - 卸载面板                 │
└────────────────────────────────────────────────────────────┘"
}

echo -e "${green}开始执行安装...${plain}"
install_base
install_x-ui "$@"
