// Ponte com o app Android (android-lab/): só existe quando a página roda dentro dele (window.LabNative).
import { reactive, watchEffect } from "vue";
import { ui, isDark } from "@/store/app";
import { transfers } from "@/store/transfers";

export const isApp = typeof window !== "undefined" && !!window.LabNative;

// arquivos recebidos por "Compartilhar" no Android, esperando uma pasta de destino; send é um sinal para o Browse abrir o seletor
export const nativeShare = reactive({ files: [], send: 0 });

function read() {
  if (!isApp) return;
  try { nativeShare.files = JSON.parse(window.LabNative.shareInfo() || "[]"); } catch (_e) { nativeShare.files = []; }
}
if (isApp) { window.addEventListener("labshare", read); read(); }

export const clearShare = () => { if (isApp) window.LabNative.clearShare(); };
export const sendShared = () => { nativeShare.send++; };
export const getLock = () => (isApp ? window.LabNative.getLock() : false);
export const setLock = (on) => { if (isApp) window.LabNative.setLock(!!on); };
export const changeServer = () => { if (isApp) window.LabNative.changeServer(); };

// as barras do Android (status e navegação) seguem o tema escolhido no site
if (isApp) watchEffect(() => { void ui.theme; window.LabNative.setBars(isDark()); });

// envios em andamento -> o app mostra o progresso numa notificação e fica vivo em segundo plano; avisa quando termina
let wasUploading = false;
let baseDone = 0;
let baseFail = 0;
if (isApp) watchEffect(() => {
  const ups = transfers.items.filter((t) => t.kind === "up");
  const act = ups.filter((t) => ["queued", "running", "paused"].includes(t.status));
  const done = ups.filter((t) => t.status === "done").length;
  const fail = ups.filter((t) => t.status === "error").length;
  if (act.length) {
    if (!wasUploading) { wasUploading = true; baseDone = done; baseFail = fail; }
    const total = act.reduce((a, t) => a + (t.size || 0), 0);
    const loaded = act.reduce((a, t) => a + (t.loaded || 0), 0);
    const pct = total ? Math.floor((loaded / total) * 100) : 0;
    window.LabNative.uploadState(act.length, pct, act[0].name, act.every((t) => t.status === "paused"));
  } else if (wasUploading) {
    wasUploading = false;
    window.LabNative.uploadDone(Math.max(0, done - baseDone), Math.max(0, fail - baseFail));
  }
});
