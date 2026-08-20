# Health Service

## Quick start

On the target device:

```bash
curl -fsSL https://raw.githubusercontent.com/robotjoosen/go-health-service/main/scripts/install.sh | bash
```

Or download and inspect it first (recommended for anything piping into a shell):

```bash
curl -fsSLO https://raw.githubusercontent.com/robotjoosen/go-health-service/main/scripts/install.sh
chmod +x install.sh
./install.sh
```

It detects the device's architecture, pulls the matching binary from the
[latest release](https://github.com/robotjoosen/go-health-service/releases), and walks through
setting it up as a systemd service — asking for the RabbitMQ URL, confirming before every
system-changing step (writing the unit file, enabling/starting the service).

To update later:

```bash
curl -fsSL https://raw.githubusercontent.com/robotjoosen/go-health-service/main/scripts/update.sh | bash
```

`update.sh` preserves whatever's already configured and only asks for values genuinely missing
from the existing install (e.g. a setting a newer release added).

To remove it entirely:

```bash
curl -fsSL https://raw.githubusercontent.com/robotjoosen/go-health-service/main/scripts/uninstall.sh | bash
```

`uninstall.sh` stops and disables the service, then confirms before removing the unit file, the
binary, and the version marker.

None of `install.sh`, `update.sh`, or `uninstall.sh` are published as release assets — they're
fetched straight from `main` each time, same as the install command above. If you downloaded
`install.sh` to disk instead of piping it, fetch the other two the same way rather than expecting
them to already be sitting next to it — all three live under `scripts/`.

Already have the repo checked out? `task install`, `task update`, and `task uninstall` run the
same scripts.
