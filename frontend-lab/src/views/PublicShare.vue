<script setup>
import { ref, computed, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import Icon from "@/components/Icon.vue";
import * as shareApi from "@/api/share";
import { kindOf, iconKey } from "@/lib/fileTypes";
import { bytes, date } from "@/lib/format";
import { appName } from "@/lib/config";
import { toastError } from "@/store/app";
import Toasts from "@/components/Toasts.vue";

const route = useRoute();
const router = useRouter();
const hash = computed(() => String(route.params.hash));
const rest = computed(() => String(route.params.rest || ""));
const path = computed(() => "/" + rest.value);
const data = ref(null);
const error = ref("");
const needPw = ref(false);
const pw = ref("");
const loading = ref(true);

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const key = "sharepass:" + hash.value;
    const saved = pw.value || sessionStorage.getItem(key) || "";
    data.value = await shareApi.list(hash.value, path.value.replace(/\/?$/, "/"), saved);
    if (saved) { pw.value = saved; sessionStorage.setItem(key, saved); }
    needPw.value = false;
  } catch (e) {
    if (e.status === 401 || e.status === 403) { needPw.value = true; error.value = pw.value ? "Senha incorreta." : ""; }
    else error.value = e.status === 404 ? "Este link não existe ou expirou." : e.message;
    data.value = null;
  } finally { loading.value = false; }
}
watch(() => route.fullPath, load);
onMounted(load);

const open = (it) => router.push(`/public/share/${hash.value}${it.path}`);
async function download(it) {
  if (pw.value) { try { await shareApi.downloadWithPassword(hash.value, it.path, it.name, pw.value); } catch (e) { toastError(e); } return; }
  const a = document.createElement("a");
  a.href = shareApi.downloadUrl(hash.value, [it.path], it.type === "directory");
  a.download = it.name;
  document.body.appendChild(a); a.click(); a.remove();
}
const title = computed(() => data.value?.name || "Compartilhado");
const single = computed(() => !!data.value && data.value.type !== "directory");
</script>

<template>
  <div class="share-page">
    <header class="share-top"><Icon name="hdd" /><span>{{ appName }}</span></header>
    <div class="page">
      <div class="wrap">
        <div class="phead"><h1>{{ title }}</h1><div class="crumbs"><span>Link compartilhado</span></div></div>
        <div v-if="loading" class="vazio">Carregando…</div>
        <form v-else-if="needPw" class="card form" @submit.prevent="load">
          <h2>Este link tem senha</h2>
          <div v-if="error" class="lg-erro" role="alert"><Icon name="warning" /><span>{{ error }}</span></div>
          <label>Senha<input v-model="pw" type="password" autocomplete="off" autofocus /></label>
          <button class="btn pri" :disabled="!pw">Abrir</button>
        </form>
        <div v-else-if="error" class="vazio">{{ error }}</div>
        <div v-else-if="data" class="flist">
          <div v-if="single" class="frow" tabindex="0" @click="download({ name: data.name, path: '/', type: data.type })">
            <span class="chk"><Icon :name="iconKey(kindOf(data))" /></span><span class="n">{{ data.name }}</span><span class="m hide-s">{{ date(data.modified) }}</span><span class="s">{{ bytes(data.size) }}</span>
            <button class="ib more" :aria-label="'Baixar ' + data.name" title="Baixar" @click.stop="download({ name: data.name, path: '/', type: data.type })"><Icon name="download" /></button>
          </div>
          <div v-if="rest" class="frow" tabindex="0" @click="router.back()"><span class="chk"><Icon name="back" /></span><span class="n">Voltar</span><span></span><span></span><span></span></div>
          <div v-for="it in data.items" :key="it.path" class="frow" tabindex="0" @click="it.type === 'directory' ? open(it) : download(it)">
            <span class="chk"><Icon :name="iconKey(kindOf(it))" /></span>
            <span class="n">{{ it.name }}</span>
            <span class="m hide-s">{{ date(it.modified) }}</span>
            <span class="s">{{ it.type === "directory" ? "—" : bytes(it.size) }}</span>
            <button class="ib more" :aria-label="'Baixar ' + it.name" title="Baixar" @click.stop="download(it)"><Icon name="download" /></button>
          </div>
          <div v-if="!single && !data.items.length" class="vazio" style="border: 0">Pasta vazia.</div>
        </div>
      </div>
    </div>
    <Toasts />
  </div>
</template>
