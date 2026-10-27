import { get, send, request, apiPath } from "./client";
import { join } from "@/lib/paths";

function norm(data) {
  const base = data.path === "/" ? "/" : data.path;
  const folders = (data.folders || []).map((f) => ({ ...f, type: "directory" }));
  const files = data.files || [];
  const items = [...folders, ...files].map((it) => ({
    ...it,
    source: data.source,
    path: join(base, it.name, it.type === "directory"),
  }));
  return { ...data, folders: undefined, files: undefined, items };
}

export async function list(source, path) {
  return norm(await get("resources", { source, path }));
}

export const info = (source, path, content = false) => get("resources", { source, path, content: content ? "true" : undefined });
export const mediaMeta = (source, path) => get("resources", { source, path, metadata: "true" });
export const subtitleUrl = (source, path, name, embedded) => apiPath("media/subtitles", { source, path, name, embedded: embedded ? "true" : "false" });
export const sources = () => get("settings/sources");
export const mkdir = (source, path) => send("resources", "POST", undefined, { source, path, isDir: "true", override: "false" }, { raw: true });
export const writeText = (source, path, text) => send("resources", "PUT", text, { source, path }, { raw: true });
export const remove = (items) => send("resources/bulk", "DELETE", items.map((i) => ({ source: i.source, path: i.path })), undefined, { raw: true });

export async function moveCopy(items, action, { overwrite = false, rename = false } = {}) {
  return send("resources", "PATCH", {
    items: items.map((i) => ({ fromSource: i.fromSource, fromPath: i.from, toSource: i.toSource, toPath: i.to })),
    action, overwrite, rename,
  });
}

// scope (opcional): "Fonte:/subpasta/"
export const search = (query, sources, scope) => get("tools/search", { query, sources: sources.join(","), scope });

// URLs diretas (o navegador envia o cookie de sessão sozinho)
export const downloadUrl = (source, paths, { inline = false, archive = false } = {}) =>
  apiPath("resources/download", { source, file: paths, inline: inline ? "true" : undefined, algo: archive || paths.length > 1 ? "zip" : undefined });
export const viewUrl = (source, path, viewToken) => apiPath("resources/view", { source, file: path, viewToken });
export const streamUrl = (source, path, viewToken) => apiPath("media/stream", { source, file: path, viewToken });
export const previewUrl = (source, path, modified) =>
  apiPath("resources/preview", { source, path, inline: "true", key: modified ? Date.parse(modified) : undefined });

export const viewToken = async (source, path) => {
  const d = await info(source, path);
  return d.viewToken || "";
};

// Compartilhamentos
export const shares = () => get("share/list");
// editar opções: POST com o hash (o PATCH do servidor só troca o caminho)
export const updateShare = (body) => send("share", "POST", body);
export const createShare = (body) => send("share", "POST", body);
export const deleteShare = (hash) => send("share", "DELETE", undefined, { hash }, { raw: true });
export const shareFor = (source, path) => get("share", { source, path });
export const fileChecksum = (source, path, algo) => get("resources", { source, path, checksum: algo });
export { request };

// Compactar / extrair no servidor
export const archive = (source, paths, destination) => send("resources/archive", "POST", { fromSource: source, paths, destination, format: "zip" });
export const unarchive = (source, path, destination) => send("resources/unarchive", "POST", { fromSource: source, path, destination });

// Maiores arquivos dentro de uma pasta (minMb em MB); devolve do maior para o menor
export const largest = (source, path, minMb) =>
  get("tools/search", { query: `type:largerThan=${minMb} type:file`, sources: source, scope: `${source}:${path}`, largest: "true" });
