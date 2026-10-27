// Converte legendas SRT e ASS/SSA para WebVTT (o <track> do navegador só entende WebVTT).
const pad = (n, w = 2) => String(n).padStart(w, "0");

function fromAss(text) {
  const out = ["WEBVTT", ""];
  let fmt = [];
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim();
    if (/^Format:/i.test(line) && fmt.length === 0 && /Start/i.test(line)) fmt = line.slice(7).split(",").map((x) => x.trim().toLowerCase());
    if (!/^Dialogue:/i.test(line)) continue;
    const parts = line.slice(9).split(",");
    const iS = fmt.indexOf("start") >= 0 ? fmt.indexOf("start") : 1;
    const iE = fmt.indexOf("end") >= 0 ? fmt.indexOf("end") : 2;
    const iT = fmt.indexOf("text") >= 0 ? fmt.indexOf("text") : 9;
    const t = (v) => { const m = /^(\d+):(\d\d):(\d\d)[.](\d\d)/.exec((v || "").trim()); return m ? `${pad(m[1])}:${m[2]}:${m[3]}.${m[4]}0` : null; };
    const a = t(parts[iS]), b = t(parts[iE]);
    if (!a || !b) continue;
    const txt = parts.slice(iT).join(",").replace(/\{[^}]*\}/g, "").replace(/\\N/gi, "\n").replace(/\\h/g, " ").trim();
    if (txt) out.push(`${a} --> ${b}`, txt, "");
  }
  return out.join("\n");
}

export function toVtt(text) {
  const t = String(text || "").replace(/^\uFEFF/, "");
  if (/^WEBVTT/.test(t)) return t;
  if (/^\[Script Info\]/im.test(t) || /^Dialogue:/im.test(t)) return fromAss(t);
  return "WEBVTT\n\n" + t.replace(/(\d\d:\d\d:\d\d),(\d{3})/g, "$1.$2").replace(/\r\n?/g, "\n");
}
