<script setup>
import { ref, computed, onMounted, watch, nextTick } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as authApi from "@/api/auth";
import * as filesApi from "@/api/files";
import * as adminApi from "@/api/admin";
import * as cloudApi from "@/api/cloud";
import { hideStatus, app, page, isAdmin, toast, toastError } from "@/store/app";
import { askText, askConfirm } from "@/store/dialogs";
import { passwordHeaders } from "@/lib/confirmPassword";
import { bytes, date, pct } from "@/lib/format";
import { cfg } from "@/lib/config";

const route = useRoute();
const router = useRouter();
const tab = computed(() => route.params.tab || "usuarios");
const tabs = [["usuarios", "Usuários"], ["acessos", "Regras de acesso"], ["sistema", "Sistema"], ["servicos", "Conectar serviços"], ["atividade", "Atividade"]];

// ---- usuários ----
const users = ref(null);
const editing = ref(null); // usuário em edição ou novo
const isNew = ref(false);
const sourceNames = computed(() => Object.keys(app.sources));
async function loadUsers() { try { users.value = await authApi.listUsers(); } catch (e) { users.value = []; toastError(e); } }

function newUser() {
  isNew.value = true;
  editing.value = { username: "", password: "", admin: false, share: true, api: false, scopes: sourceNames.value.map((n) => ({ name: n, on: false, scope: "/", edit: false })) };
}
function editUser(u) {
  isNew.value = false;
  editing.value = {
    username: u.username, admin: !!u.permissions?.admin, share: !!u.permissions?.share, api: !!u.permissions?.api, password: "",
    scopes: sourceNames.value.map((n) => {
      const sc = (u.scopes || []).find((s) => s.name === n);
      return { name: n, on: !!sc, scope: sc?.scope || "/", edit: !!sc?.permissions?.modify };
    }),
  };
}
function packScopes(e) {
  return e.scopes.filter((s) => s.on).map((s) => ({
    name: s.name, scope: s.scope || "/",
    permissions: { view: true, download: true, create: s.edit, modify: s.edit, delete: s.edit },
  }));
}
async function saveUser() {
  const e = editing.value;
  const headers = await passwordHeaders("Confirme a sua senha para alterar usuários.");
  if (!headers) return;
  const permissions = { admin: e.admin, share: e.share, api: e.api, realtime: false };
  try {
    if (isNew.value) {
      if (!e.username || !e.password) return toast("Informe usuário e senha.", "err");
      await authApi.createUser({ username: e.username, password: e.password, loginMethod: "password", permissions, scopes: packScopes(e) }, headers);
      // o servidor aplica as pastas padrão ao criar; em seguida ajustamos para as escolhidas aqui
      await authApi.updateUser(e.username, ["scopes", "permissions"], { permissions, scopes: packScopes(e) }, headers);
      toast("Usuário criado.");
    } else {
      const which = ["permissions", "scopes"];
      const data = { permissions, scopes: packScopes(e) };
      if (e.password) { which.push("password"); data.password = e.password; }
      await authApi.updateUser(e.username, which, data, headers);
      toast("Usuário atualizado.");
    }
    editing.value = null;
    await loadUsers();
  } catch (err) { toastError(err); }
}
async function delUser(u) {
  if (u.username === app.user.username) return toast("Você não pode apagar o próprio usuário.", "err");
  if (!(await askConfirm({ title: "Apagar usuário", message: `O usuário "${u.username}" será removido.`, confirm: "Apagar", danger: true }))) return;
  const headers = await passwordHeaders();
  if (!headers) return;
  try { await authApi.deleteUser(u.username, headers); toast("Usuário apagado."); await loadUsers(); } catch (e) { toastError(e); }
}

