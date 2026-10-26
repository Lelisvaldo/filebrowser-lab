<script setup>
import { computed } from "vue";
import { ICONS } from "@/lib/icons";

const props = defineProps({ name: { type: String, required: true } });
const inner = computed(() => ICONS[props.name] || ICONS.file || "");
</script>

<template>
  <svg class="i f" viewBox="0 0 24 24" aria-hidden="true" v-html="inner"></svg>
</template>
