<script setup>
import { ref, watch, nextTick, computed } from "vue";
import Icon from "./Icon.vue";
import { dialog, closeDialog } from "@/store/dialogs";
import { app, userSources, hasUsbSource, usbSourceName } from "@/store/app";
import * as filesApi from "@/api/files";
import { crumbs, basename } from "@/lib/paths";
import { suggestName } from "@/lib/moveReason";

const input = ref(null);
const value = ref("");

// seletor de pasta
const src = ref("");
const cur = ref("/");
const dirs = ref([]);
const fileRows = ref([]); // arquivos da pasta de destino (só para mostrar o que já existe lá)
const loading = ref(false);
const sourceNames = computed(() => [...userSources().map((s) => s.name), ...(hasUsbSource() ? [usbSourceName] : [])]);

async function loadDirs() {
  loading.value = true;
  try {
    const d = await filesApi.list(src.value, cur.value);
    dirs.value = d.items.filter((i) => i.type === "directory");
    fileRows.value = d.items.filter((i) => i.type !== "directory");
  } catch (_e) {
    dirs.value = [];
    fileRows.value = [];
  } finally {
    loading.value = false;
  }
}

watch(() => dialog.open, async (o) => {
  if (!o) return;
  value.value = dialog.value || "";
  if (dialog.kind === "folder") {
    src.value = dialog.extra?.source || sourceNames.value[0] || "";
    cur.value = dialog.extra?.path || "/";
    await loadDirs();
  }
  await nextTick();
  if (input.value) {
    input.value.focus();
    if (dialog.kind === "text" && dialog.extra?.selectBase) {
      const n = value.value.lastIndexOf(".");
      input.value.setSelectionRange(0, n > 0 ? n : value.value.length);
    } else input.value.select?.();
  }
});

// itens que estão sendo movidos/copiados e já existem na pasta escolhida
const moving = computed(() => dialog.extra?.names || []);
const takenNames = computed(() => new Set([...dirs.value, ...fileRows.value].map((i) => i.name)));
const sameFolder = computed(() => dialog.extra?.source === src.value && String(dialog.extra?.path || "/").replace(/\/?$/, "/") === cur.value.replace(/\/?$/, "/"));
const clashes = computed(() => (sameFolder.value && dialog.extra?.verb === "Mover" ? [] : moving.value.filter((m) => takenNames.value.has(m.name))));
const clashSet = computed(() => new Set(clashes.value.map((m) => m.name)));
const clashNote = computed(() => clashes.value.map((m) => ({ from: m.name, to: suggestName(m.name, takenNames.value, m.dir) })));
const shownFiles = computed(() => fileRows.value.slice(0, 200));
const trail = computed(() => [{ name: src.value, path: "/" }, ...crumbs(cur.value)]);
const parentOf = (p) => { const t = p.replace(/\/+$/, ""); const i = t.lastIndexOf("/"); return i <= 0 ? "/" : t.slice(0, i + 1); };
function go(path) { cur.value = path; void loadDirs(); }
function pickSource(s) { src.value = s; cur.value = "/"; void loadDirs(); }
function ok() {
  if (dialog.kind === "text") {
    const v = value.value.trim();
    if (!v) return;
    closeDialog(v);
  } else if (dialog.kind === "folder") closeDialog({ source: src.value, path: cur.value });
  else closeDialog(true);
}
const cancel = () => closeDialog(dialog.kind === "confirm" || dialog.kind === "info" ? false : null);
</script>

<template>
  <div v-if="dialog.open" class="dlg-scrim" @mousedown.self="cancel" @keydown.esc="cancel">
    <form class="dlg" role="dialog" aria-modal="true" :aria-label="dialog.title" @submit.prevent="ok">
      <h2 class="dlg-t">{{ dialog.title }}</h2>
      <p v-if="dialog.message" class="dlg-m">{{ dialog.message }}</p>
      <template v-if="dialog.kind === 'text'">
        <label v-if="dialog.label" class="dlg-l" for="dlg-in">{{ dialog.label }}</label>
        <input id="dlg-in" ref="input" v-model="value" class="dlg-in" :type="dialog.extra?.password ? 'password' : 'text'" autocomplete="off" spellcheck="false" />
      </template>
      <template v-else-if="dialog.kind === 'folder'">
        <div class="pick-src">
          <button v-for="s in sourceNames" :key="s" type="button" class="chipb" :aria-pressed="s === src" @click="pickSource(s)">{{ s }}</button>
        </div>
        <div class="pick-dest"><Icon name="folderopen" /><span>Destino:</span>
          <span class="pd-c"><template v-for="(c, i) in trail" :key="c.path"><span v-if="i">/</span><b v-if="i === trail.length - 1">{{ c.name }}</b><button v-else type="button" @click="go(c.path)">{{ c.name }}</button></template></span></div>
        <div class="pick-list">
          <button v-if="cur !== '/'" type="button" class="pick-row pick-up" @click="go(parentOf(cur))"><Icon name="back" /><span>Voltar para a pasta de cima</span></button>
          <button v-for="d in dirs" :key="d.path" type="button" class="pick-row" :class="{ clash: clashSet.has(d.name) }" @click="go(d.path)">
            <Icon name="folder" /><span>{{ d.name }}</span><em v-if="clashSet.has(d.name)">já existe</em><Icon name="right" />
          </button>
          <div v-if="!dirs.length && !loading" class="pick-empty">Nenhuma subpasta aqui. Você pode usar esta pasta.</div>
          <div v-for="f in shownFiles" :key="f.path" class="pick-row pick-file" :class="{ clash: clashSet.has(f.name) }" :title="f.name"><Icon name="file" /><span>{{ f.name }}</span><em v-if="clashSet.has(f.name)">já existe</em></div>
          <div v-if="fileRows.length > shownFiles.length" class="pick-empty">+ {{ fileRows.length - shownFiles.length }} arquivos</div>
        </div>
        <div v-if="clashNote.length" class="pick-warn" role="alert"><Icon name="warning" /><span><b>Já existe nesta pasta:</b> {{ clashNote.map((c) => c.from).join(", ") }}. Nada será sobrescrito: o {{ (dialog.extra?.verb || "item").toLowerCase() === "mover" ? "movido" : "copiado" }} ficará como <b>{{ clashNote.map((c) => c.to).join(", ") }}</b>.</span></div>
      </template>
      <div class="dlg-b">
        <button v-if="dialog.kind !== 'info'" type="button" class="btn" @click="cancel">Cancelar</button>
        <button type="submit" class="btn pri" :class="{ danger: dialog.danger }" :ref="dialog.kind === 'confirm' || dialog.kind === 'info' ? 'input' : undefined">{{ dialog.kind === 'folder' ? (dialog.extra?.verb || 'Escolher') + ' nesta pasta' : dialog.confirm }}</button>
      </div>
    </form>
  </div>
</template>
