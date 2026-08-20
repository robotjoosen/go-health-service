#!/usr/bin/env bash
#
# Uninstalls health-service: stops and disables the systemd service, then
# removes the unit file, binary, and version marker left by install.sh /
# update.sh.
#
# Usage: run this directly on the device where health-service is installed:
#   ./scripts/uninstall.sh
#
# Requires: sudo, systemctl.

set -euo pipefail

SERVICE_NAME="health_service"
INSTALL_PATH="/usr/local/bin/health-service"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
VERSION_MARKER="/usr/local/bin/.health-service-version"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m!!\033[0m %s\n' "$*"; }
error() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; }

confirm() {
  # confirm "question" -- returns 0 (yes) or 1 (no). Defaults to no.
  # Reads from /dev/tty so this still works when piped via `curl | bash`.
  local reply
  read -r -p "$1 [y/N] " reply < /dev/tty
  case "$reply" in
    [yY][eE][sS]|[yY]) return 0 ;;
    *) return 1 ;;
  esac
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    error "'$1' is required but not installed. Install it and re-run this script."
    exit 1
  fi
}

main() {
  require_cmd sudo
  require_cmd systemctl

  if [ ! -f "$UNIT_PATH" ] && [ ! -f "$INSTALL_PATH" ]; then
    warn "health-service doesn't look installed (no ${UNIT_PATH} or ${INSTALL_PATH}) -- nothing to do"
    exit 0
  fi

  echo
  info "about to uninstall health-service:"
  echo "    stop + disable:  ${SERVICE_NAME}"
  echo "    remove unit:     ${UNIT_PATH}"
  echo "    remove binary:   ${INSTALL_PATH}"
  echo

  if [ -f "$UNIT_PATH" ]; then
    if confirm "Stop and disable ${SERVICE_NAME}?"; then
      sudo systemctl stop "$SERVICE_NAME" 2>/dev/null || true
      sudo systemctl disable "$SERVICE_NAME" 2>/dev/null || true
      info "${SERVICE_NAME} stopped and disabled"
    else
      warn "leaving the service as-is -- continuing with the rest of the uninstall"
    fi

    if confirm "Remove the systemd unit at ${UNIT_PATH}?"; then
      sudo rm -f "$UNIT_PATH"
      sudo systemctl daemon-reload
      info "systemd unit removed"
    else
      warn "kept ${UNIT_PATH} -- systemd will still know about ${SERVICE_NAME}"
    fi
  else
    warn "no systemd unit found at ${UNIT_PATH}, skipping service removal"
  fi

  if [ -f "$INSTALL_PATH" ]; then
    if confirm "Remove the binary at ${INSTALL_PATH}?"; then
      sudo rm -f "$INSTALL_PATH"
      info "binary removed"
    else
      warn "kept ${INSTALL_PATH}"
    fi
  else
    warn "no binary found at ${INSTALL_PATH}, skipping"
  fi

  if [ -f "$VERSION_MARKER" ]; then
    sudo rm -f "$VERSION_MARKER"
  fi

  echo
  info "done. health-service has been uninstalled."
}

main "$@"
