import path from "path";
import fs from "fs";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
let appVersion = "1.0.0";
try {
    const wailsJsonPath = path.resolve(__dirname, "../wails.json");
    const wailsJson = JSON.parse(fs.readFileSync(wailsJsonPath, "utf-8"));
    appVersion = wailsJson.info.productVersion;
} catch (_) {
    // wails.json absent en mode web — version par défaut
}
export default defineConfig({
    plugins: [react(), tailwindcss()],
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        },
    },
    define: {
        __APP_VERSION__: JSON.stringify(appVersion),
    },
});
