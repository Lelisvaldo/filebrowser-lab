import { createRouter, createWebHistory } from "vue-router";
import { baseURL } from "@/lib/config";
import { app, loadSession } from "@/store/app";

const routes = [
  { path: "/login", name: "login", component: () => import("@/views/Login.vue"), meta: { public: true } },
  { path: "/public/share/:hash/:rest(.*)?", name: "share", component: () => import("@/views/PublicShare.vue"), meta: { public: true } },
  { path: "/", name: "home", component: () => import("@/views/Overview.vue") },
  { path: "/f/:source/:rest(.*)?", name: "browse", component: () => import("@/views/Browse.vue") },
  { path: "/busca", name: "search", component: () => import("@/views/Search.vue") },
  { path: "/ajustes/:tab?", name: "account", component: () => import("@/views/Account.vue") },
  { path: "/admin/:tab?", name: "admin", component: () => import("@/views/Admin.vue") },
  { path: "/:pathMatch(.*)*", redirect: "/" },
];

export const router = createRouter({ history: createWebHistory(baseURL), routes });

router.beforeEach(async (to) => {
  if (!app.ready) await loadSession();
  if (to.meta.public) return app.user && to.name === "login" ? "/" : true;
  if (!app.user) return { name: "login", query: to.fullPath !== "/" ? { next: to.fullPath } : {} };
  return true;
});
