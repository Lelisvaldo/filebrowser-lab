<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import ItemMenu from "@/components/ItemMenu.vue";
import FileViewer from "@/components/FileViewer.vue";
import * as filesApi from "@/api/files";
import { app, ui, setUi, page, canIn, toast, toastError, refreshSources } from "@/store/app";
import { askText, askConfirm, pickFolder } from "@/store/dialogs";
import { enqueueUploads, onTransferDone, noteDownload } from "@/store/transfers";
import { kindOf, iconKey } from "@/lib/fileTypes";
import { crumbs, join, parent, routeFor } from "@/lib/paths";
import { bytes, date } from "@/lib/format";
import { filesFromDrop, filesFromInput } from "@/lib/useDrop";
import { shareItem } from "@/lib/share";

const route = useRoute();
const router = useRouter();

const source = computed(() => String(route.params.source || ""));
const rest = computed(() => String(route.params.rest || ""));
const isFile = computed(() => rest.value !== "" && !rest.value.endsWith("/"));
const fullPath = computed(() => "/" + rest.value);
const dirPath = computed(() => (isFile.value ? parent(fullPath.value) : fullPath.value));

const data = ref(null);
const loading = ref(true);
const error = ref("");
const sel = ref(new Set());
let lastIdx = -1;
const menu = ref(null);
const dragging = ref(false);
let dragDepth = 0;
const fileInput = ref(null);
const folderInput = ref(null);

const canCreate = computed(() => canIn(source.value, "create"));
const canModify = computed(() => canIn(source.value, "modify"));
const canDelete = computed(() => canIn(source.value, "delete"));
const canDownload = computed(() => canIn(source.value, "download"));
const folderName = computed(() => (dirPath.value === "/" ? source.value : dirPath.value.split("/").filter(Boolean).pop()));

const collator = new Intl.Collator("pt-BR", { numeric: true, sensitivity: "base" });
const items = computed(() => {
  const list = [...(data.value?.items || [])].filter((i) => !i.hidden || app.user?.showHidden);
  const dir = ui.sortAsc ? 1 : -1;
  list.sort((a, b) => {
    const da = a.type === "directory", db = b.type === "directory";
    if (da !== db) return da ? -1 : 1;
    let r;
    if (ui.sortBy === "size") r = (a.size || 0) - (b.size || 0);
    else if (ui.sortBy === "modified") r = new Date(a.modified) - new Date(b.modified);
    else r = collator.compare(a.name, b.name);
    return r * dir || collator.compare(a.name, b.name);
  });
  return list;
});

const openFile = computed(() => (isFile.value && data.value ? items.value.find((i) => i.path === fullPath.value) || data.value.items?.find((i) => i.path === fullPath.value) : null));
const siblingsFiles = computed(() => items.value.filter((i) => i.type !== "directory"));
const selected = computed(() => items.value.filter((i) => sel.value.has(i.path)));

async function load(quiet = false) {
  if (!quiet) { loading.value = true; error.value = ""; }
  try {
    data.value = await filesApi.list(source.value, dirPath.value);
    sel.value = new Set([...sel.value].filter((p) => data.value.items.some((i) => i.path === p)));
  } catch (e) {
    if (!quiet) { data.value = null; error.value = e.status === 404 ? "Esta pasta não existe." : e.status === 403 ? "Você não tem acesso a esta pasta." : e.message; }
  } finally {
    loading.value = false;
  }
}

watch([source, dirPath], () => { sel.value = new Set(); lastIdx = -1; void load(); }, { immediate: true });
watch([folderName], () => { page.title = folderName.value; }, { immediate: true });

const fileRoute = (it) => routeFor(source.value, it.path).replace(/\/$/, "");
function open(it) {
  if (it.type === "directory") router.push(routeFor(source.value, it.path));
  else router.push(fileRoute(it));
}
const closeViewer = () => router.push(routeFor(source.value, dirPath.value));

// ---- seleção ----
function toggle(it, idx, e) {
  const s = new Set(sel.value);
  if (e?.shiftKey && lastIdx >= 0) {
    const [a, b] = [Math.min(lastIdx, idx), Math.max(lastIdx, idx)];
    for (let k = a; k <= b; k++) s.add(items.value[k].path);
  } else if (s.has(it.path)) s.delete(it.path);
  else s.add(it.path);
  lastIdx = idx;
  sel.value = s;
}
function rowClick(it, idx, e) {
  if (e.ctrlKey || e.metaKey || e.shiftKey || sel.value.size) toggle(it, idx, e);
  else open(it);
}
function selectAll() { sel.value = sel.value.size === items.value.length ? new Set() : new Set(items.value.map((i) => i.path)); }
const clearSel = () => { sel.value = new Set(); };

