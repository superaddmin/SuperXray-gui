#!/usr/bin/env bash
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

# Load the helper without running the installation or service-management entrypoints.
for script in install.sh update.sh x-ui.sh; do
  # Only the helper is loaded, as each full script runs a management command.
  # shellcheck source=/dev/null
  source <(sed -n '/^is_port_in_use() {$/,/^}/p' "$repo/$script")
  for listener in ss netstat; do
    mkdir -p "$fixture/$listener"
    ln -s "$(command -v awk)" "$fixture/$listener/awk"
    printf '#!/bin/sh\nprintf "tcp 0 0 127.0.0.1:8443 0.0.0.0:* LISTEN\\n"\n' > "$fixture/$listener/$listener"
    chmod +x "$fixture/$listener/$listener"
    PATH="$fixture/$listener" is_port_in_use 8443
    if PATH="$fixture/$listener" is_port_in_use 443; then
      echo "$script confused ports 443 and 8443 using $listener" >&2
      exit 1
    fi
    unlink "$fixture/$listener/awk"
  done
done

mkdir -p "$fixture/archive/x-ui/bin"
printf 'new panel' > "$fixture/archive/x-ui/x-ui"
printf 'new core' > "$fixture/archive/x-ui/bin/xray-linux-amd64"
tar -czf "$fixture/release.tar.gz" -C "$fixture/archive" x-ui
tar -czf "$fixture/incomplete.tar.gz" -C "$fixture/archive" x-ui/x-ui
printf 'invalid archive' > "$fixture/broken.tar.gz"
for script in install.sh update.sh; do
  # Use the selected implementation without executing its installation entrypoint.
  # shellcheck source=/dev/null
  source <(sed -n '/^create_release_stage() {$/,/^}/p; /^extract_release() {$/,/^}/p; /^switch_release() {$/,/^}/p; /^finish_release() {$/,/^}/p; /^configure_unit_path() {$/,/^}/p; /^install_release_menu() {$/,/^}/p' "$repo/$script")
  xui_folder="$fixture/$script/custom panel"
  release=debian
  mkdir -p "$xui_folder/bin"
  printf 'local data' > "$xui_folder/bin/custom-geo.dat"
  for scenario in incomplete broken switch-failure success; do
    printf 'previous panel' > "$xui_folder/x-ui"
    if (
      xui_service="$fixture/units" create_release_stage || exit 1
      archive=$scenario
      case $scenario in switch-failure|success) archive=release ;; esac
      cp "$fixture/$archive.tar.gz" "$release_stage/release.tar.gz"
      package_arch=amd64 extract_release 2>/dev/null || exit 1
      if [[ $scenario == switch-failure ]]; then
        # switch_release invokes this command override through the sourced helper.
        # shellcheck disable=SC2317
        mv() {
          if [[ $3 == "$release_stage/unpacked/x-ui" ]]; then return 1; fi
          command mv "$@"
        }
      fi
      switch_release || exit 1
    ); then
      if [[ $scenario != success || $(cat "$xui_folder/x-ui") != 'new panel' ]]; then
        echo "$script incorrectly installed $scenario" >&2
        exit 1
      fi
    elif [[ $scenario == success || $(cat "$xui_folder/x-ui") != 'previous panel' ]]; then
      echo "$script lost previous installation during $scenario" >&2
      exit 1
    fi
    [[ $(cat "$xui_folder/bin/custom-geo.dat") == 'local data' ]]
  done
  (
    release_stage="$fixture/$script"
    release_unit="$release_stage/x-ui.service"
    cp "$repo/x-ui.service.$release" "$release_unit"
    chmod 644 "$release_unit"
    configure_unit_path
    grep -Fxq "WorkingDirectory=$xui_folder" "$release_unit"
    grep -Fxq "ExecStart=\"$xui_folder/x-ui\"" "$release_unit"
    if command -v systemd-analyze >/dev/null; then systemd-analyze verify "$release_unit"; fi
    cp "$repo/x-ui.rc" "$release_unit"
    release=alpine configure_unit_path
    # This is the unit generated for the temporary installation directory.
    # shellcheck source=/dev/null
    source "$release_unit"
    [[ ${command:?missing OpenRC command} == "$xui_folder/x-ui" ]]
    start_pre
    [[ $PWD == "$xui_folder" ]]
    cat > "$release_stage/menu.sh" <<'MENU'
#!/bin/bash
test "$XUI_MAIN_FOLDER/x-ui.sh" = "$XUI_MENU_TARGET" && test "$1" = "two words"
MENU
    # install_release_menu calls this override to keep the launcher in the fixture.
    # shellcheck disable=SC2317
    install() {
      if [[ $4 == /usr/bin/x-ui ]]; then command install "$1" "$2" "$3" "$release_stage/global-menu"
      else command install "$@"; fi
    }
    install_release_menu
    "$release_stage/global-menu" "two words"
  )
