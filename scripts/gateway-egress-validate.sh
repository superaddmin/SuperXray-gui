#!/usr/bin/env bash
set -u

# Read-only validation for SuperXray Gateway-facing egress entries.
# It intentionally avoids printing outbound settings or credentials.

GATEWAY_EGRESS_HOST="${GATEWAY_EGRESS_HOST:-127.0.0.1}"
GATEWAY_EGRESS_PORTS="${GATEWAY_EGRESS_PORTS:-11801 11802 11803 11901 11981}"
GATEWAY_OPENAI_PORT="${GATEWAY_OPENAI_PORT-11801}"
GATEWAY_ANTHROPIC_PORT="${GATEWAY_ANTHROPIC_PORT-11802}"
GATEWAY_GEMINI_PORT="${GATEWAY_GEMINI_PORT-11803}"
GATEWAY_NEGATIVE_PROBE_PORT="${GATEWAY_NEGATIVE_PROBE_PORT-$GATEWAY_OPENAI_PORT}"
GATEWAY_CONTAINER="${GATEWAY_CONTAINER:-}"
XRAY_CONFIG="${XRAY_CONFIG:-/usr/local/x-ui/bin/config.json}"
XUI_SERVICE="${XUI_SERVICE:-x-ui}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
CURL_MAX_TIME="${CURL_MAX_TIME:-30}"
GATEWAY_SECURITY_MODE="${GATEWAY_SECURITY_MODE:-test}"
GATEWAY_ALLOWED_SOURCE_CIDR="${GATEWAY_ALLOWED_SOURCE_CIDR:-}"
GATEWAY_RUN_SOCKS_PROBES="${GATEWAY_RUN_SOCKS_PROBES:-true}"
GATEWAY_SOCKS_USERNAME="${GATEWAY_SOCKS_USERNAME-}"
GATEWAY_SOCKS_PASSWORD="${GATEWAY_SOCKS_PASSWORD-}"

failures=0
warnings=0

section() {
  printf '\n== %s ==\n' "$1"
}

fail() {
  failures=$((failures + 1))
  printf 'FAIL: %s\n' "$1"
}

warn() {
  warnings=$((warnings + 1))
  printf 'WARN: %s\n' "$1"
}

pass() {
  printf 'PASS: %s\n' "$1"
}

have() {
  command -v "$1" >/dev/null 2>&1
}

validate_safe_token() {
  local name="$1"
  local value="$2"
  if [ -n "$value" ] && ! printf '%s' "$value" | grep -Eq '^[A-Za-z0-9._:-]+$'; then
    fail "$name contains characters outside the safe set [A-Za-z0-9._:-]"
    return 1
  fi
  return 0
}

is_loopback_host() {
  case "$1" in
    127.*|localhost|"::1"|"[::1]")
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

is_wildcard_host() {
  case "$1" in
    ""|"0.0.0.0"|"::"|"[::]"|"*"|"[::]:")
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

extract_listen_host() {
  local local_address="$1"
  case "$local_address" in
    \[*\]:*)
      printf '%s\n' "$local_address" | sed -E 's/^\[([^]]+)\]:[0-9]+$/\1/'
      ;;
    *:*)
      printf '%s\n' "$local_address" | sed -E 's/:([0-9]+)$//'
      ;;
    *)
      printf '%s\n' "$local_address"
      ;;
  esac
}

host_looks_private_or_local() {
  case "$1" in
    localhost|*.localhost|*.local|*.internal|127.*|0.*|10.*|192.168.*|169.254.*|100.64.*)
      return 0
      ;;
    172.1[6-9].*|172.2[0-9].*|172.3[0-1].*)
      return 0
      ;;
    "::1"|"[::1]"|fc*|fd*|fe80:*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

port_regex() {
  printf '%s\n' "$GATEWAY_EGRESS_PORTS" | tr ' ' '\n' | sed '/^$/d' | paste -sd '|' -
}

