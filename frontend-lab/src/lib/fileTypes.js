import { ext } from "./paths";
import { hueOf } from "./sourceIcons";

const BY_EXT = {
  image: "jpg jpeg png gif webp bmp svg avif heic ico tiff",
  video: "mp4 mkv avi mov webm m4v wmv flv ts mpg mpeg",
  audio: "mp3 flac wav ogg m4a aac opus wma",
  pdf: "pdf",
  zip: "zip rar 7z tar gz bz2 xz tgz zst iso",
  code: "js ts jsx tsx vue json yml yaml toml xml html css scss py go rs c h cpp java sh bash sql ini conf cfg env lua php rb",
  text: "txt md log csv nfo srt ass rtf",
  table: "xls xlsx ods csv tsv",
};

const MAP = {};
for (const [kind, list] of Object.entries(BY_EXT)) for (const e of list.split(" ")) MAP[e] ??= kind;

// "image" | "video" | "audio" | "pdf" | "zip" | "code" | "text" | "table" | "file" | "folder"
export function kindOf(item) {
  if (item.type === "directory") return "folder";
  const t = String(item.type || "");
  if (t.startsWith("image/")) return "image";
  if (t.startsWith("video/")) return "video";
  if (t.startsWith("audio/")) return "audio";
  if (t === "application/pdf") return "pdf";
  return MAP[ext(item.name)] || (t.startsWith("text/") ? "text" : "file");
}

const ICON = { folder: "folder", image: "image", video: "video", audio: "music", pdf: "pdf", zip: "zip", code: "code", text: "text", table: "table", file: "file" };
const HUE = { folder: "orange", image: "magenta", video: "violet", audio: "pink", pdf: "red", zip: "yellow", code: "cyan", text: "blue", table: "green" };
/** Classe de cor do ícone por tipo de arquivo; pastas herdam a cor da pasta raiz (source). */
export const hueClass = (kind, source) => (kind === "folder" && source ? hueOf(source) : HUE[kind] ? "hue h-" + HUE[kind] : "");
export const iconKey = (kind) => ICON[kind] || "file";

export const canView = (kind) => ["image", "video", "audio", "pdf", "code", "text"].includes(kind);
export const isEditable = (kind) => kind === "code" || kind === "text";
