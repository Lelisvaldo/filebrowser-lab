<script setup>
import { watch, ref, nextTick } from "vue";
import { useRouter } from "vue-router";
import Icon from "./Icon.vue";
import { app, ui, setUi, doLogout, isAdmin, startUsbPolling } from "@/store/app";

const router = useRouter();
const closeBtn = ref(null);
const close = () => setUi({ settingsOpen: false });

watch(() => ui.settingsOpen, async (o) => { if (o) { await nextTick(); closeBtn.value?.focus(); } });

async function sair() {
  close();
  await doLogout();
  router.replace("/login");
}
function go(path) { close(); router.push(path); }
function freq(v) { setUi({ usbRefresh: v }); startUsbPolling(); }
</script>

<template>
  <div class="cfg-scrim" @click="close"></div>
  <aside class="cfg" aria-label="Configurações" role="dialog" aria-modal="true" @keydown.esc="close">
    <header>
      <b>Configurações</b>
      <button ref="closeBtn" class="ib" aria-label="Fechar" @click="close"><Icon name="x" /></button>
    </header>
    <div class="body">
      <div class="conta">
        <span class="ico"><Icon name="person" /></span>
        <div class="l"><b>{{ app.user?.username }}</b><span>{{ isAdmin() ? "Administrador" : "Usuário" }}</span></div>
        <button class="btn" type="button" style="min-height: 40px" @click="sair" title="Sair" aria-label="Sair"><Icon name="signout" /><span class="lbl">Sair</span></button>
      </div>

      <div class="sec">Modo de cor</div>
      <div class="temas">
        <button class="tema" :aria-pressed="ui.theme === 'auto'" @click="setUi({ theme: 'auto' })"><span class="sw2"><i style="background: #e6e9ed"></i><i style="background: #1f0c4c"></i></span>Automático</button>
        <button class="tema" :aria-pressed="ui.theme === 'light'" @click="setUi({ theme: 'light' })"><span class="sw2"><i style="background: #e6e9ed"></i><i style="background: #0b62b8"></i></span>Claro</button>
        <button class="tema" :aria-pressed="ui.theme === 'dark'" @click="setUi({ theme: 'dark' })"><span class="sw2"><i style="background: #1f0c4c"></i><i style="background: #01ced3"></i></span>Escuro</button>
      </div>
      <div class="meta" style="padding: 6px 0 8px">Automático segue o claro ou escuro do aparelho. A escolha fica salva neste navegador.</div>

      <div class="sec">Aparência</div>
      <div class="opt">
        <div class="l"><b>Densidade</b><span>Linhas mais baixas mostram mais arquivos.</span></div>
        <div class="seg">
          <button :aria-pressed="ui.density === 'comfortable'" @click="setUi({ density: 'comfortable' })">Confortável</button>
          <button :aria-pressed="ui.density === 'compact'" @click="setUi({ density: 'compact' })">Compacta</button>
        </div>
      </div>
      <div class="opt">
        <div class="l"><b>Menu lateral recolhido</b><span>Só ícones, para ganhar espaço.</span></div>
        <button class="sw" role="switch" :aria-checked="ui.rail" aria-label="Menu lateral recolhido" @click="setUi({ rail: !ui.rail })"></button>
      </div>

      <template v-if="app.usb !== null">
        <div class="sec">Pendrives</div>
        <div class="opt">
          <div class="l"><b>Atualizar a lista</b><span>De quanto em quanto tempo procurar pendrives.</span></div>
          <div class="seg">
            <button :aria-pressed="ui.usbRefresh === 3" @click="freq(3)">3 s</button>
            <button :aria-pressed="ui.usbRefresh === 10" @click="freq(10)">10 s</button>
            <button :aria-pressed="ui.usbRefresh === 0" @click="freq(0)">Manual</button>
          </div>
        </div>
      </template>

      <div class="sec">Mais</div>
      <button class="linkcard" @click="go('/ajustes')"><b>Minha conta</b><small>Senha, 2FA, chaves de API e compartilhamentos</small></button>
      <button v-if="isAdmin()" class="linkcard" @click="go('/admin')"><b>Administração</b><small>Usuários, sistema e atividade</small></button>
    </div>
  </aside>
</template>
