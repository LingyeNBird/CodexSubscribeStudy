import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
export default defineConfig(({ command }) => {
  const previewFile = command === "serve" ? process.env.SURVEY_PREVIEW_FILE : undefined;
  const snapshot = previewFile
    ? JSON.stringify(JSON.parse(readFileSync(resolve(previewFile), "utf8")))
    : undefined;
  return {
    plugins: [
      vue(),
      {
        name: "local-survey-preview",
        apply: "serve",
        configureServer(server) {
          if (snapshot === undefined) return;
          server.middlewares.use((request, response, next) => {
            if (request.method !== "GET" || request.url?.split("?")[0] !== "/api/survey/statistics")
              return next();
            response.setHeader("Content-Type", "application/json; charset=utf-8");
            response.setHeader("Cache-Control", "no-store");
            response.setHeader("X-Survey-Preview", "snapshot");
            response.end(snapshot);
          });
        },
      },
    ],
    server: { proxy: { "/api": "http://127.0.0.1:8080" } },
  };
});
