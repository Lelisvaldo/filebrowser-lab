<script setup>
import Icon from "./Icon.vue";
import { visible, pause, resume, cancel, dismiss, eta } from "@/store/transfers";
import { bytes, speed, duration } from "@/lib/format";

defineProps({ compact: { type: Boolean, default: false } });
const mv = (t) => t.kind === "move" || t.kind === "copy";
const pc = (t) => mv(t) ? (t.status === "running" || t.status === "done" ? 100 : 0) : (t.size > 0 ? Math.min(100, Math.floor((t.loaded / t.size) * 100)) : t.status === "done" ? 100 : 0);
const text = (t) => {
  if (t.status === "done") return "Concluído";
  if (mv(t) && t.status === "running") return t.kind === "move" ? "Movendo…" : "Copiando…";
  if (t.status === "error") return t.error || "Falhou";
  if (t.status === "queued") return "Na fila";
  if (t.status === "paused") return pc(t) + "% · pausado";
  const e = eta(t);
  return pc(t) + "%" + (t.speed ? " · " + speed(t.speed) : "") + (isFinite(e) ? " · faltam " + duration(e) : "");
};
</script>

<template>
  <div class="tlist" :class="{ compact }">
    <template v-for="t in visible()" :key="t.id">
      <div v-if="compact" class="tc2" :class="{ ok: t.status === 'done' }" :title="mv(t) ? t.label : t.source + t.path">
        <div class="tn">{{ mv(t) ? (t.kind === "move" ? "Mover: " : "Copiar: ") : "" }}{{ t.name }}</div>
        <div class="tr2">
          <div class="bar" :class="{ anim: t.status === 'running', cheio: t.status === 'error' }"><i :style="{ width: pc(t) + '%' }"></i></div>
          <span class="meta">{{ text(t) }}</span>
          <div class="acts">
            <button v-if="t.status === 'running' && !mv(t)" class="ib" :aria-label="'Pausar ' + t.name" title="Pausar" @click="pause(t)"><Icon name="pause" /></button>
            <button v-else-if="t.status === 'paused'" class="ib" :aria-label="'Retomar ' + t.name" title="Retomar" @click="resume(t)"><Icon name="play" /></button>
            <button v-if="['running', 'paused', 'queued'].includes(t.status) && !mv(t)" class="ib" :aria-label="'Cancelar ' + t.name" title="Cancelar" @click="cancel(t)"><Icon name="dismiss" /></button>
            <button v-else-if="t.status !== 'running'" class="ib" aria-label="Dispensar" title="Dispensar" @click="dismiss(t)"><Icon name="dismiss" /></button>
          </div>
        </div>
      </div>
      <div v-else class="tr" :class="{ ok: t.status === 'done' }">
        <div class="ico"><Icon :name="mv(t) ? 'transfer' : 'upload'" /></div>
        <div style="min-width: 0">
          <div class="nm">{{ t.name }}</div>
          <div class="meta">{{ mv(t) ? t.label : t.source + t.dir + " · " + bytes(t.size) }}</div>
        </div>
        <div class="acts">
          <button v-if="t.status === 'running' && !mv(t)" class="ib" :aria-label="'Pausar ' + t.name" title="Pausar" @click="pause(t)"><Icon name="pause" /></button>
          <button v-else-if="t.status === 'paused'" class="ib" :aria-label="'Retomar ' + t.name" title="Retomar" @click="resume(t)"><Icon name="play" /></button>
          <button v-if="['running', 'paused', 'queued'].includes(t.status) && !mv(t)" class="ib" :aria-label="'Cancelar ' + t.name" title="Cancelar" @click="cancel(t)"><Icon name="dismiss" /></button>
          <button v-else-if="t.status !== 'running'" class="ib" aria-label="Dispensar" title="Dispensar" @click="dismiss(t)"><Icon name="dismiss" /></button>
        </div>
        <div class="bar" :class="{ anim: t.status === 'running', cheio: t.status === 'error' }"><i :style="{ width: pc(t) + '%' }"></i></div>
        <div class="meta">{{ text(t) }}</div>
      </div>
    </template>
    <div v-if="!visible().length" class="tvazio vi"><Icon name="transfer" /><span>Nenhuma transferência em andamento.</span></div>
  </div>
</template>
