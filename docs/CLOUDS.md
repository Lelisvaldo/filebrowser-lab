# Connecting clouds (Google Drive, OneDrive, Dropbox, Mega)

English · [Português (Brasil)](NUVENS.md)

Clouds show up as folders inside the **Nuvens** source. You sign in under **Administração → Conectar serviços** and rclone mounts each service as a folder.

## What you need

| Piece | Why |
|--|--|
| A shared folder at `FILEBROWSER_LAB_CLOUD_DIR` | filebrowser-lab writes the `rclone.conf` with the logins there |
| An rclone container that reads that `rclone.conf` and mounts each service (`rclone mount`, FUSE) into a folder that filebrowser-lab also serves as the **Nuvens** source | It is what actually lists the files. This repository **does not ship** that container yet |
| `FILEBROWSER_LAB_CLOUD_ORIGIN` (e.g. `https://files.example`) | The site's public address; the sign-in callback is `<address>/api/lab/cloud/callback` |

Rules for every cloud:

- Clouds are mounted **read-only**. An admin enables writing with the pencil icon on the cloud's row.
- Deleting inside a cloud uses the **service's own trash**. Only Google Drive's trash is listed in the Trash screen.
- rclone remotes whose name starts with `_` are hidden from the mount.

## Everyone creates their own app

The project ships no access keys. You create a (free) app at the service and paste the **client ID** and **secret** into the admin screen. The secrets stay on your server only.

### Google Drive

1. In the [Google Cloud Console](https://console.cloud.google.com/), create a project and enable the **Google Drive API**.
2. Under **OAuth consent screen**, choose *External* and add your account as a test user. If a privacy policy is asked for, use [PRIVACY.md](../PRIVACY.md).
3. Under **Credentials**, create an **OAuth client ID** of type *Web application*. Use the redirect URI the admin screen shows for Google Drive (`http://localhost/api/lab/cloud/callback`).
4. Paste the ID and secret in **Conectar serviços** and sign in. Google redirects to an `http://localhost` address that will not open: **copy that address from the browser bar and paste it back** into the screen.
5. The server then shows `GoogleDrive/Meu Drive` and `GoogleDrive/Computadores` (Google's Backup and Sync folders); computers are discovered automatically.

### OneDrive

Register an app in **Microsoft Entra ID → App registrations** with the `Files.ReadWrite.All` permission and the redirect URI `<FILEBROWSER_LAB_CLOUD_ORIGIN>/api/lab/cloud/callback`.

### Dropbox

Create an app at [dropbox.com/developers/apps](https://www.dropbox.com/developers/apps) with the redirect URI `<FILEBROWSER_LAB_CLOUD_ORIGIN>/api/lab/cloud/callback`.

### Mega

No app needed: enter the e-mail and password in the admin screen.
