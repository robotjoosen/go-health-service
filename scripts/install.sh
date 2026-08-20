#!/usr/bin/env bash
#
# Interactive installer for health-service.
#
# Downloads the latest GitHub release binary for this device's architecture,
# then walks through setting it up as a systemd service. Safe to re-run to
# update an existing install.
#
# Usage: run this directly on the target device:
#   ./scripts/install.sh
#
# Requires: curl, sudo, systemctl. Expects GitHub releases at
# github.com/robotjoosen/go-health-service to publish binary assets named
# health-service-linux-arm64 and health-service-linux-arm.

set -euo pipefail

REPO="robotjoosen/go-health-service"
SERVICE_NAME="health_service"
INSTALL_PATH="/usr/local/bin/health-service"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
VERSION_MARKER="/usr/local/bin/.health-service-version"

DEFAULT_MESSAGE_BUS_URL="amqp://guest:guest@localhost:5672"
MODE="PROD"
LOG_LEVEL="INFO"
MESSAGE_BUS_EXCHANGE="health"
MESSAGE_BUS_ROUTING_KEY="health.ping"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m!!\033[0m %s\n' "$*"; }
error() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; }

confirm() {
  # confirm "question" -- returns 0 (yes) or 1 (no). Defaults to no.
  # Reads from /dev/tty, not stdin -- when this script runs via
  # `curl ... | bash`, stdin is the script itself, not the terminal.
  local reply
  read -r -p "$1 [y/N] " reply < /dev/tty
  case "$reply" in
    [yY][eE][sS]|[yY]) return 0 ;;
    *) return 1 ;;
  esac
}

prompt_with_default() {
  # prompt_with_default "question" "default" -- echoes the chosen value.
  # Reads from /dev/tty for the same reason as confirm() above.
  local question="$1" default="$2" reply
  read -r -p "$question [$default]: " reply < /dev/tty
  if [ -z "$reply" ]; then
    echo "$default"
  else
    echo "$reply"
  fi
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    error "'$1' is required but not installed. Install it and re-run this script."
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    aarch64|arm64) echo "arm64" ;;
    armv6l|armv7l|arm) echo "arm" ;;
    *)
      error "unsupported architecture: $(uname -m) (health-service only publishes linux/arm64 and linux/arm binaries)"
      exit 1
      ;;
  esac
}

fetch_latest_release_json() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" || {
    error "failed to reach GitHub releases API for ${REPO}"
    exit 1
  }
}

release_tag() {
  printf '%s' "$1" | grep -o '"tag_name": *"[^"]*"' | head -1 | sed -E 's/.*"([^"]+)"$/\1/'
}

release_asset_url() {
  local release_json="$1" goarch="$2" asset_name="health-service-linux-${goarch}"
  local url
  url=$(printf '%s' "$release_json" | grep -o "\"browser_download_url\": *\"[^\"]*${asset_name}\"" | head -1 | sed -E 's/.*"([^"]+)"$/\1/')

  if [ -z "$url" ]; then
    error "no release asset named '${asset_name}' found on the latest ${REPO} release"
    error "(the release must publish a binary asset with exactly that name)"
    exit 1
  fi

  echo "$url"
}

main() {
  require_cmd curl
  require_cmd sudo
  require_cmd systemctl

  local goarch
  goarch=$(detect_arch)
  info "detected architecture: ${goarch}"

  info "looking up the latest health-service release..."
  local release_json tag asset_url
  release_json=$(fetch_latest_release_json)
  tag=$(release_tag "$release_json")
  asset_url=$(release_asset_url "$release_json" "$goarch")
  info "latest release: ${tag}"

  # Not `local` -- the EXIT trap below fires after main() returns, once its
  # local scope is gone, so tmp_binary must still be visible at that point.
  tmp_binary=$(mktemp)
  trap 'rm -f "$tmp_binary"' EXIT

  info "downloading binary..."
  curl -fsSL -o "$tmp_binary" "$asset_url"
  chmod +x "$tmp_binary"

  echo
  info "configuration (press enter to accept the default)"
  local message_bus_url
  message_bus_url=$(prompt_with_default "RabbitMQ URL" "$DEFAULT_MESSAGE_BUS_URL")

  echo
  info "about to install health-service as a systemd service (${SERVICE_NAME}) with:"
  echo "    message bus:  ${message_bus_url}"
  echo "    binary path:  ${INSTALL_PATH}"
  echo

  local service_was_active="false"
  if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    service_was_active="true"
    if confirm "${SERVICE_NAME} is currently running. Stop it before updating?"; then
      sudo systemctl stop "$SERVICE_NAME"
    else
      warn "leaving the running service in place -- the binary underneath it will still be replaced"
    fi
  fi

  if confirm "Install the binary to ${INSTALL_PATH}?"; then
    sudo cp "$tmp_binary" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
    echo "$tag" | sudo tee "$VERSION_MARKER" >/dev/null
    info "binary installed (${tag})"
  else
    warn "skipped installing the binary -- aborting, nothing else to do"
    exit 0
  fi

  if confirm "Write the systemd unit at ${UNIT_PATH}?"; then
    sudo bash -c "cat > '${UNIT_PATH}'" <<EOF
[Unit]
Description=Health Service application
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_PATH}
Restart=on-failure
User=${USER}
WorkingDirectory=/
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
Environment=MODE=${MODE}
Environment=LOG_LEVEL=${LOG_LEVEL}
Environment=MESSAGE_BUS_URL=${message_bus_url}
Environment=MESSAGE_BUS_EXCHANGE=${MESSAGE_BUS_EXCHANGE}
Environment=MESSAGE_BUS_ROUTING_KEY=${MESSAGE_BUS_ROUTING_KEY}

[Install]
WantedBy=multi-user.target
EOF
    info "systemd unit written"
  else
    warn "skipped writing the systemd unit -- aborting, nothing more to configure"
    exit 0
  fi

  if confirm "Reload systemd, enable, and start ${SERVICE_NAME} now?"; then
    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME"
    sudo systemctl start "$SERVICE_NAME"
    info "${SERVICE_NAME} started"
  else
    warn "skipped starting the service -- run 'sudo systemctl daemon-reload && sudo systemctl enable --now ${SERVICE_NAME}' manually when ready"
    exit 0
  fi

  echo
  info "done. check status with: systemctl status ${SERVICE_NAME}"
  info "tail logs with:          journalctl -u ${SERVICE_NAME} -f"
}

main "$@"
