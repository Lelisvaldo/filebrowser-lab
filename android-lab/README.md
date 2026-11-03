# filebrowser-lab para Android

App Android (Capacitor) que abre o seu servidor filebrowser-lab num WebView, com as cores do site.

## O que faz (versão 0.2)

- Tela de endereço do servidor (guardado no aparelho) e aviso se o servidor não responde.
- Bloqueio por biometria ou PIN ao abrir e depois de 30 s em segundo plano (desligável em Conta → Segurança).
- **Compartilhar para o servidor**: em qualquer app, Compartilhar → Filebrowser Lab; escolha a pasta e toque em Enviar. O envio usa o painel de transferências do site.
- **Envio em segundo plano**: enquanto há envios, o app roda como serviço em primeiro plano e mostra o progresso numa notificação (e avisa quando termina). Pede a permissão de notificações na primeira vez.
- **Downloads**: baixar no site salva em Downloads com notificação de progresso; ao terminar, tocar na notificação abre o arquivo ("Abrir com…").
- Barras do sistema e ícone seguem as cores e o tema do site.
- Aceita certificados instalados pelo usuário (certificado raiz de uma CA da casa).

Requer a interface do servidor na versão 2.0.2-lab90 ou mais nova (a barra de compartilhamento e a ponte `LabNative` vêm dela).

## Compilar (Docker, sem instalar o Android SDK)

```bash
cd android-lab && npm install && npx cap sync android
docker run --rm -v "$PWD":/work -w /work/android -e HOME=/tmp/h -e ANDROID_SDK_HOME=/tmp/h -e ANDROID_USER_HOME=/tmp/h/.android \
  -e GRADLE_OPTS=-Duser.home=/tmp/h mingc/android-build-box:latest ./gradlew assembleDebug --no-daemon
# APK: android/app/build/outputs/apk/debug/app-debug.apk
```

O APK de depuração é assinado com a chave de depuração do ambiente de compilação; para atualizar um app já instalado, use sempre a mesma chave.
