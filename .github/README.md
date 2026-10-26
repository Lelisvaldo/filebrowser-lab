<div align="center">

  <img width="110" alt="filebrowser-lab logo" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/logo.svg">

  # filebrowser-lab

  **A fork of [FileBrowser Quantum](https://github.com/gtsteffaniak/filebrowser) with its own, colorful interface, a disk-space cleaner and USB drive support.**

  English · [Português (Brasil)](README.pt-BR.md)

  [![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](../LICENSE)
  ![Based on](https://img.shields.io/badge/based%20on-Quantum%20v2.0.2--stable-8a63ff)
  ![UI](https://img.shields.io/badge/UI-Vue%203-42b883)

  <img width="860" alt="Tour of the filebrowser-lab interface" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/tour.gif">
</div>

> This is **not** the official FileBrowser Quantum and is not affiliated with its author. The original project, docs and credit belong to [gtsteffaniak/filebrowser](https://github.com/gtsteffaniak/filebrowser). Everything below describes what **this fork** changes. The upstream README is still in the repository root: [README.md](../README.md).

## What is this?

The Quantum backend (Go) is kept, and the web interface is **replaced** by a new one written from scratch (`frontend-lab/`, Vue 3 + Vite, no UI framework). The original `frontend/` is left untouched so new Quantum releases can be merged with very few conflicts.

Built for a home server: fast to read at a glance, good on a phone, and with the tools a home lab actually needs.

## What is different

| | FileBrowser Quantum | **filebrowser-lab** |
|--|--|--|
| Web interface | Quantum UI | **New own UI**, dark / light / follow system |
| Color identity | Single accent | **Each root folder has its own color**; sub-folders inherit it, so you always know where you are |
| Disk-space analyzer | – | **Cleaner** ("Limpeza"): treemap map + sorted list, drill into folders, select and delete with the freed space shown |
| USB drives | – | **Pendrives** in the sidebar with usage bar and **safe eject** (optional bridge, see below) |
| Mobile | Responsive | Responsive + **pull-to-refresh**, one-hand menu, user menu with sign-out |
| Server name in the menu | – | Shows the server's host name |
| Uploads | Yes | Yes, with a **transfer dock** (real progress, speed, ETA) |
| UI language | Many | **Brazilian Portuguese only** (for now) |
| Following upstream | – | **Update routine**: brings a new Quantum release and re-applies our small backend patch, keeping the UI intact, with a one-command way back |

### Everything Quantum already does still works

Backend features are untouched: multiple sources, indexed search, folder sizes, thumbnails and previews, sharing links, users and access rules, activity log, API keys and Swagger, WebDAV, OIDC / LDAP / password + 2FA. The new interface exposes browsing, search, upload/download, copy/move, zip/unzip, viewer and text editor, share links, users, access rules, activity, account and API keys. Anything not yet surfaced in the new UI is still available through the backend config and API.

## Screenshots

<p align="center">
  <img width="420" alt="Folder listing" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/browse-folder.png">
  <img width="420" alt="Light theme" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/browse-light.png">
</p>
<p align="center">
  <img width="420" alt="Cleaner: map" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/cleanup-map.png">
  <img width="420" alt="Cleaner: list with selection" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/cleanup-list.png">
</p>
<p align="center">
  <img width="190" alt="Mobile menu" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/mobile-menu.png">
  <img width="190" alt="Mobile listing" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/mobile-browse.png">
  <img width="190" alt="Mobile cleaner" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/mobile-cleanup.png">
</p>

*Screenshots use sample data.*

## Run it

```bash
# build the image (compiles the new UI instead of the Quantum one)
docker build -f _docker/Dockerfile.lab -t filebrowser-lab:dev .

# run (see the Quantum docs for config.yaml and sources)
docker run -d -p 8080:80 -v ./data:/home/filebrowser/data -v /path/to/files:/srv/files filebrowser-lab:dev
```

UI development: see [frontend-lab/README.md](../frontend-lab/README.md).

### USB drives (optional)

Enabled only when `FILEBROWSER_LAB_USB_URL` points to a small companion service that lists drives and ejects them. Variables: `FILEBROWSER_LAB_USB_ORIGIN`, `FILEBROWSER_LAB_USB_HOST_ROOT`, `FILEBROWSER_LAB_USB_MOUNT_ROOT`, `FILEBROWSER_LAB_USB_SOURCE`. Without it the section is hidden. The server name shown in the menu can be set with `FILEBROWSER_LAB_HOST_NAME`. Code: [backend/internal/web/usb.go](../backend/internal/web/usb.go).

## Staying up to date with Quantum

A maintenance routine (kept by the author, outside this repository) starts from the new upstream tag, re-applies only our small backend patch (USB bridge, host-name endpoint), restores `frontend-lab/` byte-for-byte and builds a new image with its own tag. The previous code and image stay available for rollback.

## License

Apache-2.0, same as the original. Credit to [gtsteffaniak](https://github.com/gtsteffaniak) and the FileBrowser community. Icons: Fluent System Icons (Microsoft, MIT).
