import path from "node:path";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

// O backend em Go serve esta pasta (internal/web/dist) e preenche public/index.html como template.
const backendWebDist = process.env.LAB_OUT || path.resolve(__dirname, "../backend/internal/web/dist");

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { "@": path.resolve(__dirname, "src") } },
  base: "",
  build: {
    outDir: backendWebDist,
    emptyOutDir: true,
    target: "es2022",
    sourcemap: false,
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      input: { index: path.resolve(__dirname, "./public/index.html") },
    },
  },
  experimental: {
    renderBuiltUrl(filename, { hostType }) {
      if (hostType === "html") return `{{ .htmlVars.staticURL }}/${filename}`;
      return { relative: true };
    },
  },
});
