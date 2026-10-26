<div align="center">

  <img width="110" alt="logo do filebrowser-lab" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/logo.svg">

  # filebrowser-lab

  **Um fork do [FileBrowser Quantum](https://github.com/gtsteffaniak/filebrowser) com interface própria e colorida, limpeza de espaço em disco e suporte a pendrives.**

  [English](README.md) · Português (Brasil)

  [![Licença: Apache-2.0](https://img.shields.io/badge/Licen%C3%A7a-Apache_2.0-blue.svg)](../LICENSE)
  ![Baseado em](https://img.shields.io/badge/baseado%20em-Quantum%20v2.0.2--stable-8a63ff)
  ![Interface](https://img.shields.io/badge/interface-Vue%203-42b883)

  <img width="860" alt="Passeio pela interface do filebrowser-lab" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/tour.gif">
</div>

> Este **não** é o FileBrowser Quantum oficial e não tem ligação com o autor. O projeto original, a documentação e os créditos são de [gtsteffaniak/filebrowser](https://github.com/gtsteffaniak/filebrowser). Tudo abaixo descreve o que **este fork** muda. O README do original continua na raiz do repositório: [README.md](../README.md).

## O que é

O backend do Quantum (Go) foi mantido e a interface web foi **trocada** por uma nova, escrita do zero (`frontend-lab/`, Vue 3 + Vite, sem framework de interface). O `frontend/` original fica intocado, para trazer versões novas do Quantum com pouquíssimos conflitos.

Feito para servidor doméstico: fácil de ler de relance, bom no celular e com as ferramentas que um home lab realmente usa.

## O que é diferente

| | FileBrowser Quantum | **filebrowser-lab** |
|--|--|--|
| Interface web | Interface do Quantum | **Interface nova e própria**, escura / clara / segue o sistema |
| Identidade de cor | Um destaque só | **Cada pasta raiz tem a sua cor**; as subpastas herdam, então você sempre sabe onde está |
| Analisador de espaço | – | **Limpeza**: mapa em blocos + lista ordenada, entra nas pastas, seleciona e apaga mostrando o espaço liberado |
| Pendrives | – | **Pendrives** no menu lateral com barra de uso e **remoção segura** (ponte opcional, veja abaixo) |
| Celular | Responsivo | Responsivo + **puxar para atualizar**, menu para uma mão, menu do usuário com sair |
| Nome do servidor no menu | – | Mostra o nome (hostname) do servidor |
| Envios | Sim | Sim, com **painel de transferências** (progresso real, velocidade, tempo restante) |
| Idioma da interface | Vários | **Só português do Brasil** (por enquanto) |
| Acompanhar o original | – | **Rotina de atualização**: traz uma versão nova do Quantum e reaplica o nosso remendo pequeno do backend, mantendo a interface intacta, com caminho de volta em um comando |

### Tudo que o Quantum já faz continua funcionando

O backend não foi alterado: várias fontes, busca indexada, tamanho de pastas, miniaturas e prévias, links de compartilhamento, usuários e regras de acesso, registro de atividade, chaves de API e Swagger, WebDAV, OIDC / LDAP / senha + 2FA. A interface nova cobre navegar, buscar, enviar/baixar, copiar/mover, compactar/extrair, visualizador e editor de texto, links de compartilhamento, usuários, regras de acesso, atividade, conta e chaves de API. O que ainda não aparece na interface nova continua disponível pela configuração e pela API do backend.

## Capturas de tela

<p align="center">
  <img width="420" alt="Lista de uma pasta" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/browse-folder.png">
  <img width="420" alt="Tema claro" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/browse-light.png">
</p>
<p align="center">
  <img width="420" alt="Limpeza: mapa" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/cleanup-map.png">
  <img width="420" alt="Limpeza: lista com seleção" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/cleanup-list.png">
</p>
<p align="center">
  <img width="190" alt="Menu no celular" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/mobile-menu.png">
  <img width="190" alt="Lista no celular" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/mobile-browse.png">
  <img width="190" alt="Limpeza no celular" src="https://raw.githubusercontent.com/Lelisvaldo/filebrowser-lab/lab/docs/img/mobile-cleanup.png">
</p>

*As capturas usam dados de exemplo.*

## Como rodar

```bash
# compila a imagem (usa a interface nova no lugar da do Quantum)
docker build -f _docker/Dockerfile.lab -t filebrowser-lab:dev .

# executa (config.yaml e fontes: veja a documentação do Quantum)
docker run -d -p 8080:80 -v ./data:/home/filebrowser/data -v /caminho/dos/arquivos:/srv/files filebrowser-lab:dev
```

Desenvolvimento da interface: [frontend-lab/README.md](../frontend-lab/README.md).

### Pendrives (opcional)

Só liga quando `FILEBROWSER_LAB_USB_URL` aponta para um serviço pequeno que lista e ejeta os pendrives. Variáveis: `FILEBROWSER_LAB_USB_ORIGIN`, `FILEBROWSER_LAB_USB_HOST_ROOT`, `FILEBROWSER_LAB_USB_MOUNT_ROOT`, `FILEBROWSER_LAB_USB_SOURCE`. Sem isso a seção some. O nome do servidor no menu pode ser definido com `FILEBROWSER_LAB_HOST_NAME`. Código: [backend/internal/web/usb.go](../backend/internal/web/usb.go).

## Acompanhando o Quantum

Uma rotina de manutenção (mantida pelo autor, fora deste repositório) parte da tag nova do original, reaplica só o nosso remendo pequeno do backend (ponte de pendrives e endpoint do nome do servidor), restaura o `frontend-lab/` byte a byte e compila uma imagem nova com tag própria. O código e a imagem anteriores ficam guardados para voltar.

## Licença

Apache-2.0, igual ao original. Créditos a [gtsteffaniak](https://github.com/gtsteffaniak) e à comunidade do FileBrowser. Ícones: Fluent System Icons (Microsoft, MIT).