// ---- ações ----
function targets(it) { return it && !sel.value.has(it.path) ? [it] : selected.value; }

async function doRename(it) {
  const name = await askText({ title: "Renomear", label: "Novo nome", value: it.name, confirm: "Renomear", extra: { selectBase: it.type !== "directory" } });
  if (!name || name === it.name) return;
  try {
    await filesApi.moveCopy([{ fromSource: source.value, from: it.path, toSource: source.value, to: join(dirPath.value, name, it.type === "directory") }], "move");
    toast("Renomeado.");
    clearSel();
    await load(true);
  } catch (e) { toastError(e); }
}

async function doMoveCopy(list, action) {
  const dest = await pickFolder({ title: action === "move" ? "Mover para…" : "Copiar para…", extra: { source: source.value, path: dirPath.value } });
  if (!dest) return;
  try {
    const res = await filesApi.moveCopy(list.map((i) => ({ fromSource: source.value, from: i.path, toSource: dest.source, to: join(dest.path, i.name, i.type === "directory") })), action);
    const failed = res?.failed?.length || 0;
    if (failed) toast(`${failed} item(ns) não puderam ser ${action === "move" ? "movidos" : "copiados"}.`, "err");
    else toast(action === "move" ? "Movido." : "Copiado.");
    clearSel();
    await load(true);
    void refreshSources();
  } catch (e) { toastError(e); }
}

async function doDelete(list) {
  const msg = list.length === 1 ? `"${list[0].name}" será apagado.` : `${list.length} itens serão apagados.`;
  const ok = await askConfirm({ title: "Apagar", message: msg + " Isso não pode ser desfeito.", confirm: "Apagar", danger: true });
  if (!ok) return;
  try {
    await filesApi.remove(list.map((i) => ({ source: source.value, path: i.path })));
    toast("Apagado.");
    clearSel();
    if (isFile.value) closeViewer();
    await load(true);
    void refreshSources();
  } catch (e) { toastError(e); }
}

async function doZip(list) {
  const base = list.length === 1 ? list[0].name.replace(/\.[^.]+$/, "") : folderName.value;
  const name = await askText({ title: "Compactar em .zip", label: "Nome do arquivo", value: base + ".zip", confirm: "Compactar" });
  if (!name) return;
  try {
    await filesApi.archive(source.value, list.map((i) => i.path), join(dirPath.value, name.endsWith(".zip") ? name : name + ".zip", false));
    toast("Arquivo compactado.");
    clearSel();
    await load(true);
    void refreshSources();
  } catch (e) { toastError(e); }
}

async function doUnzip(it) {
  try {
    await filesApi.unarchive(source.value, it.path, dirPath.value);
    toast("Extraído nesta pasta.");
    await load(true);
    void refreshSources();
  } catch (e) { toastError(e); }
}
const isArchive = (it) => /\.(zip|tar|tar\.gz|tgz|gz)$/i.test(it.name);

function doDownload(list) {
  const paths = list.map((i) => i.path);
  const archive = list.length > 1 || list[0].type === "directory";
  const a = document.createElement("a");
  a.href = filesApi.downloadUrl(source.value, paths, { archive });
  a.download = archive ? (list.length === 1 ? list[0].name : folderName.value) + ".zip" : list[0].name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  noteDownload(a.download, list.reduce((s, i) => s + (i.size || 0), 0));
}

async function newFolder() {
  const name = await askText({ title: "Nova pasta", label: "Nome da pasta", confirm: "Criar" });
  if (!name) return;
  try { await filesApi.mkdir(source.value, join(dirPath.value, name, true)); toast("Pasta criada."); await load(true); } catch (e) { toastError(e); }
}
async function newFile() {
  const name = await askText({ title: "Novo arquivo", label: "Nome do arquivo", value: "novo.txt", confirm: "Criar", extra: { selectBase: true } });
  if (!name) return;
  try { await filesApi.writeText(source.value, join(dirPath.value, name, false), ""); await load(true); router.push(routeFor(source.value, join(dirPath.value, name, false)).replace(/\/$/, "")); } catch (e) { toastError(e); }
}

