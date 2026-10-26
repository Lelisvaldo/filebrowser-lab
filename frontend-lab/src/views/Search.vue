<script setup>
import { ref, watch, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as filesApi from "@/api/files";
import { app, page, userSources, hasUsbSource, usbSourceName, toastError } from "@/store/app";
import { kindOf, iconKey } from "@/lib/fileTypes";
import { bytes, date } from "@/lib/format";
import { parent, routeFor } from "@/lib/paths";

const route = useRoute();
const router = useRouter();
const results = ref([]);
const by = ref("");   // "" = ordem do servidor
const asc = ref(true);
const collator = new Intl.Collator("pt-BR", { numeric: true, sensitivity: "base" });
function sortBy(k) {
  if (by.value === k) { if (asc.value) asc.value = false; else { by.value = ""; asc.value = true; } } else { by.value = k; asc.value = true; }
}
const sorted = computed(() => {
  const l = [...results.value];
  if (!by.value) return l;
  const d = asc.value ? 1 : -1;
  l.sort((a, b) => d * (by.value === "size" ? (a.size || 0) - (b.size || 0) : by.value === "modified" ? new Date(a.modified) - new Date(b.modified) : collator.compare(a.path.replace(/\/$/, "").split("/").pop(), b.path.replace(/\/$/, "").split("/").pop())));
  return l;
});
const loading = ref(false);
const q = computed(() => String(route.query.q || ""));
const onlySource = computed(() => String(route.query.source || ""));
const allSources = computed(() => [...userSources().map((s) => s.name), ...(hasUsbSource() ? [usbSourceName] : [])]);

async function run() {
  page.title = "Busca";
  if (q.value.length < 2) { results.value = []; return; }
  loading.value = true;
  try {
    const srcs = onlySource.value ? [onlySource.value] : allSources.value;
    const res = await filesApi.search(q.value, srcs);
    results.value = (Array.isArray(res) ? res : []).map((r) => {
      const path = "/" + String(r.path).replace(/^\/+/, "");
      return { ...r, path: r.type === "directory" && !path.endsWith("/") ? path + "/" : path };
    });
  } catch (e) { toastError(e); results.value = []; } finally { loading.value = false; }
}
watch(() => route.fullPath, run, { immediate: true });

function open(r) {
  if (r.type === "directory") router.push(routeFor(r.source, r.path));
  else router.push(routeFor(r.source, r.path).replace(/\/$/, ""));
}
const where = (r) => r.source + parent(r.path).replace(/\/$/, "");
</script>

<template>
  <div class="page">
    <div class="wrap wide">
      <div class="phead"><h1>Busca</h1><div class="crumbs"><span>“{{ q }}”</span><span v-if="onlySource">em {{ onlySource }}</span><span v-else>em todas as pastas</span><span>· {{ results.length }} resultado{{ results.length === 1 ? "" : "s" }}</span></div></div>
      <div v-if="loading" class="vazio">Buscando…</div>
      <div v-else-if="q.length < 2" class="vazio">Digite ao menos 2 letras na busca.</div>
      <div v-else-if="!results.length" class="vazio">Nada encontrado para “{{ q }}”.</div>
      <div v-else class="flist">
        <div class="frow fhead" role="row">
          <span></span>
          <button class="fh" @click="sortBy('name')">Nome <Icon v-if="by === 'name'" :name="asc ? 'up' : 'down'" /></button>
          <button class="fh hide-s" @click="sortBy('modified')">Data <Icon v-if="by === 'modified'" :name="asc ? 'up' : 'down'" /></button>
          <button class="fh r" @click="sortBy('size')">Tamanho <Icon v-if="by === 'size'" :name="asc ? 'up' : 'down'" /></button>
          <span></span>
        </div>
        <div v-for="r in sorted" :key="r.source + r.path" class="frow srow" tabindex="0" @click="open(r)" @keydown.enter="open(r)">
          <span class="chk" aria-hidden="true"><Icon :name="iconKey(kindOf(r))" /></span>
          <span class="n"><span>{{ r.path.replace(/\/$/, "").split("/").pop() }}</span><small>{{ where(r) }}</small></span>
          <span class="m hide-s">{{ date(r.modified) }}</span>
          <span class="s">{{ r.type === "directory" ? "—" : bytes(r.size) }}</span>
          <span></span>
        </div>
      </div>
    </div>
  </div>
</template>