print_env_summary() {
  section "validation inputs"
  printf 'GATEWAY_EGRESS_HOST=%s\n' "$GATEWAY_EGRESS_HOST"
  printf 'GATEWAY_EGRESS_PORTS=%s\n' "$GATEWAY_EGRESS_PORTS"
  printf 'GATEWAY_OPENAI_PORT=%s\n' "${GATEWAY_OPENAI_PORT:-<empty>}"
  printf 'GATEWAY_ANTHROPIC_PORT=%s\n' "${GATEWAY_ANTHROPIC_PORT:-<empty>}"
  printf 'GATEWAY_GEMINI_PORT=%s\n' "${GATEWAY_GEMINI_PORT:-<empty>}"
  printf 'GATEWAY_NEGATIVE_PROBE_PORT=%s\n' "${GATEWAY_NEGATIVE_PROBE_PORT:-<empty>}"
  printf 'GATEWAY_CONTAINER=%s\n' "${GATEWAY_CONTAINER:-<empty>}"
  printf 'XRAY_CONFIG=%s\n' "$XRAY_CONFIG"
  printf 'XUI_SERVICE=%s\n' "$XUI_SERVICE"
  printf 'GATEWAY_SECURITY_MODE=%s\n' "$GATEWAY_SECURITY_MODE"
  printf 'GATEWAY_ALLOWED_SOURCE_CIDR=%s\n' "${GATEWAY_ALLOWED_SOURCE_CIDR:-<empty>}"
  printf 'GATEWAY_RUN_SOCKS_PROBES=%s\n' "$GATEWAY_RUN_SOCKS_PROBES"
  printf 'GATEWAY_SOCKS_USERNAME=%s\n' "$(if [ -n "$GATEWAY_SOCKS_USERNAME" ]; then printf '<set>'; else printf '<empty>'; fi)"
  printf 'GATEWAY_SOCKS_PASSWORD=%s\n' "$(if [ -n "$GATEWAY_SOCKS_PASSWORD" ]; then printf '<set>'; else printf '<empty>'; fi)"
  printf 'curl timeouts: connect=%ss max=%ss\n' "$CURL_CONNECT_TIMEOUT" "$CURL_MAX_TIME"
}

check_host_strategy() {
  section "host strategy"
  validate_safe_token "GATEWAY_EGRESS_HOST" "$GATEWAY_EGRESS_HOST" || true
  validate_safe_token "GATEWAY_CONTAINER" "$GATEWAY_CONTAINER" || true

  case "$GATEWAY_EGRESS_HOST" in
    ""|"0.0.0.0"|"::"|"[::]"|"*")
      fail "manifest host is empty or wildcard"
      ;;
    127.0.0.1|localhost)
      if [ -n "$GATEWAY_CONTAINER" ]; then
        fail "Gateway container is set but manifest host is loopback; container loopback will not reach host Xray"
      else
        pass "loopback manifest host is acceptable for same-network validation"
      fi
      ;;
    *)
      pass "manifest host is not wildcard or loopback-only"
      ;;
  esac
}

check_listeners() {
  section "local listener check"
  if ! have ss; then
    if [ "$GATEWAY_SECURITY_MODE" = "production" ]; then
      fail "ss is not available; production validation cannot verify listeners"
    else
      warn "ss is not available; skipping local listener check"
    fi
    return
  fi

  local ports
  ports="$(port_regex)"
  local output
  output="$(ss -ltnp 2>/dev/null | grep -E ":(${ports})([[:space:]]|$)" || true)"
  if [ -z "$output" ]; then
    fail "no Gateway-facing listener found on ports: $GATEWAY_EGRESS_PORTS"
  else
    pass "found Gateway-facing listening ports"
    printf '%s\n' "$output"
  fi

  while IFS= read -r line; do
    [ -n "$line" ] || continue
    local_address="$(printf '%s\n' "$line" | awk '{print $4}')"
    listen_host="$(extract_listen_host "$local_address")"
    if is_wildcard_host "$listen_host"; then
      fail "Gateway-facing listener is bound to wildcard address: $local_address"
    elif ! is_loopback_host "$listen_host"; then
      if [ -z "$GATEWAY_ALLOWED_SOURCE_CIDR" ]; then
        fail "non-loopback Gateway-facing listener requires GATEWAY_ALLOWED_SOURCE_CIDR evidence: $local_address"
      else
        warn "non-loopback listener requires firewall evidence limiting source to $GATEWAY_ALLOWED_SOURCE_CIDR: $local_address"
      fi
    fi
  done <<EOF
$output
EOF
}