async function startUpload(files) {
  if (!files.length) return;
  if (!canCreate.value) { toast("Você não tem permissão para enviar aqui.", "err"); return; }
  const names = new Set((data.value?.items || []).map((i) => i.name));
  const clash = files.filter((f) => names.has(f.rel.split("/")[0]));
  let overwrite = false;
  let list = files;
  if (clash.length) {
    const ok = await askConfirm({ title: "Já existem itens com esse nome", message: `${new Set(clash.map((f) => f.rel.split("/")[0])).size} nome(s) já existem nesta pasta. Substituir?`, confirm: "Substituir" });
    if (ok) overwrite = true;
    else { const skip = new Set(clash); list = files.filter((f) => !skip.has(f)); }
  }
  if (!list.length) return;
  enqueueUploads(source.value, dirPath.value, list, { overwrite });
}
async function onPick(e, folder) {
  const files = filesFromInput(e.target);
  e.target.value = "";
  await startUpload(files);
}

// ---- arrastar e soltar ----
function dragEnter(e) { if (e.dataTransfer?.types?.includes("Files")) { dragDepth++; dragging.value = true; } }
function dragLeave() { dragDepth = Math.max(0, dragDepth - 1); if (!dragDepth) dragging.value = false; }
async function drop(e) {
  dragDepth = 0; dragging.value = false;
  if (!e.dataTransfer?.types?.includes("Files")) return;
  e.preventDefault();
  await startUpload(await filesFromDrop(e.dataTransfer));
}

// ---- menu de contexto ----
function menuFor(it) {
  const one = targets(it).length === 1;
  const t = targets(it);
  const e = [{ id: "open", icon: it.type === "directory" ? "folderopen" : "open", label: "Abrir" }];
  if (canDownload.value) e.push({ id: "download", icon: "download", label: "Baixar" });
  e.push({ sep: true });
  if (canModify.value && one) e.push({ id: "rename", icon: "edit", label: "Renomear" });
  if (canCreate.value) e.push({ id: "copy", icon: "copy", label: "Copiar para…" });
  if (canModify.value) e.push({ id: "move", icon: "move", label: "Mover para…" });
  if (canCreate.value) e.push({ id: "zip", icon: "zip", label: "Compactar (.zip)" });
  if (canCreate.value && one && isArchive(it)) e.push({ id: "unzip", icon: "zip", label: "Extrair aqui" });
  if (one) e.push({ id: "share", icon: "share", label: "Compartilhar" });
  if (canDelete.value) { e.push({ sep: true }); e.push({ id: "delete", icon: "trash", label: t.length > 1 ? `Apagar ${t.length} itens` : "Apagar", danger: true }); }
  return e;
}
function showMenu(it, ev) {
  ev.preventDefault();
  ev.stopPropagation();
  const r = ev.currentTarget?.getBoundingClientRect?.();
  const x = ev.clientX || (r ? r.right - 8 : 100);
  const y = ev.clientY || (r ? r.bottom : 100);
  menu.value = { it, x, y, entries: menuFor(it) };
}
function menuPick(id) {
  const it = menu.value.it;
  const list = targets(it);
  menu.value = null;
  if (id === "open") open(it);
  else if (id === "download") doDownload(list);
  else if (id === "rename") void doRename(it);
  else if (id === "copy") void doMoveCopy(list, "copy");
  else if (id === "move") void doMoveCopy(list, "move");
  else if (id === "zip") void doZip(list);
  else if (id === "unzip") void doUnzip(it);
  else if (id === "share") void shareItem(source.value, it);
  else if (id === "delete") void doDelete(list);
}

// ---- teclado ----
function onKey(e) {
  if (/input|textarea|select/i.test(e.target.tagName) || e.target.isContentEditable) return;
  if (isFile.value) return;
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "a") { e.preventDefault(); selectAll(); }
  else if (e.key === "Delete" && selected.value.length && canDelete.value) void doDelete(selected.value);
  else if (e.key === "F2" && selected.value.length === 1 && canModify.value) void doRename(selected.value[0]);
  else if (e.key === "Escape") clearSel();
}

let off = null;
onMounted(() => {
  document.addEventListener("keydown", onKey);
  off = onTransferDone((t) => { if (t.source === source.value && t.dir.startsWith(dirPath.value.replace(/\/$/, "") + "/") || t.source === source.value && t.dir === dirPath.value) void load(true); });
});
onBeforeUnmount(() => { document.removeEventListener("keydown", onKey); off && off(); });

