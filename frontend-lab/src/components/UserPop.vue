<script setup>
import { onMounted, onBeforeUnmount, ref, computed } from "vue";
import { useRouter } from "vue-router";
import Icon from "./Icon.vue";
import { app, ui, setUi, doLogout } from "@/store/app";

const router = useRouter();
const el = ref(null);
const close = () => setUi({ userOpen: false });
function away(e) {
  if (!ui.userOpen) return;
  if (el.value && !el.value.contains(e.target) && !e.target.closest('[aria-label^="Conta"]')) close();
}
onMounted(() => document.addEventListener("mousedown", away));
onBeforeUnmount(() => document.removeEventListener("mousedown", away));
const name = computed(() => (app.user && app.user.username) || "");

function config() { setUi({ userOpen: false, settingsOpen: true }); }
function conta() { close(); void router.push("/ajustes"); }
async function sair() {
  close();
  await doLogout();
  router.replace("/login");
}
</script>

<template>
  <div v-if="ui.userOpen" ref="el" class="pop userpop" :class="{ fromside: ui.userFrom === 'side' }" role="menu" aria-label="Menu da conta" @keydown.esc="close">
    <div class="mu">{{ name }}</div>
    <button class="mi" role="menuitem" type="button" @click="conta"><Icon name="person" /><span>Minha conta</span></button>
    <button class="mi" role="menuitem" type="button" @click="config"><Icon name="gear" /><span>Configurações</span></button>
    <button class="mi danger" role="menuitem" type="button" @click="sair"><Icon name="signout" /><span>Sair</span></button>
  </div>
</template>
