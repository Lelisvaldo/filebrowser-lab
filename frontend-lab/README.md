# frontend-lab

*English:* the own web interface of **filebrowser-lab** (Vue 3 + Vite, no UI framework). It replaces the Filebrowser Quantum UI (`../frontend/`, left untouched to ease upstream updates) and talks to the Go backend API (`../backend`). Build: `npm install && npm run build` (outputs to `../backend/internal/web/dist`, which the Go server serves; restart the backend after building). The UI language is Brazilian Portuguese. Icons: Fluent System Icons (MIT), regenerated with `npm run icons`. USB drives need `FILEBROWSER_LAB_USB_URL` (see `../backend/internal/web/usb.go`).

*Português (Brasil):* abaixo.

Interface própria do **filebrowser-lab** (Vue 3 + Vite). Substitui a interface do Filebrowser Quantum (`../frontend/`, que fica intocado para facilitar a atualização do original). Fala com a API do backend em Go (`../backend`).

| Pasta | O que tem |
|--|--|
| `src/views` | Login, "Este servidor", navegador de arquivos, busca, minha conta, administração, link público |
| `src/components` | Diálogos, menu de contexto, visualizador, transferências, configurações |
| `src/store` | Estado global (`app.js`), fila de envios com progresso real (`transfers.js`), diálogos |
| `src/api` | Cliente da API (arquivos, login, usuários, pendrives, links públicos) |
| `src/css` | `base.css` (identidade visual da prévia) e `app.css` (restante) |
| `public/index.html` | Template Go: o backend preenche `{{ .globalVars }}` e `{{ .cspNonce }}` |

## Compilar

```bash
npm install
npm run build        # gera ../backend/internal/web/dist (o Go serve essa pasta)
```

Na imagem Docker isso é feito por `../_docker/Dockerfile.lab`. O template do `index.html` é lido pelo Go na partida: depois de compilar, reinicie o backend.

## Ícones

Fluent System Icons (Microsoft, MIT). `npm run icons` regenera `src/lib/icons.js` a partir do pacote `@fluentui/svg-icons` (não fica nas dependências).

## Pendrives

O backend expõe `GET /api/usb` e `POST /api/usb/{uuid}/eject` quando `FILEBROWSER_LAB_USB_URL` aponta para o usb-manager do servidor (ver `../backend/internal/web/usb.go`). Sem a variável, a interface esconde a seção.
