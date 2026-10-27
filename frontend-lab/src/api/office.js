// Documentos do Office de qualquer geração (LibreOffice no servidor).
import { send, apiPath, request } from "./client";

export const pdfUrl = (source, path) => apiPath("lab/office/pdf", { source, path });
export async function pdfBlob(source, path) {
  const res = await request("lab/office/pdf", { params: { source, path }, raw: true });
  return res.blob();
}
export const convert = (source, path, to) => send("lab/office/convert", "POST", { source, path, to });

// Formatos de destino por família (o servidor valida de novo).
const W = ["pdf", "docx", "odt"], C = ["pdf", "xlsx", "ods"], I = ["pdf", "pptx", "odp"];
const FAM = {};
for (const e of "doc dot docx docm dotx odt ott rtf wpd wps".split(" ")) FAM[e] = W;
for (const e of "xls xlt xlsx xlsm xltx xlsb ods ots".split(" ")) FAM[e] = C;
for (const e of "ppt pps pot pptx pptm ppsx potx odp otp".split(" ")) FAM[e] = I;
export const convertTargets = (extension) => (FAM[String(extension).toLowerCase()] || []).filter((t) => t !== String(extension).toLowerCase());
