// Ponte com o app Android (android-lab/): só existe quando a página roda dentro dele (window.LabNative).
import { reactive, watchEffect } from "vue";
import { ui, isDark } from "@/store/app";

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