check_firewall_evidence() {
  section "firewall source restriction evidence"
  if [ -z "$GATEWAY_ALLOWED_SOURCE_CIDR" ]; then
    warn "GATEWAY_ALLOWED_SOURCE_CIDR is empty; non-loopback production listeners are blocked"
    return
  fi

  local ports
  ports="$(port_regex)"
  local evidence
  evidence="$(
    {
      if have nft; then nft list ruleset 2>/dev/null; fi
      if have iptables-save; then iptables-save 2>/dev/null; fi
    } | grep -E "(${ports}|${GATEWAY_ALLOWED_SOURCE_CIDR})" || true
  )"

  if [ -z "$evidence" ]; then
    if [ "$GATEWAY_SECURITY_MODE" = "production" ]; then
      fail "no nftables/iptables evidence found for $GATEWAY_ALLOWED_SOURCE_CIDR and ports $GATEWAY_EGRESS_PORTS"
    else
      warn "no nftables/iptables evidence found for $GATEWAY_ALLOWED_SOURCE_CIDR and ports $GATEWAY_EGRESS_PORTS"
    fi
    return
  fi

  pass "firewall evidence mentions Gateway ports or allowed source"
  printf '%s\n' "$evidence" | sed -n '1,40p'
}

check_xray_config() {
  section "xray config shape"
  if [ ! -r "$XRAY_CONFIG" ]; then
    fail "Xray config is not readable: $XRAY_CONFIG"
    return
  fi
  if ! have jq; then
    if have python3; then
      check_xray_config_python
      return
    fi
    fail "jq and python3 are not available; cannot verify Xray config shape"
    return
  fi

  local gateway_inbounds
  gateway_inbounds="$(jq -r '.inbounds[]? | select((.tag // "") | startswith("gateway-")) | [.tag, (.listen // ""), (.port // ""), (.protocol // "")] | @tsv' "$XRAY_CONFIG")"
  if [ -z "$gateway_inbounds" ]; then
    fail "no gateway-* inbounds found in Xray config"
  else
    pass "gateway-* inbounds found"
    printf '%s\n' "$gateway_inbounds"
  fi

  local risky_inbounds
  risky_inbounds="$(jq -r '.inbounds[]? | select((.tag // "") | startswith("gateway-")) | select((.settings.auth // "") == "noauth") | [.tag, (.listen // ""), (.port // ""), (.settings.auth // "")] | @tsv' "$XRAY_CONFIG")"
  if [ -n "$risky_inbounds" ]; then
    while IFS="$(printf '\t')" read -r tag listen port auth; do
      if is_wildcard_host "$listen"; then
        fail "$tag uses noauth SOCKS on wildcard listen address"
      elif ! is_loopback_host "$listen"; then
        if [ -z "$GATEWAY_ALLOWED_SOURCE_CIDR" ]; then
          fail "$tag uses noauth SOCKS on non-loopback listen address $listen:$port without allowed source evidence"
        else
          warn "$tag uses noauth SOCKS on non-loopback listen address $listen:$port; verify firewall allows only $GATEWAY_ALLOWED_SOURCE_CIDR"
        fi
      fi
      [ "$auth" = "noauth" ] || true
    done <<EOF
$risky_inbounds
EOF
  fi

  local placeholder_count
  placeholder_count="$(jq '[.outbounds[]? | select(((.tag == "openai-egress") or (.tag == "anthropic-egress") or (.tag == "gemini-egress")) and (.protocol == "freedom") and has("_gatewayEgressMvp"))] | length' "$XRAY_CONFIG")"
  if [ "$placeholder_count" != "0" ]; then
    fail "platform egress outbounds still contain MVP freedom placeholders"
  else
    pass "platform egress outbounds are not MVP freedom placeholders"
  fi

  if jq -e '
    def listify: if type == "array" then . else [.] end;
    def meaningful_condition_count:
      to_entries
      | map(select(
          (.key as $key | ["type", "outboundTag", "_gatewayEgressMvp"] | index($key) | not)
          and (.value != null)
          and (.value != "")
          and (.value != [])
        ))
      | length;
    [.inbounds[]? | select((.tag // "") | startswith("gateway-")) | .tag] as $gatewayTags
    | any(.routing.rules[]?;
        (.outboundTag == "blocked")
        and (
          (._gatewayEgressMvp == true)
          or (((.inboundTag // []) | listify) as $tags | any($tags[]; . as $tag | $gatewayTags | index($tag)))
          or (meaningful_condition_count == 0)
        )
      )
  ' "$XRAY_CONFIG" >/dev/null; then
    pass "final blocked rule for generated gateway inbounds exists"
  else
    fail "missing final blocked rule for generated gateway inbounds"
  fi

  if jq -e '.policy.system.statsOutboundUplink == true and .policy.system.statsOutboundDownlink == true' "$XRAY_CONFIG" >/dev/null; then
    pass "outbound statistics are enabled"
  else
    fail "outbound statistics are not enabled"
  fi

  if jq -e '((.observatory? | type) == "object") or ((.burstObservatory? | type) == "object")' "$XRAY_CONFIG" >/dev/null; then
    pass "observatory or burstObservatory is configured"
  else
    fail "observatory and burstObservatory are missing"
  fi

  local probe_urls
  probe_urls="$(jq -r '[
    .observatory.probeURL?,
    .burstObservatory.probeURL?
  ] | .[] | select(. != null and . != "")' "$XRAY_CONFIG")"
  if [ -n "$probe_urls" ]; then
    while IFS= read -r probe_url; do
      validate_probe_url "$probe_url"
    done <<EOF
$probe_urls
EOF
  fi
}

check_xray_config_python() {
  python3 - "$XRAY_CONFIG" <<'PY' | while IFS='|' read -r level message; do
import ipaddress
import json
import sys
from urllib.parse import urlparse

path = sys.argv[1]

def emit(level, message):
    print(f"{level}|{message}")

def as_list(value):
    if value is None:
        return []
    if isinstance(value, list):
        return value
    return [value]

def meaningful_condition_count(rule):
    ignored = {"type", "outboundTag", "_gatewayEgressMvp"}
    count = 0
    for key, value in rule.items():
        if key in ignored:
            continue
        if value in (None, "", []):
            continue
        count += 1
    return count

def host_looks_private_or_local(host):
    if not host:
        return True
    lowered = host.lower()
    if lowered in {"localhost"} or lowered.endswith((".localhost", ".local", ".internal")):
        return True
    try:
        ip = ipaddress.ip_address(lowered)
    except ValueError:
        return False
    return ip.is_private or ip.is_loopback or ip.is_link_local or ip.is_reserved or ip.is_multicast

with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)

gateway_inbounds = [
    inbound for inbound in data.get("inbounds", []) or []
    if str(inbound.get("tag", "")).startswith("gateway-")
]
gateway_tags = {inbound.get("tag", "") for inbound in gateway_inbounds}
if gateway_inbounds:
    emit("PASS", "gateway-* inbounds found")
    for inbound in gateway_inbounds:
        emit("DATA", "\t".join([
            str(inbound.get("tag", "")),
            str(inbound.get("listen", "")),
            str(inbound.get("port", "")),
            str(inbound.get("protocol", "")),
        ]))
else:
    emit("FAIL", "no gateway-* inbounds found in Xray config")

for inbound in gateway_inbounds:
    settings = inbound.get("settings") or {}
    listen = str(inbound.get("listen", ""))
    port = str(inbound.get("port", ""))
    tag = str(inbound.get("tag", ""))
    if settings.get("auth") == "noauth":
        if listen in {"", "0.0.0.0", "::", "[::]", "*"}:
            emit("FAIL", f"{tag} uses noauth SOCKS on wildcard listen address")
        elif not (listen.startswith("127.") or listen in {"localhost", "::1", "[::1]"}):
            emit("WARN", f"{tag} uses noauth SOCKS on non-loopback listen address {listen}:{port}; verify firewall source restriction")

placeholder_count = sum(
    1 for outbound in data.get("outbounds", []) or []
    if outbound.get("tag") in {"openai-egress", "anthropic-egress", "gemini-egress"}
    and outbound.get("protocol") == "freedom"
    and "_gatewayEgressMvp" in outbound
)
if placeholder_count:
    emit("FAIL", "platform egress outbounds still contain MVP freedom placeholders")
else:
    emit("PASS", "platform egress outbounds are not MVP freedom placeholders")

blocked_guard = False
for rule in (data.get("routing", {}) or {}).get("rules", []) or []:
    if rule.get("outboundTag") != "blocked":
        continue
    inbound_tags = set(as_list(rule.get("inboundTag")))
    if rule.get("_gatewayEgressMvp") is True or (inbound_tags & gateway_tags) or meaningful_condition_count(rule) == 0:
        blocked_guard = True
        break
if blocked_guard:
    emit("PASS", "final blocked rule for generated gateway inbounds exists")
else:
    emit("FAIL", "missing final blocked rule for generated gateway inbounds")

system_policy = ((data.get("policy") or {}).get("system") or {})
if system_policy.get("statsOutboundUplink") is True and system_policy.get("statsOutboundDownlink") is True:
    emit("PASS", "outbound statistics are enabled")
else:
    emit("FAIL", "outbound statistics are not enabled")

observability = [
    value for value in (data.get("observatory"), data.get("burstObservatory"))
    if isinstance(value, dict)
]
if observability:
    emit("PASS", "observatory or burstObservatory is configured")
else:
    emit("FAIL", "observatory and burstObservatory are missing")

for value in observability:
    probe_url = value.get("probeURL")
    if not probe_url:
        continue
    parsed = urlparse(probe_url)
    if parsed.scheme not in {"http", "https"}:
        emit("FAIL", f"observatory probeURL must use http or https: {probe_url}")
    elif host_looks_private_or_local(parsed.hostname):
        emit("FAIL", f"observatory probeURL targets local/private host: {probe_url}")
    else:
        emit("PASS", f"observatory probeURL has public-looking host: {parsed.hostname}")
PY
    case "$level" in
      PASS)
        pass "$message"
        ;;
      WARN)
        warn "$message"
        ;;
      FAIL)
        fail "$message"
        ;;
      DATA)
        printf '%s\n' "$message"
        ;;
    esac
  done
}

