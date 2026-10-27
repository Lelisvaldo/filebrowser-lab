<script setup>
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as authApi from "@/api/auth";
import * as filesApi from "@/api/files";
import { hideStatus, app, page, toast, toastError } from "@/store/app";
import { askText, askConfirm, showInfo } from "@/store/dialogs";
import { passwordHeaders } from "@/lib/confirmPassword";
import { date } from "@/lib/format";
import { cfg } from "@/lib/config";
import QrCode from "@/components/QrCode.vue";
import { openShare } from "@/store/shareDlg";

const route = useRoute();
const router = useRouter();
const origin = window.location.origin;
const tab = computed(() => route.params.tab || "perfil");
const tabs = [["perfil", "Perfil"], ["seguranca", "Segurança"], ["chaves", "Chaves de API"], ["links", "Links compartilhados"]];
onMounted(() => { hideStatus(); page.title = "Minha conta"; });

// senha
const cur = ref(""); const n1 = ref(""); const n2 = ref(""); const busy = ref(false);
async function changePassword() {
  if (n1.value.length < (cfg.passwordMinLength || 5)) return toast(`A nova senha precisa ter ao menos ${cfg.passwordMinLength || 5} caracteres.`, "err");
  if (n1.value !== n2.value) return toast("As duas senhas novas não são iguais.", "err");
  busy.value = true;
  try {
    await authApi.updateUser(app.user.username, ["password"], { password: n1.value }, { "X-Password": encodeURIComponent(cur.value) });
    toast("Senha alterada.");
    cur.value = n1.value = n2.value = "";
  } catch (e) { toastError(e); } finally { busy.value = false; }
}

// chaves de acesso (passkey)
const pkOn = !!cfg.passkeyAvailable && authApi.passkeySupported();
const pkPass = ref("");
const pks = ref([]);
async function pkLoad() { if (pkOn) { try { pks.value = await authApi.passkeyList(); } catch (_e) { pks.value = []; } } }
onMounted(pkLoad);
async function pkAdd() {
  const name = await askText({ title: "Nova chave de acesso", label: "Nome (ex.: celular, notebook)", value: "Meu aparelho", confirm: "Continuar" });
  if (!name) return;
  try { await authApi.passkeyRegister(name, pkPass.value); toast("Chave de acesso cadastrada."); pkPass.value = ""; await pkLoad(); } catch (e) { toastError(e); }
}
async function pkDel(c) {
  if (!(await askConfirm({ title: "Remover chave de acesso", message: `"${c.name}" deixará de funcionar.`, confirm: "Remover", danger: true }))) return;
  try { await authApi.passkeyDelete(c.id, pkPass.value); toast("Removida."); await pkLoad(); } catch (e) { toastError(e); }
}

// 2FA
const otpUrl = ref(""); const otpCode = ref(""); const otpPass = ref("");
async function otpStart() {
  try { const r = await authApi.otpGenerate(app.user.username, otpPass.value); otpUrl.value = r.url; } catch (e) { toastError(e); }
}
async function otpFinish() {
  try {
    await authApi.otpVerify(app.user.username, otpPass.value, otpCode.value);
    toast("Verificação em duas etapas ativada.");
    otpUrl.value = otpCode.value = otpPass.value = "";
    app.user = await authApi.me();
  } catch (e) { toastError(e); }
}

// chaves
const keys = ref(null);
async function loadKeys() { try { const r = await authApi.listApiKeys(); keys.value = Array.isArray(r) ? r : Object.values(r || {}); } catch (e) { keys.value = []; if (e.status !== 404) toastError(e); } }
async function newKey() {
  const name = await askText({ title: "Nova chave de API", label: "Nome da chave", confirm: "Criar" });
  if (!name) return;
  try { await authApi.createApiKey({ name, days: 30 }); toast("Chave criada."); await loadKeys(); } catch (e) { toastError(e); }
}
async function delKey(k) {
  if (!(await askConfirm({ title: "Apagar chave", message: `A chave "${k.name}" deixará de funcionar.`, confirm: "Apagar", danger: true }))) return;
  try { await authApi.deleteApiKey(k.name); await loadKeys(); } catch (e) { toastError(e); }
}
async function showKey(k) { await showInfo({ title: k.name, message: k.token || "(sem token)", confirm: "Fechar" }); }

// links
const links = ref(null);
async function loadLinks() { try { links.value = await filesApi.shares(); } catch (e) { links.value = []; toastError(e); } }
async function delLink(l) {
  if (!(await askConfirm({ title: "Remover link", message: "Quem tem o link perderá o acesso.", confirm: "Remover", danger: true }))) return;
  try { await filesApi.deleteShare(l.hash); await loadLinks(); } catch (e) { toastError(e); }
}
async function editLink(l) { await openShare(l.source, { name: String(l.path).split("/").filter(Boolean).pop() || l.source, path: l.path, type: l.isDir ? "directory" : "file" }, l); await loadLinks(); }
async function copy(url) { try { await navigator.clipboard.writeText(url); toast("Link copiado."); } catch (_e) { await showInfo({ title: "Link", message: url, confirm: "Fechar" }); } }

onMounted(() => { void loadKeys(); void loadLinks(); });
</script>

