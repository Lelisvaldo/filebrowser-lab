# Conectar nuvens (Google Drive, OneDrive, Dropbox, Mega)

[English](CLOUDS.md) · Português (Brasil)

As nuvens aparecem como pastas dentro da fonte **Nuvens**. O login é feito em **Administração → Conectar serviços** e o rclone monta cada serviço como uma pasta.

## O que você precisa

| Peça | Para quê |
|--|--|
| Uma pasta compartilhada em `FILEBROWSER_LAB_CLOUD_DIR` | O filebrowser-lab grava aí o `rclone.conf` com os logins |
| Um container rclone que lê esse `rclone.conf` e monta cada serviço (`rclone mount`, FUSE) numa pasta que também entra no filebrowser-lab como a fonte **Nuvens** | É quem de fato mostra os arquivos. Este repositório **ainda não traz** esse container pronto |
| `FILEBROWSER_LAB_CLOUD_ORIGIN` (ex.: `https://arquivos.exemplo`) | Endereço público do site; o retorno do login é `<endereço>/api/lab/cloud/callback` |

Regras que valem para todas as nuvens:

- As nuvens são montadas **somente leitura**. Para gravar, o administrador libera a escrita pelo ícone de lápis na linha da nuvem.
- Apagar numa nuvem usa a **lixeira do próprio serviço**. Só a lixeira do Google Drive aparece na tela Lixeira.
- Remotos do rclone cujo nome começa com `_` ficam ocultos na montagem.

## Cada pessoa cria o próprio aplicativo

O projeto não traz chaves de acesso. Você cria um aplicativo no serviço (gratuito) e cola o **ID do cliente** e o **segredo** na tela de administração. Os segredos ficam só no seu servidor.

### Google Drive

1. No [Google Cloud Console](https://console.cloud.google.com/), crie um projeto e ative a **Google Drive API**.
2. Em **Tela de consentimento OAuth**, escolha *Externo* e adicione a sua conta como usuário de teste. Se for aparecer a política de privacidade, use a de [PRIVACY.md](../PRIVACY.md).
3. Em **Credenciais**, crie um **ID do cliente OAuth** do tipo *Aplicativo da Web*. Como URI de redirecionamento, use o que a tela de administração mostra para o Google Drive (`http://localhost/api/lab/cloud/callback`).
4. Cole o ID e o segredo na tela **Conectar serviços** e entre com a sua conta. O Google redireciona para um endereço `http://localhost` que não abre: **copie esse endereço da barra do navegador e cole de volta** na tela.
5. O servidor mostra `GoogleDrive/Meu Drive` e `GoogleDrive/Computadores` (as pastas do Backup e Sincronização do Google); a descoberta dos computadores é automática.

### OneDrive

Registre um aplicativo em **Microsoft Entra ID → Registros de aplicativo**, com a permissão `Files.ReadWrite.All` e o URI de redirecionamento `<FILEBROWSER_LAB_CLOUD_ORIGIN>/api/lab/cloud/callback`.

### Dropbox

Crie um aplicativo em [dropbox.com/developers/apps](https://www.dropbox.com/developers/apps) com o URI de redirecionamento `<FILEBROWSER_LAB_CLOUD_ORIGIN>/api/lab/cloud/callback`.

### Mega

Não usa aplicativo: informe o e-mail e a senha na tela de administração.