validate_probe_url() {
  local url="$1"
  local host_port
  local host
  case "$url" in
    http://*|https://*)
      ;;
    *)
      fail "observatory probeURL must use http or https: $url"
      return
      ;;
  esac

  host_port="$(printf '%s' "$url" | sed -E 's#^[A-Za-z][A-Za-z0-9+.-]*://([^/@/]+).*#\1#')"
  case "$host_port" in
    \[*\]*)
      host="$(printf '%s' "$host_port" | sed -E 's/^\[([^]]+)\].*$/\1/')"
      ;;
    *)
      host="${host_port%%:*}"
      ;;
  esac
  if [ -z "$host" ] || host_looks_private_or_local "$host"; then
    fail "observatory probeURL targets local/private host: $url"
  else
    pass "observatory probeURL has public-looking host: $host"
  fi
}

curl_probe() {
  local scope="$1"
  local port="$2"
  local label="$3"
  local url="$4"
  local output

  if [ -n "$GATEWAY_SOCKS_USERNAME" ] || [ -n "$GATEWAY_SOCKS_PASSWORD" ]; then
    output="$(curl -sS -o /dev/null \
      --socks5-hostname "${GATEWAY_EGRESS_HOST}:${port}" \
      --proxy-user "${GATEWAY_SOCKS_USERNAME}:${GATEWAY_SOCKS_PASSWORD}" \
      --connect-timeout "$CURL_CONNECT_TIMEOUT" \
      --max-time "$CURL_MAX_TIME" \
      -w "platform=${label} http=%{http_code} connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer} total=%{time_total}" \
      "$url" 2>&1)"
  else
    output="$(curl -sS -o /dev/null \
      --socks5-hostname "${GATEWAY_EGRESS_HOST}:${port}" \
      --connect-timeout "$CURL_CONNECT_TIMEOUT" \
      --max-time "$CURL_MAX_TIME" \
      -w "platform=${label} http=%{http_code} connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer} total=%{time_total}" \
      "$url" 2>&1)"
  fi
  local code=$?
  printf '%s %s\n' "$scope" "$output"
  if [ "$code" -ne 0 ] || printf '%s' "$output" | grep -q 'http=000'; then
    fail "$scope probe failed for $label on port $port"
  else
    pass "$scope probe reached $label through port $port"
  fi
}

