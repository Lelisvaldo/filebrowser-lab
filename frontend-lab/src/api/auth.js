import { request, get, send, ApiError } from "./client";

// Login por senha. Lança ApiError; status 403 + "password change" = troca obrigatória.
export async function login(username, password, otp = "") {
  try {
    await request("auth/login", {
      method: "POST",
      params: { username },
      headers: { "X-Password": encodeURIComponent(password), "X-Secret": otp },
      raw: true,
      quiet401: true,
    });
  } catch (e) {
    if (e instanceof ApiError) {
      const m = String(e.message).toLowerCase();
      if (e.status === 403 && m.includes("password change")) e.code = "change-password";
      else if (e.status === 403 && m.includes("passkey")) e.code = "passkey";
      else if (m.includes("otp") || m.includes("totp") || m.includes("two-factor") || m.includes("2fa")) e.code = "otp";
    }
    throw e;
  }
}

export async function changeRequiredPassword(username, current, next, otp = "") {
  await request("auth/password/change-required", {
    method: "POST",
    params: { username },
    headers: { "X-Password": encodeURIComponent(current), "X-Secret": otp },
    body: { password: next, passwordConfirm: next },
    raw: true,
    quiet401: true,
  });
}

export const logout = () => request("auth/logout", { method: "POST", raw: true, quiet401: true }).catch(() => {});
export const renew = () => request("auth/renew", { method: "POST", raw: true, quiet401: true });
export const me = () => get("users", { username: "self" }, { quiet401: true });

// Usuários (administração)
export const listUsers = () => get("users");
export const getUser = (username) => get("users", { username });
export const createUser = (data, headers) => send("users", "POST", { which: [], data }, undefined, { headers, raw: true });
export const updateUser = (username, which, data, headers) => send("users", "PATCH", { which, data }, { username }, { headers, raw: true });
export const deleteUser = (username, headers) => send("users", "DELETE", undefined, { username }, { headers, raw: true });

// 2FA por aplicativo (TOTP)
export const otpGenerate = (username, password) =>
  request("auth/otp/generate", { method: "POST", params: { username }, headers: { "X-Password": encodeURIComponent(password) } });
export const otpVerify = (username, password, code) =>
  request("auth/otp/verify", { method: "POST", params: { username }, headers: { "X-Password": encodeURIComponent(password), "X-Secret": code }, raw: true });

// Chaves de API
export const listApiKeys = () => get("auth/token/list");
export const createApiKey = (params) => request("auth/token", { method: "POST", params });
export const deleteApiKey = (name) => request("auth/token", { method: "DELETE", params: { name }, raw: true });

// Preferências do usuário (viewMode, ordenação, etc.)
export const updatePrefs = (username, which, data) => updateUser(username, which, data);

// Chave de acesso (passkey) como segundo fator: o servidor usa WebAuthn; aqui só converte base64url <-> binário.
const dec = (v) => { const t = String(v).replace(/-/g, "+").replace(/_/g, "/"); return Uint8Array.from(atob(t.padEnd(Math.ceil(t.length / 4) * 4, "=")), (c) => c.charCodeAt(0)).buffer; };
const enc = (b) => btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
function toCreation(pk) { const o = { ...pk, challenge: dec(pk.challenge), user: { ...pk.user, id: dec(pk.user.id) } }; if (pk.excludeCredentials) o.excludeCredentials = pk.excludeCredentials.map((c) => ({ ...c, id: dec(c.id) })); return o; }
function toRequest(pk) { const o = { ...pk, challenge: dec(pk.challenge) }; if (pk.allowCredentials) o.allowCredentials = pk.allowCredentials.map((c) => ({ ...c, id: dec(c.id) })); return o; }
function credJson(c) {
  const r = c.response;
  const response = r.attestationObject
    ? { attestationObject: enc(r.attestationObject), clientDataJSON: enc(r.clientDataJSON), transports: r.getTransports ? r.getTransports() : [] }
    : { authenticatorData: enc(r.authenticatorData), clientDataJSON: enc(r.clientDataJSON), signature: enc(r.signature), userHandle: r.userHandle ? enc(r.userHandle) : null };
  return { id: c.id, rawId: enc(c.rawId), type: c.type, authenticatorAttachment: c.authenticatorAttachment, response, clientExtensionResults: c.getClientExtensionResults ? c.getClientExtensionResults() : {} };
}
export const passkeySupported = () => typeof window !== "undefined" && !!window.PublicKeyCredential && !!navigator.credentials;
export async function passkeyRegister(name, password) {
  const headers = password ? { "X-Password": encodeURIComponent(password) } : {};
  const r = await request("auth/webauthn/begin-register", { method: "POST", headers });
  const cred = await navigator.credentials.create({ publicKey: toCreation(r.publicKey) });
  await request("auth/webauthn/finish-register", { method: "POST", params: { session_id: r.sessionID, name }, body: credJson(cred), headers, raw: true });
}
export async function passkeyLogin(username, password) {
  const r = await request("auth/webauthn/begin-login", { method: "POST", params: { username }, headers: { "X-Password": encodeURIComponent(password) }, quiet401: true });
  const cred = await navigator.credentials.get({ publicKey: toRequest(r.publicKey) });
  await request("auth/webauthn/finish-login", { method: "POST", params: { session_id: r.sessionID }, body: credJson(cred), raw: true, quiet401: true });
}
export const passkeyDelete = (id, password) => request("auth/webauthn/" + encodeURIComponent(id), { method: "DELETE", headers: password ? { "X-Password": encodeURIComponent(password) } : {}, raw: true });
export const passkeyList = () => get("lab/passkeys");
