// Passo a passo para cadastrar o aplicativo de cada nuvem (feito uma vez por quem instala).
// Cada passo: t = texto, link = página a abrir, copy = [rótulo, texto] botões de copiar ("{callback}" vira o endereço de retorno).
export const GUIDES = {
  dropbox: {
    portal: "https://www.dropbox.com/developers/apps",
    idLabel: "App key", secretLabel: "App secret",
    steps: [
      { t: "Entre com a sua conta do Dropbox e toque em **Create app**.", link: "https://www.dropbox.com/developers/apps/create" },
      { t: "Escolha **Scoped access**, depois **Full Dropbox**. Dê um nome único (ex.: filebrowser-lab-seunome) e toque em Create app." },
      { t: "Na aba **Permissions**, marque as cinco permissões abaixo e toque em **Submit**. Faça isso antes de entrar, senão o Dropbox nega o acesso.", copy: [["Copiar a lista", "files.metadata.read files.metadata.write files.content.read files.content.write account_info.read"]] },
      { t: "Na aba **Settings**, em **Redirect URIs**, cole o endereço de retorno e toque em **Add**.", copy: [["Copiar endereço de retorno", "{callback}"]] },
      { t: "Ainda em Settings, copie o **App key** e o **App secret** (toque em Show) e cole nos campos abaixo." },
      { t: "Toque em **Salvar**. O Dropbox abre e pergunta se você autoriza o aplicativo: toque em **Permitir** e você volta para cá já conectado." },
    ],
  },
  onedrive: {
    portal: "https://entra.microsoft.com",
    idLabel: "ID do aplicativo (cliente)", secretLabel: "Segredo do cliente (coluna Valor)",
    steps: [
      { t: "Entre com a sua conta Microsoft em **Entra → Aplicativos → Registros de aplicativo → Novo registro**. Se a conta é pessoal e nunca usou o Azure, abra antes o portal.azure.com uma vez.", link: "https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade" },
      { t: "Dê um nome (ex.: FileBrowserLab). Em tipos de conta, escolha **Contas em qualquer diretório organizacional e contas pessoais da Microsoft**." },
      { t: "Em **URI de redirecionamento**, escolha a plataforma **Web**, cole o endereço de retorno e toque em **Registrar**.", copy: [["Copiar endereço de retorno", "{callback}"]] },
      { t: "Na **Visão geral** do aplicativo aparecem três IDs. Copie só o primeiro, **ID do aplicativo (cliente)**, para o primeiro campo abaixo. Ignore o ID do Objeto e o ID do diretório (locatário)." },
      { t: "Na linha **Credenciais de cliente**, toque em **Adicionar um certificado ou segredo**. Na aba **Segredos do cliente**, toque em **Novo segredo do cliente**, ponha uma descrição, deixe 24 meses e toque em **Adicionar**." },
      { t: "Na tabela aparecem **Descrição, Expira em, Valor** e **ID secreto**. Copie a coluna **Valor** agora (depois ela fica oculta) para o segundo campo abaixo. Não use o ID secreto. Esse segredo vale 24 meses: quando vencer, crie outro e cole aqui de novo." },
      { t: "Em **Permissões de APIs → Adicionar uma permissão → Microsoft Graph → Permissões delegadas**, marque as três abaixo e toque em Adicionar.", copy: [["Copiar a lista", "Files.ReadWrite.All offline_access User.Read"]] },
      { t: "Toque em **Salvar** abaixo. A Microsoft abre e pergunta se você autoriza o aplicativo: toque em **Aceitar** e você volta para cá já conectado." },
    ],
  },
  drive: {
    portal: "https://console.cloud.google.com",
    idLabel: "ID do cliente", secretLabel: "Chave secreta do cliente",
    steps: [
      { t: "Entre com a sua conta Google e **crie um projeto** novo (ex.: filebrowser-lab).", link: "https://console.cloud.google.com/projectcreate" },
      { t: "Em **APIs e serviços → Biblioteca**, procure **Google Drive API** e toque em **Ativar**.", link: "https://console.cloud.google.com/apis/library/drive.googleapis.com" },
      { t: "Abra **Google Auth Platform** (menu ☰ → APIs e serviços → Tela de permissão OAuth). Se pedir para começar, em **Branding** ponha um nome (ex.: filebrowser-lab) e o seu email, e salve.", link: "https://console.cloud.google.com/auth/overview" },
      { t: "Em **Público-alvo**, escolha **Externo** e, em **Usuários de teste**, adicione o seu email. Depois toque em **Publicar app**. Sem isso o Google apaga o seu login a cada 7 dias. Ao entrar, ele avisa que o app não foi verificado: toque em **Avançado → continuar**. O app é seu." },
      { t: "Em **Clientes → Criar cliente**, escolha **Aplicativo da Web**. Em **URIs de redirecionamento autorizados**, cole o endereço abaixo. O Google só aceita domínio público ou localhost, por isso o retorno é http://localhost: no fim do login você copia o endereço que aparecer e cola aqui.", copy: [["Copiar endereço de retorno", "{callback}"]], link: "https://console.cloud.google.com/auth/clients" },
      { t: "Copie o **ID do cliente** e a **chave secreta** e cole nos campos abaixo." },
      { t: "Toque em **Salvar**. Abre o login do Google: autorize, copie o endereço da página que não abrir (começa com http://localhost) e cole no campo que aparece aqui." },
    ],
  },
};