const sortLabel = { name: "Nome", size: "Tamanho", modified: "Modificado" };
function sortBy(k) { setUi(ui.sortBy === k ? { sortAsc: !ui.sortAsc } : { sortBy: k, sortAsc: true }); }
const thumb = (it) => (it.hasPreview && ["image", "video"].includes(kindOf(it)) ? filesApi.previewUrl(source.value, it.path, it.modified) : "");
const crumbList = computed(() => crumbs(dirPath.value));
</script>

<template>
  <div class="page browse" @dragenter="dragEnter" @dragleave="dragLeave" @dragover.prevent @drop="drop" @click="menu = null">
    <div class="wrap wide">
      <div class="phead">
        <h1>{{ folderName }}</h1>
        <div class="crumbs">
          <router-link to="/">Início</router-link><span>/</span>
          <router-link :to="routeFor(source, '/')">{{ source }}</router-link>
          <template v-for="c in crumbList" :key="c.path"><span>/</span><router-link :to="routeFor(source, c.path)">{{ c.name }}</router-link></template>
        </div>
      </div>

      <div class="tbar" :class="{ has: selected.length }">
        <template v-if="!selected.length">
          <button v-if="canCreate" class="btn pri" type="button" @click="fileInput.click()"><Icon name="upload" /><span class="lbl">Enviar</span></button>
          <button v-if="canCreate" class="btn" type="button" title="Enviar uma pasta inteira" @click="folderInput.click()"><Icon name="folderopen" /><span class="lbl">Enviar pasta</span></button>
          <button v-if="canCreate" class="btn" type="button" @click="newFolder"><Icon name="foldernew" /><span class="lbl">Nova pasta</span></button>
          <button v-if="canCreate" class="btn" type="button" title="Novo arquivo de texto" @click="newFile"><Icon name="file" /><span class="lbl">Novo arquivo</span></button>
          <span class="sp"></span>
          <label class="sortsel"><Icon name="sort" /><select :value="ui.sortBy" aria-label="Ordenar por" @change="setUi({ sortBy: $event.target.value })"><option v-for="(l, k) in sortLabel" :key="k" :value="k">{{ l }}</option></select></label>
          <button class="ib sortord" :title="ui.sortAsc ? 'Crescente' : 'Decrescente'" :aria-label="ui.sortAsc ? 'Ordem crescente' : 'Ordem decrescente'" @click="setUi({ sortAsc: !ui.sortAsc })"><Icon :name="ui.sortAsc ? 'up' : 'down'" /></button>
          <div class="seg" role="group" aria-label="Modo de exibição">
            <button :aria-pressed="ui.view === 'list'" aria-label="Lista" title="Lista" @click="setUi({ view: 'list' })"><Icon name="list" /></button>
            <button :aria-pressed="ui.view === 'grid'" aria-label="Grade" title="Grade" @click="setUi({ view: 'grid' })"><Icon name="grid" /></button>
          </div>
          <button class="ib" aria-label="Atualizar" title="Atualizar" @click="load(true)"><Icon name="refresh" /></button>
        </template>
        <template v-else>
          <button class="ib" aria-label="Limpar seleção" title="Limpar seleção (Esc)" @click="clearSel"><Icon name="dismiss" /></button>
          <b class="selcount">{{ selected.length }} selecionado{{ selected.length > 1 ? "s" : "" }}</b>
          <span class="sp"></span>
          <button v-if="canDownload" class="btn" @click="doDownload(selected)"><Icon name="download" /><span class="lbl">Baixar</span></button>
          <button v-if="canModify && selected.length === 1" class="btn" @click="doRename(selected[0])"><Icon name="edit" /><span class="lbl">Renomear</span></button>
          <button v-if="canCreate" class="btn" @click="doMoveCopy(selected, 'copy')"><Icon name="copy" /><span class="lbl">Copiar</span></button>
          <button v-if="canModify" class="btn" @click="doMoveCopy(selected, 'move')"><Icon name="move" /><span class="lbl">Mover</span></button>
          <button v-if="canCreate" class="btn" @click="doZip(selected)"><Icon name="zip" /><span class="lbl">Compactar</span></button>
          <button v-if="selected.length === 1" class="btn" @click="shareItem(source, selected[0])"><Icon name="share" /><span class="lbl">Compartilhar</span></button>
          <button v-if="canDelete" class="btn danger" @click="doDelete(selected)"><Icon name="trash" /><span class="lbl">Apagar</span></button>
        </template>
      </div>
      <input ref="fileInput" type="file" multiple hidden @change="onPick($event)" />
      <input ref="folderInput" type="file" webkitdirectory hidden @change="onPick($event)" />

      <div v-if="loading" class="vazio">Carregando…</div>
      <div v-else-if="error" class="vazio">{{ error }}</div>
      <div v-else-if="!items.length" class="vazio">Pasta vazia.<span v-if="canCreate"> Arraste arquivos para cá ou use Enviar.</span></div>

      <div v-else-if="ui.view === 'list'" class="flist" role="table" aria-label="Arquivos">
        <div class="frow fhead" role="row">
          <button class="chk" :aria-label="sel.size === items.length ? 'Desmarcar todos' : 'Selecionar todos'" @click="selectAll"><Icon :name="sel.size === items.length ? 'checkboxon' : 'checkbox'" /></button>
          <button class="fh" @click="sortBy('name')">Nome <Icon v-if="ui.sortBy === 'name'" :name="ui.sortAsc ? 'up' : 'down'" /></button>
          <button class="fh hide-s" @click="sortBy('modified')">Modificado <Icon v-if="ui.sortBy === 'modified'" :name="ui.sortAsc ? 'up' : 'down'" /></button>
          <button class="fh r" @click="sortBy('size')">Tamanho <Icon v-if="ui.sortBy === 'size'" :name="ui.sortAsc ? 'up' : 'down'" /></button>
          <span></span>
        </div>
        <div v-for="(it, i) in items" :key="it.path" class="frow" :class="{ sel: sel.has(it.path) }" role="row" tabindex="0"
          @click="rowClick(it, i, $event)" @keydown.enter="open(it)" @contextmenu="showMenu(it, $event)">
          <button class="chk" :aria-label="'Selecionar ' + it.name" @click.stop="toggle(it, i, $event)"><Icon :name="sel.has(it.path) ? 'checkboxon' : iconKey(kindOf(it))" /></button>
          <span class="n" :title="it.name">{{ it.name }}</span>
          <span class="m hide-s">{{ date(it.modified) }}</span>
          <span class="s">{{ it.type === "directory" ? "—" : bytes(it.size) }}</span>
          <button class="ib more" aria-label="Mais ações" title="Mais ações" @click.stop="showMenu(it, $event)"><Icon name="more" /></button>
        </div>
      </div>

      <div v-else class="fgrid">
        <div v-for="(it, i) in items" :key="it.path" class="gcard" :class="{ sel: sel.has(it.path) }" tabindex="0"
          @click="rowClick(it, i, $event)" @keydown.enter="open(it)" @contextmenu="showMenu(it, $event)">
          <button class="gchk" :aria-label="'Selecionar ' + it.name" @click.stop="toggle(it, i, $event)"><Icon :name="sel.has(it.path) ? 'checkboxon' : 'checkbox'" /></button>
          <div class="gthumb">
            <img v-if="thumb(it)" :src="thumb(it)" :alt="''" loading="lazy" @error="$event.target.remove()" />
            <Icon :name="iconKey(kindOf(it))" />
          </div>
          <div class="gname" :title="it.name">{{ it.name }}</div>
          <div class="gmeta">{{ it.type === "directory" ? "Pasta" : bytes(it.size) }}</div>
        </div>
      </div>

      <div v-if="data" class="foot-info">{{ items.length }} {{ items.length === 1 ? "item" : "itens" }}</div>
    </div>

    <div v-if="dragging" class="dropzone"><div><Icon name="upload" /><b>Solte para enviar para {{ folderName }}</b></div></div>
    <ItemMenu v-if="menu" :x="menu.x" :y="menu.y" :entries="menu.entries" @pick="menuPick" @close="menu = null" />
    <FileViewer v-if="openFile" :source="source" :item="openFile" :siblings="siblingsFiles" :can-modify="canModify" :can-download="canDownload"
      @close="closeViewer" @go="open" @saved="load(true)" @delete="doDelete([$event])" @share="shareItem(source, $event)" />
    <div v-else-if="isFile && !loading && !error" class="vazio">Arquivo não encontrado.</div>
  </div>
</template>
