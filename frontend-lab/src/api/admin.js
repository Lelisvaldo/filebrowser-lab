import { get, send } from "./client";

export const systemConfig = () => get("settings/config");
export const settings = () => get("settings");
export const userDefaults = () => get("settings/user-defaults");
export const accessRules = (source) => get("access", { source });
export const activity = (params) => get("tools/activity", params);

// Regras de acesso por pasta
export const accessRule = (source, path) => get("access", { source, path });
export const accessRulesAll = (source) => get("access", { source });
export const accessAdd = (source, path, body) => send("access", "POST", body, { source, path });
export const accessDel = (source, path, { allow, ruleCategory, value }) => send("access", "DELETE", undefined, { source, path, ruleType: allow ? "allow" : "deny", ruleCategory, value }, { raw: true });
export const accessGroups = () => get("access/groups");
