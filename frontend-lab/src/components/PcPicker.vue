<script setup>
// Explorador do computador: depois que o usuário escolhe uma pasta-base na janela nativa do navegador (showDirectoryPicker),
// ele navega pelas pastas e marca arquivos e pastas misturados, como no celular. Nada é lido além do que for marcado.
import { ref, reactive, computed, onMounted } from "vue";
import Icon from "./Icon.vue";
import { bytes } from "@/lib/format";

const props = defineProps({ root: { type: Object, required: true }, dest: { type: String, default: "" }, sendNow: { type: Boolean, default: true } });
const emit = defineEmits(["done", "close"]);

const stack = ref([{ name: props.root.name, handle: props.root }]); // pasta-base + subpastas abertas
const entries = ref([]);
const loading = ref(false);
const err = ref("");
const busy = ref(false);
const sel = reactive(new Map()); // chave (caminho completo) -> { handle, name, kind }
const cur = computed(() => stack.value[stack.value.length - 1]);
const pathKey = (name) => stack.value.map((s) => s.name).join("/") + "/" + name;
const natural = new Intl.Collator("pt-BR", { numeric: true, sensitivity: "base" });

async function load() {
  loading.value = true; err.value = "";
  try {
    const list = [];
    for await (const [name, handle] of cur.value.handle.entries()) list.push({ name, handle, kind: handle.kind });
    list.sort((a, b) => (a.kind === b.kind ? natural.compare(a.name, b.name) : a.kind === "directory" ? -1 : 1));
    entries.value = list;
  } catch (e) {
    entries.value = [];
    err.value = "Não foi possível abrir esta pasta: " + (e?.message || e);
  } finally { loading.value = false; }
}
onMounted(load);

function enter(e) { stack.value = [...stack.value, { name: e.name, handle: e.handle }]; load(); }
function goto(i) { stack.value = stack.value.slice(0, i + 1); load(); }
function up() { if (stack.value.length > 1) goto(stack.value.length - 2); }
const isSel = (e) => sel.has(pathKey(e.name));
function toggle(e) { const k = pathKey(e.name); sel.has(k) ? sel.delete(k) : sel.set(k, { handle: e.handle, name: e.name, kind: e.kind }); }
const allSel = computed(() => entries.value.length > 0 && entries.value.every(isSel));
function toggleAll() { for (const e of entries.value) { const k = pathKey(e.name); if (allSel.value) sel.delete(k); else sel.set(k, { handle: e.handle, name: e.name, kind: e.kind }); } }
function clickRow(e) { e.kind === "directory" ? enter(e) : toggle(e); }

const counts = computed(() => {
  let d = 0, f = 0;
  for (const v of sel.values()) v.kind === "directory" ? d++ : f++;
  return { d, f, n: d + f };
});
const plural = (n, a, b) => n + " " + (n === 1 ? a : b);
const label = computed(() => [counts.value.d && plural(counts.value.d, "pasta", "pastas"), counts.value.f && plural(counts.value.f, "arquivo", "arquivos")].filter(Boolean).join(" · "));

// pasta marcada: percorre tudo dentro dela (pasta vazia vira { dir: true })
async function walk(handle, prefix, out) {
  let any = false;
  for await (const [name, h] of handle.entries()) {
    any = true;
    if (h.kind === "file") out.push({ file: await h.getFile(), rel: prefix + "/" + name });
    else await walk(h, prefix + "/" + name, out);
  }
  if (!any) out.push({ dir: true, rel: prefix });
}
async function confirm() {
  if (!counts.value.n || busy.value) return;
  busy.value = true; err.value = "";
  try {
    const out = [];
    for (const v of sel.values()) {
      if (v.kind === "file") out.push({ file: await v.handle.getFile(), rel: v.name });
      else await walk(v.handle, v.name, out);
    }
    emit("done", out);
  } catch (e) {
    err.value = "Falha ao ler o que foi marcado: " + (e?.message || e);
    busy.value = false;
  }
}
</script>

<template>
  <div class="dlg-scrim" @click.self="emit('close')" @keydown.esc="emit('close')">
    <div class="dlg pcp" role="dialog" aria-label="Escolher o que enviar">
      <div class="pcp-h">
        <b>Escolher o que enviar</b>
        <small v-if="dest">Destino: {{ dest }}</small>
      </div>
      <div class="pcp-bar">
        <button type="button" class="btn" :disabled="stack.length < 2" @click="up" title="Subir uma pasta" aria-label="Subir"><Icon name="up" /></button>
        <div class="pcp-path">
          <template v-for="(s, i) in stack" :key="i"><span v-if="i">/</span><button type="button" @click="goto(i)">{{ s.name }}</button></template>
        </div>
        <button type="button" class="btn" :disabled="!entries.length" @click="toggleAll"><Icon :name="allSel ? 'checkboxon' : 'checkbox'" /> Marcar tudo</button>
      </div>
      <div class="pcp-list" role="listbox" aria-multiselectable="true">
        <div v-if="loading" class="pcp-msg">Carregando…</div>
        <div v-else-if="err" class="pcp-msg err">{{ err }}</div>
        <div v-else-if="!entries.length" class="pcp-msg">Pasta vazia.</div>
        <template v-else>
          <div v-for="e in entries" :key="e.name" class="pcp-row" :class="{ on: isSel(e) }" role="option" :aria-selected="isSel(e)" @click="clickRow(e)">
            <button type="button" class="pcp-ck" :aria-label="'Marcar ' + e.name" @click.stop="toggle(e)"><Icon :name="isSel(e) ? 'checkboxon' : 'checkbox'" /></button>
            <Icon class="pcp-ic" :name="e.kind === 'directory' ? 'folder' : 'file'" />
            <span class="pcp-n">{{ e.name }}</span>
            <Icon v-if="e.kind === 'directory'" class="pcp-go" name="chev" />
          </div>
        </template>
      </div>
      <p class="pcp-tip">Clique numa pasta para abrir. Use a caixinha para marcar pastas e arquivos de qualquer lugar da navegação.</p>
      <div class="dlg-act pcp-act">
        <span class="pcp-sum">{{ label || "Nada marcado" }}</span>
        <button class="btn" type="button" @click="emit('close')">Cancelar</button>
        <button class="btn pri" type="button" :disabled="!counts.n || busy" @click="confirm">{{ busy ? "Lendo…" : sendNow ? "Enviar" : "Adicionar" }}</button>
      </div>
    </div>
  </div>
</template>
