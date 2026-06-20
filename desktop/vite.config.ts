import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({
  plugins: [react()],
  clearScreen: false,
  resolve: { dedupe: ["react", "react-dom"] },
  // Pre-bundle the SDK (and exclude the wasm-loader deps that break vite's
  // optimizer) at server startup, so the webview never loads mid-optimization
  // and hits invalidated chunks.
  optimizeDeps: { include: ["@hanzo/ai/desktop"], exclude: ["pyodide", "sql.js"] },
  server: { port: 5173, strictPort: true, watch: { ignored: ["**/src-tauri/**"] } },
});
