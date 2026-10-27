<script setup>
import { computed } from "vue";
import qrcode from "qrcode-generator";
const props = defineProps({ text: String });
const svg = computed(() => { try { const q = qrcode(0, "M"); q.addData(props.text || ""); q.make(); return q.createSvgTag({ cellSize: 5, margin: 2, scalable: true }); } catch (_e) { return ""; } });
</script>
<template><div class="qr" role="img" aria-label="QR code" v-html="svg"></div></template>
