<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import Icon from "./Icon.vue";
import { stage, summary, clearStage, sendStage, askFiles, askFolder } from "@/store/stage";
import { bytes } from "@/lib/format";
import { isApp } from "@/store/native";

const route = useRoute();
const inFolder = computed(() => route.name === "browse");
const plural = (n, a, b) => n + " " + (n === 1 ? a : b);
</script>

<template>
  <div v-if="summary.any" class="sharebar" role="status">
    <Icon name="upload" />
    <span class="st">
      <b>Para enviar: <template v-if="summary.folders">{{ plural(summary.folders, "pasta", "pastas") }}<template v-if="summary.files"> · </template></template><template v-if="summary.files">{{ plural(summary.files, "arquivo", "arquivos") }}</template></b>
      <small v-if="summary.size"> · {{ bytes(summary.size) }}</small>
      <span class="sh">{{ inFolder ? "As pastas são recriadas no servidor com tudo que está dentro." : "Abra a pasta de destino no servidor." }}</span>
    </span>
    <button v-if="inFolder" class="btn pri" type="button" @click="sendStage">Enviar para esta pasta</button>
    <button v-if="isApp" class="btn" type="button" @click="askFiles">+ Adicionar</button>
    <template v-else>
      <button class="btn" type="button" @click="askFiles">+ Arquivos</button>
      <button class="btn" type="button" @click="askFolder">+ Pasta</button>
    </template>
    <button class="btn" type="button" @click="clearStage">Cancelar</button>
  </div>
</template>
