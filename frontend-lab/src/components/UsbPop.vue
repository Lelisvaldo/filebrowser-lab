<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from "vue";
import { useRouter } from "vue-router";
import Icon from "./Icon.vue";
import { app, ui, setUi, eject, ejecting } from "@/store/app";
import { routeFor } from "@/lib/paths";
import { bytes } from "@/lib/format";

const router = useRouter();
const el = ref(null);
const close = () => setUi({ usbOpen: false });
function away(e) {
  if (!ui.usbOpen) return;
  if (el.value && !el.value.contains(e.target) && !e.target.closest('[aria-label^="Pendrives"]')) close();
}
onMounted(() => document.addEventListener("mousedown", away));
onBeforeUnmount(() => document.removeEventListener("mousedown", away));

const list = computed(() => app.usb || []);
const pct = (d) => (d.size && d.free != null ? Math.min(100, Math.round(((d.size - d.free) / d.size) * 100)) : 0);
function open(d) {
  if (!d.mounted) return;
  close();
  void router.push(routeFor("Pendrives", "/" + d.name + "/"));
}
</script>

<template>
  <div v-if="ui.usbOpen" ref="el" class="pop usbpop" role="dialog" aria-label="Pendrives conectados" @keydown.esc="close">
    <header>Pendrives <span v-if="list.length" class="badge ok">{{ list.filter((d) => d.mounted).length }}</span></header>
    <div class="ulist">
      <div v-if="!list.length" class="sem-usb"><Icon name="usb" /><span>Nenhum conectado</span></div>
      <div v-for="d in list" :key="d.uuid" class="urowp" :class="{ off: !d.mounted }">
        <button class="ubody" type="button" :disabled="!d.mounted" @click="open(d)">
          <span class="ico sm hue" :class="d.mounted ? 'h-green' : 'h-red'"><Icon name="usb" /></span>
          <span class="uinfo">
            <span class="un">{{ d.name }}</span>
            <span class="us" v-if="d.mounted && d.size">{{ bytes(d.size - (d.free || 0)) }} de {{ bytes(d.size) }} · {{ d.filesystem }}</span>
            <span class="us" v-else-if="!d.mounted">Desmontado: tire e espete de novo</span>
            <span v-if="d.mounted && d.size" class="bar" :class="{ cheio: pct(d) >= 90 }"><i :style="{ width: pct(d) + '%' }"></i></span>
          </span>
        </button>
        <button v-if="d.mounted && d.ejectable" class="ib" type="button" :disabled="ejecting[d.uuid]" :aria-label="'Remover ' + d.name" title="Remover com segurança" @click="eject(d)"><Icon name="eject" /></button>
      </div>
    </div>
  </div>
</template>
