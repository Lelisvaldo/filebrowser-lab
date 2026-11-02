// Motivo de uma falha ao mover/copiar, em português claro (o servidor devolve textos técnicos em inglês).
export function moveReason(msg) {
  const m = String(msg || "");
  if (/read-only file system|source is read-only|read.?only/i.test(m)) return "Esta pasta está em só leitura. Para gravar numa nuvem, um administrador libera a escrita em Administração → Conectar serviços (ícone de lápis na linha da nuvem).";
  if (/itself/i.test(m)) return "O item já está nessa pasta.";
  if (/exist/i.test(m)) return "Já existe um item com esse nome no destino.";
  if (/no space|disk quota|quota/i.test(m)) return "Sem espaço no destino.";
  if (/permission|forbidden|not allowed/i.test(m)) return "Sem permissão para gravar no destino.";
  return m;
}

// "nome.ext" -> "nome (2).ext", pulando os nomes que já existem (é o que o servidor faz)
export function suggestName(name, taken, isDir) {
  const dot = isDir ? -1 : name.lastIndexOf(".");
  const base = dot > 0 ? name.slice(0, dot) : name;
  const ext = dot > 0 ? name.slice(dot) : "";
  for (let n = 2; n < 1000; n++) {
    const c = `${base} (${n})${ext}`;
    if (!taken.has(c)) return c;
  }
  return name;
}
