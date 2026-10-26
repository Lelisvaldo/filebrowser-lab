<script setup>
import { ref, computed, watch, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as filesApi from "@/api/files";
import { app, page, hideStatus, toastError, refreshSources } from "@/store/app";
import { kindOf, iconKey } from "@/lib/fileTypes";
import { bytes, date, pct } from "@/lib/format";
import { crumbs, parent, routeFor } from "@/lib/paths";

const route = useRoute();
const router = useRouter();
const source = computed(() => String(route.params.source || ""));
const dirPath = computed(() => "/" + String(route.params.rest || ""));
const tab = ref("arquivos");
const minMb = ref(100);
const files = ref(null);
const dirs = ref(null);
const loading = ref(false);

const info = computed(() => app.sources[source.value] || {});
const crumbList = computed(() => crumbs(dirPath.value));
const folderName = computed(() => (dirPath.value === "/" ? source.value : dirPath.value.split("/").filter(Boolean).pop()));

async function loadFiles() {
  loading.value = true;
  files.value = null;
  try {
    const r = await filesApi.largest(source.value, dirPath.value, minMb.value);
    files.value = (Array.isArray(r) ? r : []).map((x) => ({ ...x, path: "/" + String(x.path).replace(/^\/+/, "") })).sort((a, b) => b.size - a.size).slice(0, 100);
  } catch (e) { files.value = []; toastError(e); } finally { loading.value = false; }
}
async function loadDirs() {
  try {
    const d = await filesApi.list(source.value, dirPath.value);
    dirs.value = d.items.filter((i) => i.type === "directory").sort((a, b) => (b.size || 0) - (a.size || 0));
    total.value = d.size || d.items.reduce((s, i) => s + (i.size || 0), 0);
  } catch (e) { dirs.value = []; toastError(e); }
}
const total = ref(0);

watch([source, dirPath], () => { void loadFiles(); void loadDirs(); page.title = "Limpeza · " + folderName.value; }, { immediate: true });
watch(minMb, loadFiles);
onMounted(() => { hideStatus(); void refreshSources(); });

const maxFile = computed(() => Math.max(1, ...(files.value || []).map((f) => f.size)));
const maxDir = computed(() => Math.max(1, ...(dirs.value || []).map((f) => f.size || 0)));
const hot = (size, base) => base > 0 && size / base > 0.25;
const parentOf = (f) => source.value + parent(f.path).replace(/\/$/, "");
const openFolder = (f) => router.push(routeFor(source.value, parent(f.path)));
const openFile = (f) => router.push(routeFor(source.value, f.path).replace(/\/$/, ""));
const goDir = (d) => router.push("/limpeza/" + encodeURIComponent(source.value) + routeFor(source.value, d.path).replace(/^\/f\/[^/]+/, ""));
const up = () => router.push("/limpeza/" + encodeURIComponent(source.value) + routeFor(source.value, parent(dirPath.value.replace(/\/?$/, "/"))).replace(/^\/f\/[^/]+/, ""));
const sizes = [[1, "1 MB"], [100, "100 MB"], [1024, "1 GB"]];
</script>

<template>
  <div class="page">
    <div class="wrap wide">
      <div class="phead">
        <h1>Limpeza</h1>
        <div class="crumbs">
          <router-link to="/">Início</router-link><span>/</span>
          <router-link :to="'/limpeza/' + encodeURIComponent(source) + '/'">{{ source }}</router-link>
          <template v-for="c in crumbList" :key="c.path"><span>/</span><router-link :to="'/limpeza/' + encodeURIComponent(source) + routeFor(source, c.path).replace(/^\/f\/[^/]+/, '')">{{ c.name }}</router-link></template>
        </div>
      </div>

      <div class="sum">
        <div><span class="s">Pasta</span><b>{{ folderName }}</b></div>
        <div><span class="s">Ocupa</span><b>{{ bytes(total || info.used) }}</b></div>
        <div v-if="info.total"><span class="s">Disco</span><b>{{ bytes(info.used) }} de {{ bytes(info.total) }} · {{ pct(info.used, info.total) }}%</b></div>
      </div>

      <div class="tbar">
        <div class="seg" role="tablist">
          <button role="tab" :aria-pressed="tab === 'arquivos'" title="Maiores arquivos" aria-label="Maiores arquivos" @click="tab = 'arquivos'"><Icon name="file" /></button>
          <button role="tab" :aria-pressed="tab === 'pastas'" title="Maiores pastas" aria-label="Maiores pastas" @click="tab = 'pastas'"><Icon name="folder" /></button>
        </div>
        <span class="meta">{{ tab === "arquivos" ? "Maiores arquivos dentro desta pasta" : "Subpastas, da maior para a menor" }}</span>
        <span class="sp"></span>
        <div v-if="tab === 'arquivos'" class="seg" role="group" aria-label="Tamanho mínimo">
          <button v-for="[v, l] in sizes" :key="v" :aria-pressed="minMb === v" @click="minMb = v">≥ {{ l }}</button>
        </div>
        <button v-if="dirPath !== '/'" class="ib" title="Subir um nível" aria-label="Subir um nível" @click="up"><Icon name="back" /></button>
        <router-link class="ib" :to="routeFor(source, dirPath)" title="Abrir esta pasta nos arquivos" aria-label="Abrir esta pasta nos arquivos"><Icon name="folderopen" /></router-link>
      </div>

      <template v-if="tab === 'arquivos'">
        <div v-if="loading || files === null" class="vazio">Procurando os maiores arquivos…</div>
        <div v-else-if="!files.length" class="vazio vi"><Icon name="broom" /><span>Nenhum arquivo com {{ minMb >= 1024 ? "1 GB" : minMb + " MB" }} ou mais aqui.</span></div>
        <div v-else class="flist">
          <div v-for="f in files" :key="f.path" class="frow urow" tabindex="0" @click="openFile(f)" @keydown.enter="openFile(f)">
            <span class="chk"><Icon :name="iconKey(kindOf(f))" /></span>
            <span class="n"><span>{{ f.path.split("/").pop() }}</span><small>{{ parentOf(f) }}</small></span>
            <div class="bar" :class="{ cheio: hot(f.size, total) }"><i :style="{ width: Math.max(2, (f.size / maxFile) * 100) + '%' }"></i></div>
            <span class="m hide-s">{{ date(f.modified) }}</span>
            <span class="s">{{ bytes(f.size) }}</span>
            <button class="ib more" title="Mostrar na pasta" aria-label="Mostrar na pasta" @click.stop="openFolder(f)"><Icon name="folderopen" /></button>
          </div>
        </div>
      </template>
      <template v-else>
        <div v-if="dirs === null" class="vazio">Carregando…</div>
        <div v-else-if="!dirs.length" class="vazio vi"><Icon name="folder" /><span>Esta pasta não tem subpastas.</span></div>
        <div v-else class="flist">
          <div v-for="d in dirs" :key="d.path" class="frow urow" tabindex="0" @click="goDir(d)" @keydown.enter="goDir(d)">
            <span class="chk"><Icon name="folder" /></span>
            <span class="n">{{ d.name }}</span>
            <div class="bar" :class="{ cheio: hot(d.size || 0, total) }"><i :style="{ width: Math.max(2, ((d.size || 0) / maxDir) * 100) + '%' }"></i></div>
            <span class="m hide-s">{{ total ? Math.round(((d.size || 0) / total) * 100) + "%" : "" }}</span>
            <span class="s">{{ bytes(d.size || 0) }}</span>
            <button class="ib more" title="Abrir nos arquivos" aria-label="Abrir nos arquivos" @click.stop="router.push(routeFor(source, d.path))"><Icon name="folderopen" /></button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