check_host_probes() {
  section "host socks5h probes"
  if [ "$GATEWAY_RUN_SOCKS_PROBES" = "false" ]; then
    warn "GATEWAY_RUN_SOCKS_PROBES=false; skipping host socks5h probes"
    return
  fi
  if ! have curl; then
    if [ "$GATEWAY_SECURITY_MODE" = "production" ]; then
      fail "curl is not available; production validation cannot run socks5h probes"
    else
      warn "curl is not available; skipping host socks5h probes"
    fi
    return
  fi

  if [ -n "$GATEWAY_OPENAI_PORT" ]; then
    curl_probe "host" "$GATEWAY_OPENAI_PORT" "openai" "https://api.openai.com/v1/models"
  fi
  if [ -n "$GATEWAY_ANTHROPIC_PORT" ]; then
    curl_probe "host" "$GATEWAY_ANTHROPIC_PORT" "anthropic" "https://api.anthropic.com/v1/messages"
  fi
  if [ -n "$GATEWAY_GEMINI_PORT" ]; then
    curl_probe "host" "$GATEWAY_GEMINI_PORT" "gemini" "https://generativelanguage.googleapis.com/\$discovery/rest?version=v1beta"
  fi
}

check_blocked_probe() {
  section "blocked rule negative probe"
  if [ "$GATEWAY_RUN_SOCKS_PROBES" = "false" ]; then
    warn "GATEWAY_RUN_SOCKS_PROBES=false; skipping blocked-rule negative probe"
    return
  fi
  if [ -z "$GATEWAY_NEGATIVE_PROBE_PORT" ]; then
    warn "GATEWAY_NEGATIVE_PROBE_PORT is empty; skipping blocked-rule negative probe"
    return
  fi
  if ! have curl; then
    if [ "$GATEWAY_SECURITY_MODE" = "production" ]; then
      fail "curl is not available; production validation cannot run blocked-rule negative probe"
    else
      warn "curl is not available; skipping blocked rule negative probe"
    fi
    return
  fi

  local output code
  if [ -n "$GATEWAY_SOCKS_USERNAME" ] || [ -n "$GATEWAY_SOCKS_PASSWORD" ]; then
    output="$(curl -sS -o /dev/null \
      --socks5-hostname "${GATEWAY_EGRESS_HOST}:${GATEWAY_NEGATIVE_PROBE_PORT}" \
      --proxy-user "${GATEWAY_SOCKS_USERNAME}:${GATEWAY_SOCKS_PASSWORD}" \
      --connect-timeout "$CURL_CONNECT_TIMEOUT" \
      --max-time 10 \
      -w "http=%{http_code} connect=%{time_connect} total=%{time_total}" \
      "https://example.com" 2>&1)"
  else
    output="$(curl -sS -o /dev/null \
      --socks5-hostname "${GATEWAY_EGRESS_HOST}:${GATEWAY_NEGATIVE_PROBE_PORT}" \
      --connect-timeout "$CURL_CONNECT_TIMEOUT" \
      --max-time 10 \
      -w "http=%{http_code} connect=%{time_connect} total=%{time_total}" \
      "https://example.com" 2>&1)"
  fi
  code=$?
  printf 'blocked-probe %s\n' "$output"
  if [ "$code" -eq 0 ] && ! printf '%s' "$output" | grep -q 'http=000'; then
    fail "negative probe port reached non-allowed domain; blocked guard may be ineffective"
  else
    pass "negative probe port did not act as a general-purpose proxy"
  fi
}

