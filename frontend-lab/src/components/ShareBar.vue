<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import Icon from "./Icon.vue";
import { nativeShare, clearShare, sendShared } from "@/store/native";
import { bytes } from "@/lib/format";

const route = useRoute();
const n = computed(() => nativeShare.files.length);
const inFolder = computed(() => route.name === "browse");
const total = computed(() => nativeShare.files.reduce((s, f) => s + Math.max(0, f.size || 0), 0));
</script>

<template>
  <div v-if="n" class="sharebar" role="status">
    <Icon name="upload" />
    <span class="st">
      <b>{{ n }} {{ n === 1 ? "arquivo" : "arquivos" }} do celular</b>
      <small v-if="total"> · {{ bytes(total) }}</small>
      <span class="sh">{{ inFolder ? "Escolha a pasta e toque em Enviar." : "Abra a pasta de destino e toque em Enviar." }}</span>
    </span>
    <button v-if="inFolder" class="btn pri" type="button" @click="sendShared">Enviar para esta pasta</button>
    <button class="btn" type="button" @click="clearShare">Cancelar</button>
  </div>
</template>
