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
