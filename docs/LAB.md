# filebrowser-lab: o que este fork acrescenta

Fork do FileBrowser Quantum (v2.0.2-stable). A interface própria fica em `frontend-lab/` (Vue 3); o `frontend/` original não é alterado. O código do servidor está em `backend/`, nos arquivos `lab_*.go`.

## Recursos

| Recurso | Resumo |
|--|--|
| Interface própria | Login sem barras de rolagem em qualquer resolução, tema claro/escuro, telas de Arquivos, Busca, Limpeza, Visão geral, Conta e Administração |
| Transferências | Envios, **mover** e **copiar** aparecem na fila de transferências ("Movendo…", "Copiando…") |
| Sem sobrescrita silenciosa | Mover/copiar para um nome que já existe cria `nome (2).ext`, e a tela avisa. Mover para a mesma pasta é recusado com aviso |
| Mover/Copiar para… | O seletor mostra os arquivos do destino e avisa, antes de confirmar, quando já existe um com o mesmo nome. Erros aparecem em texto claro (somente leitura, sem espaço, sem permissão, destino dentro de si mesmo) |
| Lixeira | Apagar move para `.lixeira` da própria fonte, por 30 dias. Tela **Lixeira** no rodapé do menu lateral, com restaurar, apagar de vez e esvaziar |
| Pendrives | Listagem e remoção segura por uma ponte com um serviço local de USB |
| Nuvens | Conexão de Google Drive, OneDrive, Dropbox e outros pela tela de Administração, montados via rclone |
| Arquivos compactados e Office | Listar, editar, criar e extrair arquivos compactados; converter documentos Office |

## Lixeira

- Retenção de 30 dias; uma limpeza automática roda a cada 6 horas.
- Cada usuário vê só o que apagou. O administrador vê tudo.
- Guardada em `<fonte>/.lixeira/<id>/<nome>` mais um `<id>.json` com origem, data e autor. A pasta `.lixeira` não é indexada e a API recusa acessá-la diretamente.
- **Pendrives e nuvens não usam a lixeira local.** A tela de apagar avisa:
  - Pendrives: a exclusão é direta.
  - Google Drive: segue a lixeira do próprio Google, que aparece na tela Lixeira (listar, restaurar, apagar de vez).
  - OneDrive, Mega e outros: o item vai para a lixeira do provedor e não aparece na nossa.

| Variável | Efeito |
|--|--|
| `FILEBROWSER_LAB_TRASH_DAYS` | Dias de retenção (padrão 30) |
| `FILEBROWSER_LAB_TRASH_OFF` | Fontes sem lixeira local, separadas por vírgula (padrão `Pendrives,Nuvens`) |
| `FILEBROWSER_LAB_TRASH=off` | Desliga a lixeira |

## Nuvens (rclone)

- O administrador conecta o serviço em **Administração → Conectar serviços**. As chaves do aplicativo OAuth são coladas na tela e guardadas no servidor; nunca vão para o Git.
- As nuvens são montadas **somente leitura**. Para gravar, o administrador libera a escrita (ícone de lápis na linha da nuvem).
- **Google Drive** aparece como `GoogleDrive/Meu Drive` e `GoogleDrive/Computadores`. Os computadores (pastas do backup do Google) são descobertos sozinhos ao abrir a tela e em segundo plano; também há o botão de buscar de novo.
- Remotes cujo nome começa com `_` ficam ocultos na montagem.
- Também mostra espaço usado e livre de cada nuvem.

| Variável | Efeito |
|--|--|
| `FILEBROWSER_LAB_CLOUD_DIR` | Pasta compartilhada com o rclone (onde fica o `rclone.conf`) |
| `FILEBROWSER_LAB_CLOUD_ORIGIN` | Endereço público usado no retorno do login OAuth |

## Rotas novas (`/api/lab/...`)

| Rota | Quem | Para quê |
|--|--|--|
| `GET /lab/trash`, `POST /lab/trash/restore`, `POST /lab/trash/purge` | usuário | Lixeira local |
| `GET /lab/cloud/trash`, `POST /lab/cloud/trash/restore`, `POST /lab/cloud/trash/purge` | admin | Lixeira do Google Drive |
| `GET/DELETE /lab/cloud`, `PUT /lab/cloud/app`, `POST /lab/cloud/start`, `/senha`, `/colar`, `/mode`, `/computadores`, `GET /lab/cloud/quota` | admin | Conectar e administrar nuvens |
| `GET/POST /lab/archive/*`, `/lab/office/*` | usuário | Arquivos compactados e Office |

## Compilar e testar

```bash
# testes do servidor (sem Go instalado no host)
docker run --rm -v "$PWD/backend":/src -w /src golang:alpine go test ./...
```

A interface é compilada na imagem pelo `_docker/Dockerfile.lab`. A política de privacidade usada no aplicativo OAuth do Google está em [PRIVACY.md](../PRIVACY.md).
