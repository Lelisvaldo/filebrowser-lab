import { askText } from "@/store/dialogs";

// Ações sensíveis pedem a senha atual (cabeçalho X-Password). Devolve os cabeçalhos ou null se cancelar.
export async function passwordHeaders(message = "Digite a sua senha para confirmar.") {
  const pw = await askText({ title: "Confirmar com a sua senha", message, label: "Senha atual", confirm: "Confirmar", extra: { password: true } });
  return pw ? { "X-Password": encodeURIComponent(pw) } : null;
}
