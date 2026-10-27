<script setup>
// Editor de código com o motor do VS Code (Monaco), carregado só quando um arquivo de texto é aberto.
import { ref, watch, onMounted, onBeforeUnmount } from "vue";
import { isDark } from "@/store/app";

const props = defineProps({ modelValue: { type: String, default: "" }, name: { type: String, default: "" }, readonly: Boolean });
const emit = defineEmits(["update:modelValue", "save"]);
const el = ref(null);
const failed = ref(false);
let monaco = null, ed = null, applying = false;

const LANG = {
  js: "javascript", mjs: "javascript", cjs: "javascript", jsx: "javascript", ts: "typescript", tsx: "typescript", json: "json", md: "markdown", markdown: "markdown",
  py: "python", go: "go", rs: "rust", c: "c", h: "c", cpp: "cpp", cc: "cpp", hpp: "cpp", java: "java", sh: "shell", bash: "shell", zsh: "shell", sql: "sql",
  yml: "yaml", yaml: "yaml", xml: "xml", html: "html", htm: "html", vue: "html", css: "css", scss: "scss", less: "less", php: "php", rb: "ruby", lua: "lua",
  ini: "ini", conf: "ini", cfg: "ini", env: "ini", toml: "ini", cs: "csharp", kt: "kotlin", swift: "swift", ps1: "powershell", bat: "bat", dockerfile: "dockerfile",
};
function langOf(name) {
  const n = String(name || "").toLowerCase();
  if (n === "dockerfile") return "dockerfile";
  const e = n.includes(".") ? n.split(".").pop() : "";
  return LANG[e] || "plaintext";
}
const themeName = () => (isDark() ? "lab-dark" : "lab-light");
function cssColor(v, fallback) { const c = getComputedStyle(document.documentElement).getPropertyValue(v).trim(); return /^#[0-9a-f]{6}$/i.test(c) ? c : fallback; }

onMounted(async () => {
  try {
    const [m, w] = await Promise.all([import("monaco-editor/esm/vs/editor/edcore.main"), import("monaco-editor/esm/vs/editor/editor.worker?worker")]);
    await import("monaco-editor/esm/vs/basic-languages/monaco.contribution");
    self.MonacoEnvironment = { getWorker: () => new w.default() };
    monaco = m;
    monaco.editor.defineTheme("lab-dark", { base: "vs-dark", inherit: true, rules: [], colors: { "editor.background": cssColor("--card", "#1a1033") } });
    monaco.editor.defineTheme("lab-light", { base: "vs", inherit: true, rules: [], colors: { "editor.background": cssColor("--card", "#ffffff") } });
    ed = monaco.editor.create(el.value, {
      value: props.modelValue, language: langOf(props.name), theme: themeName(), readOnly: props.readonly, automaticLayout: true,
      fontSize: 14, minimap: { enabled: true }, scrollBeyondLastLine: false, wordWrap: "off", renderWhitespace: "selection", tabSize: 2, smoothScrolling: true, padding: { top: 8 },
    });
    ed.onDidChangeModelContent(() => { if (!applying) emit("update:modelValue", ed.getValue()); });
    ed.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => emit("save"));
    ed.focus();
  } catch (e) { failed.value = true; }
});
watch(() => props.modelValue, (v) => { if (ed && v !== ed.getValue()) { applying = true; ed.setValue(v); applying = false; } });
watch(() => props.name, (n) => { if (ed) monaco.editor.setModelLanguage(ed.getModel(), langOf(n)); });
watch(() => props.readonly, (r) => ed && ed.updateOptions({ readOnly: r }));
onBeforeUnmount(() => { if (ed) ed.dispose(); });
defineExpose({ failed });
</script>

<template>
  <div class="code-ed"><div ref="el" class="code-host" aria-label="Editor de código"></div></div>
</template>
