import { get, send } from "./client";

export const systemConfig = () => get("settings/config");
export const settings = () => get("settings");
export const userDefaults = () => get("settings/user-defaults");
export const accessRules = (source) => get("access", { source });
export const activity = (params) => get("tools/activity", params);