done

# Exercise both configuration entrypoints with one settings read per invocation.
(
  xui_folder="$fixture/path panel"
  mkdir -p "$xui_folder" "$fixture/path-bin"
  export PANEL_SNAPSHOT="$fixture/settings" PANEL_CALLS="$fixture/panel-calls"
  cat > "$fixture/path-bin/curl" <<'SH'
#!/bin/sh
printf '127.0.0.1\n200\n'
SH
  cat > "$xui_folder/x-ui" <<'SH'
#!/bin/sh
printf '%s\n' "$*" >> "$PANEL_CALLS"
case "$*" in
  'setting -show -getCert') cat "$PANEL_SNAPSHOT"; exit "${PANEL_READ_STATUS:-0}" ;;
  migrate) exit 0 ;;
  'setting -webBasePath '*) exit "${PANEL_WRITE_STATUS:-0}" ;;
  *) exit 1 ;;
esac
SH
  chmod +x "$fixture/path-bin/curl" "$xui_folder/x-ui"
  for script in install update; do
    # Load the configuration entrypoint without running a system installation.
    # shellcheck source=/dev/null
    source <(sed -n '/^read_panel_settings() {$/,/^}/p; /^config_after_.*() {$/,/^}/p; /^gen_random_string() {$/,/^}/p' "$repo/$script.sh")
    for scenario in short old-short root empty normalized-root read-error no-path no-cert no-port bad-port no-credential write-error; do
      : > "$PANEL_CALLS"
      export PANEL_READ_STATUS=0 PANEL_WRITE_STATUS=0
      {
        [[ $scenario == no-credential ]] || echo 'hasDefaultCredential: false'
        case $scenario in no-port) ;; bad-port) echo 'port: unknown' ;; *) echo 'port: 54321' ;; esac
        [[ $scenario == no-cert ]] || echo 'cert: /test/example/cert.pem'
        case $scenario in
          no-path) ;;
          root|write-error) echo 'webBasePath: /' ;;
          empty) echo 'webBasePath: ' ;;
          normalized-root) printf 'webBasePath: /super/\ninitializeWebBasePath: true\n' ;;
          short) printf 'webBasePath: /a/\ninitializeWebBasePath: false\n' ;;
          *) echo 'webBasePath: /b/' ;;
        esac
      } > "$PANEL_SNAPSHOT"
      if [[ $scenario == read-error ]]; then PANEL_READ_STATUS=1; fi
      if [[ $scenario == write-error ]]; then PANEL_WRITE_STATUS=1; fi
      result=0
      PATH="$fixture/path-bin:$PATH" green='' plain='' yellow='' red='' "config_after_$script" > "$fixture/path-log" 2>&1 || result=$?
      [[ $(grep -Fxc 'setting -show -getCert' "$PANEL_CALLS") == 1 ]]
      case $scenario in
        short|old-short) [[ $result == 0 ]] && ! grep -q 'setting -webBasePath' "$PANEL_CALLS" || exit 1 ;;
        root|empty|normalized-root) [[ $result == 0 ]] && grep -Eq '^setting -webBasePath [a-zA-Z0-9]{18}$' "$PANEL_CALLS" || exit 1 ;;
        write-error) [[ $result != 0 ]] && grep -q 'setting -webBasePath' "$PANEL_CALLS" || exit 1 ;;
        *) [[ $result != 0 && $(wc -l < "$PANEL_CALLS") == 1 ]] || exit 1 ;;
      esac
    done
  done
)

# Mirror the OpenRC command interface without signalling a real process.
for unit in "$repo"/x-ui.service.*; do
  grep -Fxq 'ExecStart=/usr/local/x-ui/x-ui' "$unit"
  grep -Fxq 'WorkingDirectory=/usr/local/x-ui/' "$unit"