<template>
  <div class="page">
    <div class="wrap">
      <div class="phead"><h1>Minha conta</h1><div class="crumbs"><span>{{ app.user.username }}</span></div></div>
      <div class="tabs" role="tablist"><button v-for="[id, l] in tabs" :key="id" role="tab" :aria-selected="tab === id" @click="router.push('/ajustes/' + id)">{{ l }}</button></div>

      <section v-if="tab === 'perfil'" class="card">
        <h2>Trocar senha</h2>
        <form class="form" @submit.prevent="changePassword">
          <label>Senha atual<input v-model="cur" type="password" autocomplete="current-password" /></label>
          <label>Nova senha<input v-model="n1" type="password" autocomplete="new-password" /></label>
          <label>Repita a nova senha<input v-model="n2" type="password" autocomplete="new-password" /></label>
          <button class="btn pri" :disabled="busy || !cur || !n1">Alterar senha</button>
        </form>
      </section>

      <section v-else-if="tab === 'seguranca'" class="card">
        <h2>Verificação em duas etapas (aplicativo autenticador)</h2>
        <p class="meta">Situação: <b>{{ app.user.otpEnabled ? "ativada" : "desativada" }}</b>.</p>
        <div class="form">
          <label>Sua senha<input v-model="otpPass" type="password" autocomplete="current-password" /></label>
          <button class="btn" :disabled="!otpPass" @click="otpStart"><Icon name="shield" /><span>{{ app.user.otpEnabled ? "Gerar novo código" : "Ativar" }}</span></button>
        </div>
        <div v-if="otpUrl" class="form" style="margin-top: 14px">
          <p class="meta">Leia o QR code no aplicativo autenticador (Google Authenticator, Aegis, etc.). Se preferir, copie o link abaixo.</p>
          <QrCode :text="otpUrl" />
          <textarea readonly rows="3" class="mono" :value="otpUrl" @focus="$event.target.select()"></textarea>
          <label>Código de 6 dígitos<input v-model="otpCode" inputmode="numeric" autocomplete="one-time-code" /></label>
          <button class="btn pri" :disabled="!otpCode" @click="otpFinish">Confirmar</button>
        </div>
        <div v-if="pkOn" class="pk-sec">
        <h2>Chave de acesso (passkey)</h2>
        <p class="meta">Segundo fator no login: biometria do aparelho ou chave de segurança, depois da senha.</p>
        <div v-if="pks.length" class="tbl">
          <div v-for="c in pks" :key="c.id" class="trow"><div class="main"><b>{{ c.name }}</b><small>criada {{ c.createdAt ? date(new Date(c.createdAt * 1000).toISOString()) : "" }}</small></div><button class="ib" title="Remover" aria-label="Remover chave" @click="pkDel(c)"><Icon name="trash" /></button></div>
        </div>
        <div class="form"><label v-if="pks.length">Sua senha (para alterar)<input v-model="pkPass" type="password" autocomplete="current-password" /></label>
          <button class="btn" type="button" @click="pkAdd"><Icon name="key" /><span>Cadastrar chave de acesso</span></button></div>
        </div>
      </section>

      <section v-else-if="tab === 'chaves'" class="card">
        <div class="cardhead"><h2>Chaves de API</h2><button class="btn" @click="newKey" title="Nova chave" aria-label="Nova chave"><Icon name="plus" /><span class="lbl">Nova chave</span></button></div>
        <div v-if="keys === null" class="vazio">Carregando…</div>
        <div v-else-if="!keys.length" class="vazio">Nenhuma chave criada.</div>
        <div v-else class="tbl">
          <div v-for="k in keys" :key="k.name" class="trow">
            <div class="main"><b>{{ k.name }}</b><small v-if="k.expiresAt">expira em {{ date(new Date(k.expiresAt * 1000).toISOString()) }}</small></div>
            <button class="ib" title="Ver chave" aria-label="Ver chave" @click="showKey(k)"><Icon name="eye" /></button>
            <button class="ib" title="Apagar" aria-label="Apagar" @click="delKey(k)"><Icon name="trash" /></button>
          </div>
        </div>
      </section>

      <section v-else class="card">
        <h2>Links compartilhados</h2>
        <div v-if="links === null" class="vazio">Carregando…</div>
        <div v-else-if="!links.length" class="vazio">Você ainda não compartilhou nada. Use “Compartilhar” em um arquivo ou pasta.</div>
        <div v-else class="tbl">
          <div v-for="l in links" :key="l.hash" class="trow">
            <div class="main"><b>{{ String(l.path).split("/").filter(Boolean).pop() || l.source }}</b><small>{{ l.source }} · {{ l.expire ? "expira " + date(new Date(l.expire * 1000).toISOString()) : "sem validade" }}</small></div>
            <button class="ib" title="Copiar link" aria-label="Copiar link" @click="copy(l.shareURL || origin + '/public/share/' + l.hash)"><Icon name="link" /></button>
            <button class="ib" title="Editar opções" aria-label="Editar opções" @click="editLink(l)"><Icon name="edit" /></button>
            <button class="ib" title="Remover" aria-label="Remover" @click="delLink(l)"><Icon name="trash" /></button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
