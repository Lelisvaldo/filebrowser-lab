<script setup>
// Prévia de planilhas (xlsx, xls, ods, tsv) no navegador: mostra as abas e até 2000 linhas por aba.
import { ref, watch, onMounted } from "vue";
const props = defineProps({ url: String });
const emit = defineEmits(["fail"]);
const LIM = 2000, COLS = 60;
const sheets = ref([]);
const cur = ref(0);
const state = ref("loading");
const cut = ref(false);
async function load() {
  state.value = "loading"; cut.value = false;
  try {
    const [res, X] = await Promise.all([fetch(props.url, { credentials: "same-origin" }), import("xlsx")]);
    if (!res.ok) throw new Error(String(res.status));
    const wb = X.read(await res.arrayBuffer(), { type: "array", cellDates: true });
    sheets.value = wb.SheetNames.map((n) => {
      const rows = X.utils.sheet_to_json(wb.Sheets[n], { header: 1, defval: "", raw: false });
      if (rows.length > LIM) cut.value = true;
      return { name: n, rows: rows.slice(0, LIM).map((r) => r.slice(0, COLS)) };
    });
    cur.value = 0; state.value = "ok";
  } catch (e) { state.value = "err"; emit("fail"); }
}
onMounted(load);
watch(() => props.url, load);
</script>

<template>
  <div class="sheet-wrap">
    <div v-if="state === 'loading'" class="vw-msg">Abrindo planilha…</div>
    <div v-else-if="state === 'err'" class="vw-msg">Não foi possível mostrar esta planilha. Baixe para abrir no computador.</div>
    <template v-else>
      <div v-if="sheets.length > 1" class="sheet-tabs" role="tablist">
        <button v-for="(s, i) in sheets" :key="s.name" role="tab" :aria-selected="cur === i" :class="{ on: cur === i }" @click="cur = i">{{ s.name }}</button>
      </div>
      <div class="sheet-scroll">
        <table class="sheet">
          <tbody><tr v-for="(r, ri) in sheets[cur].rows" :key="ri"><th>{{ ri + 1 }}</th><td v-for="(c, ci) in r" :key="ci">{{ c }}</td></tr></tbody>
        </table>
      </div>
      <div v-if="cut" class="sheet-note">Mostrando só as primeiras {{ LIM }} linhas. Baixe o arquivo para ver tudo.</div>
    </template>
  </div>
</template>
