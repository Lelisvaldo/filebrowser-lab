// Passo a passo para cadastrar o aplicativo de cada nuvem (feito uma vez por quem instala).
// Cada passo: t = texto, link = página a abrir, copy = [rótulo, texto] botões de copiar ("{callback}" vira o endereço de retorno).
export const GUIDES = {
  dropbox: {
    portal: "https://www.dropbox.com/developers/apps",
    idLabel: "App key", secretLabel: "App secret",
    steps: [
      { t: "Entre com a sua conta do Dropbox e toque em “Create app”.", link: "https://www.dropbox.com/developers/apps/create" },
      { t: "Escolha “Scoped access” e depois “Full Dropbox”. Dê um nome qualquer e único (ex.: filebrowser-lab-seunome)." },
      { t: "Na aba Permissions, marque as cinco permissões abaixo e toque em Submit.", copy: [["Copiar a lista", "files.metadata.read files.metadata.write files.content.read files.content.write account_info.read"]] },
      { t: "Na aba Settings, em “Redirect URIs”, cole o endereço de retorno e toque em Add.", copy: [["Copiar endereço de retorno", "{callback}"]] },
      { t: "Ainda em Settings, copie o “App key” e o “App secret” (toque em Show) e cole abaixo." },
    ],
  },
  onedrive: {
    portal: "https://entra.microsoft.com",
    idLabel: "ID do aplicativo (cliente)", secretLabel: "Segredo do cliente (coluna Valor)",
    steps: [
      { t: "Entre com a sua conta Microsoft em Entra → Aplicativos → Registros de aplicativo → Novo registro. Conta pessoal sem diretório: abra antes o portal.azure.com uma vez.", link: "https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" },
      { t: "Dê um nome. Em tipos de conta, escolha “Contas em qualquer diretório organizacional e contas pessoais da Microsoft”." },
      { t: "Em URI de redirecionamento, escolha a plataforma “Web”, cole o endereço de retorno e toque em Registrar.", copy: [["Copiar endereço de retorno", "{callback}"]] },
      { t: "A página do aplicativo mostra três IDs. Copie só o primeiro, **ID do aplicativo (cliente)**, e cole abaixo. Ignore o “ID do Objeto” e o “ID do diretório (locatário)”." },
      { t: "Em Certificados e segredos → Novo segredo do cliente, crie um e copie o campo **Valor** (ele só aparece uma vez) para o campo Segredo abaixo." },
      { t: "Em Permissões de API → Adicionar → Microsoft Graph → Permissões delegadas, marque as três abaixo.", copy: [["Copiar a lista", "Files.ReadWrite.All offline_access User.Read"]] },
    ],
  },
  drive: {
    portal: "https://console.cloud.google.com",
    idLabel: "ID do cliente", secretLabel: "Chave secreta do cliente",
    steps: [
      { t: "Entre com a sua conta Google e crie um projeto novo (ex.: filebrowser-lab).", link: "https://console.cloud.google.com/projectcreate" },
      { t: "Em APIs e serviços → Biblioteca, procure “Google Drive API” e toque em Ativar.", link: "https://console.cloud.google.com/apis/library/drive.googleapis.com" },
      { t: "Em Tela de permissão OAuth, escolha “Externo”, dê um nome, ponha o seu email e salve. Adicione o seu email como usuário de teste." },
      { t: "Depois toque em “Publicar aplicativo”. Sem isso o Google apaga o seu login a cada 7 dias. Ao entrar, ele avisa que o app não foi verificado: toque em Avançado → continuar. É o seu próprio app." },
      { t: "Em Credenciais → Criar credenciais → ID do cliente OAuth, escolha “Aplicativo da Web”. Em URIs de redirecionamento, cole o endereço de retorno.", copy: [["Copiar endereço de retorno", "{callback}"]], link: "https://console.cloud.google.com/apis/credentials" },
      { t: "Copie o ID do cliente e a chave secreta e cole abaixo." },
    ],
  },
};
