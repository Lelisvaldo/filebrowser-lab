# filebrowser-lab

Fork do [Filebrowser Quantum](https://github.com/gtsteffaniak/filebrowser) (v2.0.2-stable, Apache-2.0) com **interface própria**.

| O quê | Onde |
|--|--|
| Interface nova (Vue 3 + Vite, só `vue`, `vue-router`, `marked`, `dompurify`) | `frontend-lab/` |
| Imagem Docker que compila a interface nova no lugar da do Quantum | `_docker/Dockerfile.lab` |
| Ponte opcional com um serviço de pendrives (lista e remoção segura) | `backend/internal/web/usb.go` |
| Nome do servidor para a interface | `GET /api/lab/host` |

O `frontend/` original do Quantum fica intocado, para trazer versões novas do original com poucos conflitos.

## Pendrives (opcional)

Ligada só quando `FILEBROWSER_LAB_USB_URL` está definida. Variáveis: `FILEBROWSER_LAB_USB_ORIGIN`, `FILEBROWSER_LAB_USB_HOST_ROOT`, `FILEBROWSER_LAB_USB_MOUNT_ROOT`, `FILEBROWSER_LAB_USB_SOURCE`, `FILEBROWSER_LAB_HOST_NAME`. Sem a URL, as rotas respondem 404 e a interface esconde a seção.

## Compilar

```bash
docker build -f _docker/Dockerfile.lab -t filebrowser-lab:dev .
```

Detalhes da interface: `frontend-lab/README.md`. Licença: a do projeto original.
