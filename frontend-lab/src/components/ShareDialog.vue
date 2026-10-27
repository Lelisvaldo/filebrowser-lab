<script setup>
// Compartilhar com link: validade, senha, limite de downloads, aparência e permissões.
import { ref, computed, watch } from "vue";
import Icon from "./Icon.vue";
import * as filesApi from "@/api/files";
import { shareDlg, closeShare } from "@/store/shareDlg";
import { toast, toastError } from "@/store/app";
import { askConfirm } from "@/store/dialogs";

const f = ref({});
const busy = ref(false);
const isDir = computed(() => shareDlg.item && shareDlg.item.type === "directory");
const url = computed(() => (shareDlg.link ? shareDlg.link.shareURL || `${location.origin}/public/share/${shareDlg.link.hash}` : ""));
const editing = computed(() => !!shareDlg.link);

watch(() => shareDlg.open, (o) => {
  if (!o) return;
  const l = shareDlg.link || {};
  f.value = {
    mode: l.expire ? "days" : shareDlg.link ? "never" : "days",
    num: shareDlg.link ? "" : "7",
    password: "", clearPw: false, hasPw: !!l.hasPassword,
    downloadsLimit: l.downloadsLimit || "",
    title: l.title || "",
    theme: l.enforceDarkLightMode || "",
    allowCreate: !!l.allowCreate,
    disableDownload: !!l.disableDownload,
  };
  if (l.expire) { const left = Math.max(1, Math.round((l.expire * 1000 - Date.now()) / 3600000)); f.value.mode = left > 72 ? "days" : "hours"; f.value.num = String(left > 72 ? Math.round(left / 24) : left); }
}, { immediate: true });

function body() {
  const v = f.value;
  const b = { enforceDarkLightMode: v.theme || "", title: v.title, allowCreate: isDir.value && v.allowCreate, disableDownload: v.disableDownload, downloadsLimit: Number(v.downloadsLimit) || 0 };
  if (v.mode !== "never" && Number(v.num) > 0) { b.expires = String(Math.floor(Number(v.num))); b.unit = v.mode; }
  if (v.password) b.password = v.password; else if (v.clearPw) b.password = "";
  return b;
}
async function save() {
  busy.value = true;
  try {
    let link;
    if (editing.value) link = await filesApi.updateShare({ hash: shareDlg.link.hash, path: shareDlg.link.path, ...body() });
    else link = await filesApi.createShare({ path: shareDlg.item.path, source: shareDlg.source, ...body() });
    shareDlg.link = link;
    toast(editing.value ? "Link atualizado." : "Link criado.");
    await copy();
  } catch (e) { toastError(e); } finally { busy.value = false; }
}
async function copy() { try { await navigator.clipboard.writeText(url.value); toast("Link copiado."); } catch (_e) { /* o endereço fica na tela */ } }
async function remove() {
  if (!(await askConfirm({ title: "Remover link", message: "Quem tem o link perderá o acesso.", confirm: "Remover", danger: true }))) return;
  try { await filesApi.deleteShare(shareDlg.link.hash); toast("Link removido."); closeShare(null); } catch (e) { toastError(e); }
}
</script>

<template>
  <div v-if="shareDlg.open" class="fs-ov" @click.self="closeShare(shareDlg.link)" @keydown.esc="closeShare(shareDlg.link)">
    <div class="fs-box sh-box" role="dialog" aria-modal="true" aria-label="Compartilhar com link">
      <div class="fs-top"><span class="fs-prev"><Icon name="share" /></span><div><b>Compartilhar com link</b><small>{{ shareDlg.item && shareDlg.item.name }}</small></div></div>

      <div v-if="editing" class="sh-url"><input readonly :value="url" aria-label="Endereço do link" @focus="$event.target.select()" /><button class="btn" type="button" @click="copy"><Icon name="copy" /><span>Copiar</span></button></div>

      <div class="sh-row"><span class="fs-lbl">Vale por</span>
        <select v-model="f.mode" aria-label="Validade"><option value="never">Sem validade</option><option value="minutes">minutos</option><option value="hours">horas</option><option value="days">dias</option></select>
        <input v-if="f.mode !== 'never'" v-model="f.num" type="number" min="1" inputmode="numeric" aria-label="Quantidade" /></div>
      <label class="sh-row"><span class="fs-lbl">Senha</span><input v-model="f.password" type="password" autocomplete="new-password" :placeholder="f.hasPw ? 'Já tem senha (deixe vazio para manter)' : 'Sem senha'" />
        <span v-if="f.hasPw" class="sh-chk"><input id="shcl" v-model="f.clearPw" type="checkbox" /><label for="shcl">Tirar a senha</label></span></label>
      <label class="sh-row"><span class="fs-lbl">Limite de downloads</span><input v-model="f.downloadsLimit" type="number" min="0" inputmode="numeric" placeholder="Sem limite" /></label>
      <label class="sh-row"><span class="fs-lbl">Título da página</span><input v-model="f.title" type="text" placeholder="Nome do item" /></label>
      <label class="sh-row"><span class="fs-lbl">Tema</span><select v-model="f.theme"><option value="">Do visitante</option><option value="dark">Sempre escuro</option><option value="light">Sempre claro</option></select></label>
      <label v-if="isDir" class="sh-chk2"><input v-model="f.allowCreate" type="checkbox" /> Permitir que enviem arquivos para esta pasta</label>
      <label class="sh-chk2"><input v-model="f.disableDownload" type="checkbox" /> Só visualizar (sem botão de baixar)</label>

      <div class="fs-act">
        <button v-if="editing" class="btn danger" type="button" @click="remove"><Icon name="trash" /><span>Remover link</span></button>
        <span class="sp"></span>
        <button class="btn" type="button" @click="closeShare(shareDlg.link)">Fechar</button>
        <button class="btn pri" type="button" :disabled="busy" @click="save">{{ editing ? "Salvar" : "Criar link" }}</button>
      </div>
    </div>
  </div>
</template>