// ---- regras de acesso ----
const acSource = ref("");
const acPath = ref("/");
const acRule = ref(null);
const acForm = ref({ allow: false, ruleCategory: "user", value: "" });
async function loadRule() {
  if (!acSource.value) acSource.value = sourceNames.value[0] || "";
  if (!acSource.value) return;
  try { acRule.value = await adminApi.accessRule(acSource.value, acPath.value || "/"); } catch (e) { acRule.value = null; toastError(e); }
}
async function addRule() {
  if (!acForm.value.value.trim()) return;
  try {
    await adminApi.accessAdd(acSource.value, acPath.value || "/", { allow: acForm.value.allow, ruleCategory: acForm.value.ruleCategory, value: acForm.value.value.trim() });
    acForm.value.value = "";
    toast("Regra salva.");
    await loadRule();
  } catch (e) { toastError(e); }
}
async function delRule(allow, ruleCategory, value) {
  try { await adminApi.accessDel(acSource.value, acPath.value || "/", { allow, ruleCategory, value }); await loadRule(); } catch (e) { toastError(e); }
}
const rules = computed(() => {
  const r = acRule.value || {};
  const out = [];
  for (const [kind, set] of [["Negar", r.deny], ["Permitir", r.allow]]) {
    for (const u of set?.users || []) out.push({ kind, allow: kind === "Permitir", cat: "user", catLabel: "usuário", value: u });
    for (const g of set?.groups || []) out.push({ kind, allow: kind === "Permitir", cat: "group", catLabel: "grupo", value: g });
  }
  return out;
});

// ---- sistema ----
const sources = computed(() => Object.values(app.sources));

// ---- atividade ----
const act = ref(null);
const evLabel = { upload: "Envio", download: "Download", delete: "Apagou", bulkDelete: "Apagou", move: "Moveu", copy: "Copiou", rename: "Renomeou", login: "Entrou", logout: "Saiu", create: "Criou", edit: "Editou", shareCreate: "Criou link", shareDelete: "Removeu link" };
async function loadAct() { try { act.value = (await adminApi.activity({ limit: 50 })).items || []; } catch (e) { act.value = []; toastError(e); } }
const when = (s) => {
  const d = date(new Date(s * 1000).toISOString());
  return /\d:\d\d/.test(d) ? d : d + " " + new Date(s * 1000).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
};

// ---- conectar serviços (nuvens) ----
const cloud = ref(null);
const cloudForm = ref(null);
const cloudApp = ref(null); // { id, name, clientId, clientSecret } | { id, name, user, pass, senha: true }
const cloudHue = { onedrive: "blue", drive: "green", dropbox: "cyan", mega: "red" };
async function loadCloud() {
  try { cloud.value = await cloudApi.list(); } catch (e) { cloud.value = { items: [], callback: "" }; if (e.status !== 404) toastError(e); }
}
async function connectCloud(it) {
  if (it.kind === "senha") { cloudApp.value = { id: it.id, name: it.name, user: "", pass: "", senha: true }; focusForm(); return; }
  try { const r = await cloudApi.start(it.id); location.href = r.url; } catch (e) {
    if (e.status === 412) { openApp(it); toast("Cadastre primeiro o aplicativo do " + it.name + "."); } else toastError(e);
  }
}
function openApp(it) { cloudApp.value = { id: it.id, name: it.name, clientId: "", clientSecret: "" }; focusForm(); }
function focusForm() { void nextTick(() => cloudForm.value?.scrollIntoView({ behavior: "smooth", block: "nearest" })); }
async function saveCloudApp() {
  const a = cloudApp.value;
  try {
    if (a.senha) { await cloudApi.passwordLogin(a.id, a.user, a.pass); toast(a.name + " conectado."); }
    else { await cloudApi.saveApp(a.id, a.clientId, a.clientSecret); toast("Aplicativo guardado. Agora toque em entrar."); }
    cloudApp.value = null; await loadCloud();
  } catch (e) { toastError(e); }
}
async function dropCloud(it) {
  if (!(await askConfirm({ title: "Desconectar " + it.name + "?", message: "O token é apagado do servidor. Seus arquivos na nuvem não são tocados.", confirm: "Desconectar", danger: true }))) return;
  try { await cloudApi.disconnect(it.id); await loadCloud(); } catch (e) { toastError(e); }
}
async function toggleRO(it) {
  if (it.readOnly && !(await askConfirm({ title: "Liberar escrita no " + it.name + "?", message: "Será possível enviar, mover e apagar arquivos na nuvem pelo filebrowser.", confirm: "Liberar", danger: true }))) return;
  try { await cloudApi.setReadOnly(it.id, !it.readOnly); await loadCloud(); } catch (e) { toastError(e); }
}
function cloudReturn() {
  const q = route.query;
  if (!q.nuvem) return;
  if (q.nuvem === "ok") toast("Nuvem conectada."); else toastError(new Error(String(q.msg || "não foi possível conectar")));
  void router.replace("/admin/servicos");
}

