// Lista de envio: o que o usuário escolheu (arquivos e pastas, do aparelho ou do computador) e ainda não foi enviado.
// Um botão só ("Enviar") junta tudo; a estrutura de pastas é recriada no servidor, inclusive pastas vazias.
import { reactive, computed } from "vue";
import { nativeShare, clearShare as clearNative } from "./native";

// items: { file, rel } (arquivo) ou { dir: true, rel } (pasta, vazia ou não)
export const stage = reactive({ items: [], send: 0, askFiles: 0, askFolder: 0 });

export function addStage(list) {
  const seen = new Set(stage.items.map((i) => (i.dir ? "d:" : "f:") + i.rel + ":" + (i.file ? i.file.size : "")));
  for (const it of list) {
    const k = (it.dir ? "d:" : "f:") + it.rel + ":" + (it.file ? it.file.size : "");
    if (!seen.has(k)) { seen.add(k); stage.items.push(it); }
  }
}

export function clearStage() {
  stage.items = [];
  clearNative();
}

const top = (rel) => String(rel || "").split("/")[0];

export const summary = computed(() => {
  const files = stage.items.filter((i) => !i.dir).length + nativeShare.files.length;
  const tops = new Set();
  for (const i of stage.items) if (i.dir || (i.rel || "").includes("/")) tops.add(top(i.rel));
  for (const f of nativeShare.files) if ((f.rel || "").includes("/")) tops.add(top(f.rel));
  for (const d of nativeShare.dirs) tops.add(top(d));
  const size = stage.items.reduce((s, i) => s + (i.file ? i.file.size : 0), 0) + nativeShare.files.reduce((s, f) => s + Math.max(0, f.size || 0), 0);
  return { files, folders: tops.size, size, any: files > 0 || tops.size > 0 || nativeShare.dirs.length > 0 };
});

export const sendStage = () => { stage.send++; };
export const askFiles = () => { stage.askFiles++; };
export const askFolder = () => { stage.askFolder++; };
