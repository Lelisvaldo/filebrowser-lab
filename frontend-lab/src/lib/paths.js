// Caminhos dentro de uma fonte: sempre começam com "/" e pastas terminam com "/".
export function join(dir, name, isDir) {
  const d = dir.endsWith("/") ? dir : dir + "/";
  return d + name + (isDir ? "/" : "");
}

export function parent(path) {
  const t = path.replace(/\/+$/, "");
  const i = t.lastIndexOf("/");
  return i <= 0 ? "/" : t.slice(0, i + 1);
}

export function basename(path) {
  const t = path.replace(/\/+$/, "");
  return t.slice(t.lastIndexOf("/") + 1);
}

export function ext(name) {
  const i = name.lastIndexOf(".");
  return i > 0 ? name.slice(i + 1).toLowerCase() : "";
}

export function crumbs(path) {
  const parts = path.split("/").filter(Boolean);
  let acc = "/";
  return parts.map((p) => {
    acc += p + "/";
    return { name: p, path: acc };
  });
}

// URL da rota do app para uma pasta de uma fonte.
export function routeFor(source, path) {
  const segs = path.split("/").filter(Boolean).map(encodeURIComponent);
  return "/f/" + encodeURIComponent(source) + (segs.length ? "/" + segs.join("/") + "/" : "/");
}