onMounted(() => { hideStatus(); page.title = "Administração"; if (!isAdmin()) router.replace("/"); });
watch(tab, (t) => { if (t === "usuarios") void loadUsers(); if (t === "atividade") void loadAct(); if (t === "acessos") void loadRule(); if (t === "servicos") { void loadCloud(); cloudReturn(); } }, { immediate: true });
</script>

<template>
  <div class="page">
    <div class="wrap wide">
      <div class="phead"><div class="ph-t hue h-violet"><span class="ph-ico"><Icon name="gear" /></span><h1>Administração</h1></div><div class="crumbs"><span>{{ cfg.version }}</span></div></div>
      <div class="tabs" role="tablist"><button v-for="[id, l] in tabs" :key="id" role="tab" :aria-selected="tab === id" @click="router.push('/admin/' + id)">{{ l }}</button></div>

      <section v-if="tab === 'usuarios'" class="card">
        <div class="cardhead"><h2>Usuários</h2><button class="btn" @click="newUser" title="Novo usuário" aria-label="Novo usuário"><Icon name="plus" /><span class="lbl">Novo usuário</span></button></div>
        <div v-if="users === null" class="vazio">Carregando…</div>
        <div v-else class="tbl">
          <div v-for="u in users" :key="u.username" class="trow">
            <span class="ico sm"><Icon name="person" /></span>
            <div class="main"><b>{{ u.username }}</b><small>{{ u.permissions?.admin ? "Administrador" : "Usuário" }} · {{ (u.scopes || []).map((s) => s.name).join(", ") || "sem pastas" }}</small></div>
            <button class="ib" title="Editar" aria-label="Editar" @click="editUser(u)"><Icon name="edit" /></button>
            <button class="ib" title="Apagar" aria-label="Apagar" @click="delUser(u)"><Icon name="trash" /></button>
          </div>
        </div>

        <form v-if="editing" class="form userform" @submit.prevent="saveUser">
          <h2>{{ isNew ? "Novo usuário" : "Editar " + editing.username }}</h2>
          <label v-if="isNew">Usuário<input v-model="editing.username" autocomplete="off" /></label>
          <label>{{ isNew ? "Senha" : "Nova senha (deixe vazio para manter)" }}<input v-model="editing.password" type="password" autocomplete="new-password" /></label>
          <div class="opt"><div class="l"><b>Administrador</b><span>Acesso a tudo, inclusive usuários.</span></div><button type="button" class="sw" role="switch" :aria-checked="editing.admin" @click="editing.admin = !editing.admin"></button></div>
          <div class="opt"><div class="l"><b>Pode compartilhar links</b></div><button type="button" class="sw" role="switch" :aria-checked="editing.share" @click="editing.share = !editing.share"></button></div>
          <div class="opt"><div class="l"><b>Pode usar chaves de API</b></div><button type="button" class="sw" role="switch" :aria-checked="editing.api" @click="editing.api = !editing.api"></button></div>
          <div class="sec">Pastas liberadas</div>
          <div v-for="s in editing.scopes" :key="s.name" class="scoperow">
            <label class="chkl"><input v-model="s.on" type="checkbox" /> {{ s.name }}</label>
            <input v-model="s.scope" class="scopein" :disabled="!s.on" aria-label="Subpasta" placeholder="/" />
            <label class="chkl"><input v-model="s.edit" type="checkbox" :disabled="!s.on" /> Pode alterar</label>
          </div>
          <div class="dlg-b"><button type="button" class="btn" @click="editing = null">Cancelar</button><button class="btn pri">Salvar</button></div>
        </form>
      </section>

      <section v-else-if="tab === 'acessos'" class="card">
        <h2>Regras de acesso por pasta</h2>
        <p class="meta">Bloqueie ou libere uma pasta para um usuário ou grupo, além do que já vale pelas pastas liberadas de cada usuário.</p>
        <div class="fillrow">
          <label class="fld">Fonte<select v-model="acSource" class="scopein" @change="loadRule"><option v-for="n in sourceNames" :key="n" :value="n">{{ n }}</option></select></label>
          <label class="fld grow">Pasta<input v-model="acPath" placeholder="/" @keydown.enter.prevent="loadRule" /></label>
          <button class="btn" type="button" @click="loadRule" title="Ver" aria-label="Ver"><Icon name="refresh" /><span class="lbl">Ver</span></button>
        </div>
        <div v-if="acRule && !acRule.pathExists && acPath !== '/'" class="meta" style="margin: 8px 0">Essa pasta não existe na fonte (a regra pode ser criada mesmo assim).</div>
        <div v-if="acRule && acRule.denyAll" class="meta" style="margin: 8px 0">Todos negados nesta pasta.</div>
        <div class="tbl" style="margin-top: 10px">
          <div v-for="r in rules" :key="r.kind + r.cat + r.value" class="trow">
            <span class="chip" :class="{ off: !r.allow }" style="margin: 0">{{ r.kind }}</span>
            <div class="main"><b>{{ r.value }}</b><small>{{ r.catLabel }}</small></div>
            <button class="ib" title="Remover regra" aria-label="Remover regra" @click="delRule(r.allow, r.cat, r.value)"><Icon name="trash" /></button>
          </div>
          <div v-if="!rules.length" class="vazio" style="border: 0">Sem regras nesta pasta.</div>
        </div>
        <form class="userform wide" @submit.prevent="addRule">
          <h2>Nova regra</h2>
          <div class="fillrow">
            <select v-model="acForm.allow" class="scopein" aria-label="Ação"><option :value="false">Negar</option><option :value="true">Permitir</option></select>
            <select v-model="acForm.ruleCategory" class="scopein" aria-label="Tipo"><option value="user">usuário</option><option value="group">grupo</option></select>
            <input v-model="acForm.value" class="fld grow ctl" placeholder="nome do usuário ou grupo" aria-label="Nome do usuário ou grupo" />
            <button class="btn pri" :disabled="!acForm.value.trim()">Adicionar</button>
          </div>
        </form>
      </section>

      <section v-else-if="tab === 'sistema'" class="card">
        <h2>Pastas e discos</h2>
        <div class="tbl">
          <div v-for="s in sources" :key="s.name" class="trow">
            <div class="main"><b>{{ s.name }}</b><small>{{ s.status }} · {{ s.numFiles }} arquivos · {{ s.numDirs }} pastas · {{ bytes(s.used) }} de {{ bytes(s.total) }} ({{ pct(s.used, s.total) }}%)</small></div>
          </div>
        </div>
        <h2 style="margin-top: 24px">Versão</h2>
        <p class="meta">filebrowser-lab {{ cfg.version }} · commit {{ cfg.commitSHA }} · base Filebrowser Quantum</p>
        <p class="meta">Métodos de entrada: senha {{ cfg.passwordAvailable ? "ligado" : "desligado" }}, SSO {{ cfg.oidcAvailable ? "ligado" : "desligado" }}, proxy {{ cfg.proxyAvailable ? "ligado" : "desligado" }}, LDAP {{ cfg.ldapAvailable ? "ligado" : "desligado" }}.</p>
        <p class="meta">A configuração das pastas e dos métodos de entrada fica no arquivo <code>config.yaml</code> do servidor.</p>
      </section>

      <section v-else-if="tab === 'servicos'" class="card">
        <div class="cardhead"><h2>Nuvens</h2><button class="ib" aria-label="Atualizar" title="Atualizar" @click="loadCloud"><Icon name="refresh" /></button></div>
        <p class="meta">Cada nuvem aparece como uma pasta em <code>Nuvens</code> no menu lateral. Entre uma por vez; o token fica guardado só no servidor.</p>
        <div v-if="cloud === null" class="vazio">Carregando…</div>
        <div v-else-if="!cloud.items.length" class="vazio">As nuvens não estão ligadas neste servidor.</div>
        <div v-else class="tbl">
          <div v-for="it in cloud.items" :key="it.id" class="trow cl-row">
            <span :class="['ico', 'hue', 'h-' + cloudHue[it.id]]"><Icon name="cloud" /></span>
            <div class="main"><b>{{ it.name }}</b><small>{{ it.connected ? (it.account || "conta conectada") + (it.readOnly ? " · somente leitura" : " · leitura e escrita") : it.hasApp ? "Pronto para entrar" : "Aplicativo ainda não cadastrado" }}</small></div>
            <span :class="['chip', 'cl-chip', { off: !it.connected }]">{{ it.connected ? "Conectada" : it.hasApp ? "Desconectada" : "Sem app" }}</span>
            <button v-if="it.connected" class="ib" :aria-label="it.readOnly ? 'Liberar escrita' : 'Voltar a só leitura'" :title="it.readOnly ? 'Só leitura (toque para liberar escrita)' : 'Escrita liberada (toque para voltar a só leitura)'" @click="toggleRO(it)"><Icon :name="it.readOnly ? 'lock' : 'edit'" /></button>
            <button v-if="it.kind === 'oauth'" class="ib" aria-label="Cadastrar aplicativo" title="Aplicativo do provedor (ID e segredo)" @click="openApp(it)"><Icon name="key" /></button>
            <button v-if="it.connected" class="btn danger sm" :aria-label="'Desconectar ' + it.name" title="Desconectar" @click="dropCloud(it)"><Icon name="signout" /><span class="lbl">Desconectar</span></button>
            <button v-else :class="['btn', 'sm', { pri: it.hasApp }]" :aria-label="'Entrar no ' + it.name" title="Entrar" @click="connectCloud(it)"><Icon name="signin" /><span class="lbl">Entrar</span></button>
          </div>
        </div>
        <form v-if="cloudApp" ref="cloudForm" class="form cl-form" @submit.prevent="saveCloudApp">
          <div class="cl-fhead"><span :class="['ico', 'sm', 'hue', 'h-' + cloudHue[cloudApp.id]]"><Icon :name="cloudApp.senha ? 'person' : 'key'" /></span><h2>{{ cloudApp.senha ? "Entrar no " + cloudApp.name : "Aplicativo do " + cloudApp.name }}</h2></div>
          <template v-if="cloudApp.senha">
            <label>Email<input v-model="cloudApp.user" autocomplete="off" /></label>
            <label>Senha<input v-model="cloudApp.pass" type="password" autocomplete="new-password" /></label>
          </template>
          <template v-else>
            <p class="meta">Endereço de retorno a cadastrar no provedor: <code>{{ cloud?.callback }}</code></p>
            <label>ID do aplicativo<input v-model="cloudApp.clientId" autocomplete="off" /></label>
            <label>Segredo<input v-model="cloudApp.clientSecret" type="password" autocomplete="new-password" /></label>
          </template>
          <div class="dlg-b"><button type="button" class="btn" @click="cloudApp = null"><Icon name="back" /><span>Cancelar</span></button><button class="btn pri"><Icon name="check" /><span>Salvar</span></button></div>
        </form>
      </section>

      <section v-else class="card">
        <div class="cardhead"><h2>Atividade recente</h2><button class="ib" aria-label="Atualizar" title="Atualizar" @click="loadAct"><Icon name="refresh" /></button></div>
        <div v-if="act === null" class="vazio">Carregando…</div>
        <div v-else-if="!act.length" class="vazio">Sem atividade registrada.</div>
        <div v-else class="tbl">
          <div v-for="a in act" :key="a.id" class="trow">
            <div class="main"><b>{{ evLabel[a.eventType] || a.eventType }}</b><small>{{ a.username }} · {{ a.source }}{{ a.path }}{{ a.targetPath ? " → " + a.targetPath : "" }}</small></div>
            <small class="when">{{ when(a.createdAt) }}</small>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
