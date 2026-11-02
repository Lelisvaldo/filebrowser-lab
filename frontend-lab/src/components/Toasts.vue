<script setup>
import { computed } from "vue";
import { app, ui, closeToast } from "@/store/app";
import Icon from "./Icon.vue";

const ICON = { ok: "check", info: "info", err: "warning" };
// com um painel do topo aberto (transferências, pendrives, conta), os avisos descem para o meio da lateral direita
const side = computed(() => ui.transfersOpen || ui.usbOpen || ui.userOpen);
</script>

<template>
  <div class="toasts" :class="{ side }" aria-live="polite">
    <div v-for="t in app.toasts" :key="t.id" class="toast" :class="'k-' + t.kind" :role="t.kind === 'err' ? 'alert' : 'status'">
      <Icon :name="ICON[t.kind] || 'check'" /><span>{{ t.text }}</span>
      <button class="tx" type="button" aria-label="Fechar aviso" @click="closeToast(t.id)"><Icon name="x" /></button>
      <i class="tcount" :style="{ animationDuration: t.ms + 'ms' }" @animationend="closeToast(t.id)"></i>
    </div>
  </div>
</template>
