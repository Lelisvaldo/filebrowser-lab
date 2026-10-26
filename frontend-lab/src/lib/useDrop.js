// Lê arquivos soltos (inclusive pastas) de um evento de arrastar e soltar.
async function walk(entry, prefix, out) {
  if (entry.isFile) {
    const file = await new Promise((res, rej) => entry.file(res, rej));
    out.push({ file, rel: prefix + entry.name });
  } else if (entry.isDirectory) {
    const reader = entry.createReader();
    let batch;
    do {
      batch = await new Promise((res, rej) => reader.readEntries(res, rej));
      for (const e of batch) await walk(e, prefix + entry.name + "/", out);
    } while (batch.length);
  }
}

export async function filesFromDrop(dt) {
  const out = [];
  const entries = [...(dt.items || [])].map((i) => (i.webkitGetAsEntry ? i.webkitGetAsEntry() : null)).filter(Boolean);
  if (entries.length) {
    for (const e of entries) await walk(e, "", out);
    return out;
  }
  return [...(dt.files || [])].map((file) => ({ file, rel: file.name }));
}

export function filesFromInput(input) {
  return [...input.files].map((file) => ({ file, rel: file.webkitRelativePath || file.name }));
}