done
(
  # The unit is the shell fragment executed by the OpenRC test below.
  # shellcheck source=/dev/null
  source "$repo/x-ui.rc"
  # The sourced OpenRC reload hook invokes these three framework commands.
  # shellcheck disable=SC2317
  ebegin() { :; }
  # shellcheck disable=SC2317
  eend() { return "$1"; }
  # shellcheck disable=SC2317
  start-stop-daemon() {
    [[ $# == 6 && $1 == --signal && $2 == USR1 && $3 == --pidfile && $4 == "${pidfile:?missing OpenRC pidfile}" && $5 == --exec && $6 == "${command:?missing OpenRC command}" ]]
  }
  [[ ${extra_started_commands:?missing OpenRC command list} == reload ]]
  RC_SVCNAME=x-ui reload
)

# No external queries or service operations. Exercise the real gateway CLI with
# a known listener, first using its Python fallback and then its jq implementation.
mkdir "$fixture/tools"
for tool in bash python3 awk grep sed tr paste; do
  ln -s "$(command -v "$tool")" "$fixture/tools/$tool"
done
printf '#!/bin/sh\nprintf "LISTEN 0 5 127.0.0.1:11801 0.0.0.0:*\\n"\n' > "$fixture/tools/ss"
chmod +x "$fixture/tools/ss"
export XRAY_CONFIG="$fixture/xray.json" GATEWAY_RUN_SOCKS_PROBES=false
export GATEWAY_EGRESS_HOST=127.0.0.1 GATEWAY_EGRESS_PORTS=11801 GATEWAY_ALLOWED_SOURCE_CIDR='' GATEWAY_CONTAINER=''
cat > "$fixture/valid.json" <<'JSON'
{"inbounds":[{"tag":"gateway-test","port":11801,"protocol":"socks","listen":"127.0.0.1","settings":{"auth":"noauth"}}],"routing":{"rules":[{"inboundTag":["gateway-test"],"outboundTag":"blocked"}]},"policy":{"system":{"statsOutboundUplink":true,"statsOutboundDownlink":true}},"observatory":{}}
JSON
for parser in python3 jq; do
  if [[ $parser == jq ]]; then
    ln -s "$(command -v jq)" "$fixture/tools/jq"
  fi
  cp "$fixture/valid.json" "$XRAY_CONFIG"
  PATH="$fixture/tools" bash "$repo/scripts/gateway-egress-validate.sh" > "$fixture/log"
  for invalid in wildcard nonloopback malformed; do
    case "$invalid" in
      wildcard) sed 's/"listen":"127.0.0.1",//' "$fixture/valid.json" > "$XRAY_CONFIG" ;;
      nonloopback) sed 's/127\.0\.0\.1/192.0.2.1/' "$fixture/valid.json" > "$XRAY_CONFIG" ;;
      malformed) printf '{' > "$XRAY_CONFIG" ;;
    esac
    if PATH="$fixture/tools" bash "$repo/scripts/gateway-egress-validate.sh" > "$fixture/log" 2>&1; then
      echo "$parser accepted $invalid configuration" >&2
      exit 1
    fi
    grep -q '^FAIL:' "$fixture/log"
  done
done

XUI_ENABLE_FAIL2BAN=false sh "$repo/DockerEntrypoint.sh" sh -c 'test "$1" = "two words"' sh 'two words'
if (cd "$fixture" && sh "$repo/DockerInit.sh" unknown >/dev/null 2>&1); then
  echo 'DockerInit accepted an unknown target architecture' >&2
  exit 1
fi
[[ ! -e "$fixture/build" ]]

(
  # Load just the menu functions. All command side effects go into this fixture.
  # shellcheck source=/dev/null
  source <(sed -n '/^reset_user() {$/,/^}/p; /^run_downloaded_script() ($/,/^)/p' "$repo/x-ui.sh")
  xui_folder="$fixture/menu panel"
  mkdir -p "$xui_folder" "$fixture/menu-bin" "$fixture/menu-tmp"
  export PATH="$fixture/menu-bin:$PATH" TMPDIR="$fixture/menu-tmp"
  export ARGS_OUTPUT="$fixture/panel-args" green='' plain=''
  ln -s "$(type -P true)" "$fixture/menu-bin/confirm"
  ln -s "$(type -P true)" "$fixture/menu-bin/confirm_restart"
  cat > "$xui_folder/x-ui" <<'SH'
#!/bin/sh
printf '%s\n' "$@" > "$ARGS_OUTPUT"
SH
  chmod +x "$xui_folder/x-ui"
  for answer in n y; do
    flag=false
    if [[ $answer == y ]]; then flag=true; fi
    reset_user < <(printf '%s\n' 'fixture user' 'fixture password' "$answer") > /dev/null
    diff -u <(printf '%s\n' setting -username 'fixture user' -password 'fixture password' "-resetTwoFactor=$flag") "$ARGS_OUTPUT"
  done

  cat > "$fixture/menu-bin/curl" <<'SH'
#!/bin/sh
test "$2" = -o || exit 2
cp "$DOWNLOAD_BODY" "$3" || exit 2
exit "$DOWNLOAD_STATUS"
SH
  chmod +x "$fixture/menu-bin/curl"
  export DOWNLOAD_BODY="$fixture/mock-download" DOWNLOAD_STATUS=18
  export DOWNLOAD_RESULT="$fixture/download-result"
  cat > "$DOWNLOAD_BODY" <<'SH'
printf '%s\n' "$1" > "$DOWNLOAD_RESULT"
SH
  if run_downloaded_script fixture bash 'two words'; then
    echo 'Reported success for an incomplete download' >&2
    exit 1
  fi
  [[ ! -e "$DOWNLOAD_RESULT" ]]
  DOWNLOAD_STATUS=0 run_downloaded_script fixture bash 'two words'
  [[ $(cat "$DOWNLOAD_RESULT") == 'two words' ]]
  [[ -z $(ls -A "$TMPDIR") ]]
)
echo "Shell regression checks passed"