check_gateway_container_probes() {
  section "gateway container probes"
  if [ "$GATEWAY_RUN_SOCKS_PROBES" = "false" ]; then
    warn "GATEWAY_RUN_SOCKS_PROBES=false; skipping container-to-host reachability"
    return
  fi
  if [ -z "$GATEWAY_CONTAINER" ]; then
    warn "GATEWAY_CONTAINER is empty; skipping container-to-host reachability"
    return
  fi
  if [ -z "$GATEWAY_OPENAI_PORT" ]; then
    warn "GATEWAY_OPENAI_PORT is empty; skipping container openai probe"
    return
  fi
  if ! have docker; then
    fail "docker is not available but GATEWAY_CONTAINER is set"
    return
  fi

  local script='
set -u
host="$1"
port="$2"
url="$3"
label="$4"
connect_timeout="$5"
max_time="$6"
proxy_user="${GATEWAY_PROBE_SOCKS_USERNAME:-}"
proxy_pass="${GATEWAY_PROBE_SOCKS_PASSWORD:-}"
if ! command -v curl >/dev/null 2>&1; then
  echo "curl missing in container"
  exit 127
fi
if [ -n "$proxy_user" ] || [ -n "$proxy_pass" ]; then
  curl -sS -o /dev/null --socks5-hostname "${host}:${port}" --proxy-user "${proxy_user}:${proxy_pass}" --connect-timeout "$connect_timeout" --max-time "$max_time" -w "platform=${label} http=%{http_code} connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer} total=%{time_total}" "$url"
else
  curl -sS -o /dev/null --socks5-hostname "${host}:${port}" --connect-timeout "$connect_timeout" --max-time "$max_time" -w "platform=${label} http=%{http_code} connect=%{time_connect} tls=%{time_appconnect} ttfb=%{time_starttransfer} total=%{time_total}" "$url"
fi
'

  local output code
  output="$(
    GATEWAY_PROBE_SOCKS_USERNAME="$GATEWAY_SOCKS_USERNAME" \
    GATEWAY_PROBE_SOCKS_PASSWORD="$GATEWAY_SOCKS_PASSWORD" \
      docker exec \
        --env GATEWAY_PROBE_SOCKS_USERNAME \
        --env GATEWAY_PROBE_SOCKS_PASSWORD \
        "$GATEWAY_CONTAINER" sh -lc "$script" sh "$GATEWAY_EGRESS_HOST" "$GATEWAY_OPENAI_PORT" "https://api.openai.com/v1/models" "openai" "$CURL_CONNECT_TIMEOUT" "$CURL_MAX_TIME" 2>&1
  )"
  code=$?
  printf 'container %s\n' "$output"
  if [ "$code" -ne 0 ] || printf '%s' "$output" | grep -q 'http=000'; then
    fail "container probe failed for openai on port $GATEWAY_OPENAI_PORT"
  else
    pass "container probe reached openai through port $GATEWAY_OPENAI_PORT"
  fi
}

print_recent_logs_hint() {
  section "recent x-ui logs hint"
  if have journalctl; then
    journalctl -u "$XUI_SERVICE" -n 40 --no-pager 2>/dev/null | sed 's/[[:cntrl:]]//g' || true
  else
    warn "journalctl is not available; collect x-ui and xray logs manually"
  fi
}

main() {
  print_env_summary
  check_host_strategy
  check_listeners
  check_firewall_evidence
  check_xray_config
  check_host_probes
  check_blocked_probe
  check_gateway_container_probes
  print_recent_logs_hint

  section "summary"
  printf 'failures=%s warnings=%s\n' "$failures" "$warnings"
  if [ "$failures" -gt 0 ]; then
    exit 1
  fi
}

main "$@"
