import { defineConfig } from "vite";
import dotenv from "dotenv";
import react from "@vitejs/plugin-react";

dotenv.config();
const REMOTE_BACKEND = process.env.B4_BACKEND_URL || "http://192.168.1.1:7000";

console.log("Using backend:", REMOTE_BACKEND);
export default defineConfig({
  plugins: [react()],
  build: { outDir: "dist", emptyOutDir: true },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: REMOTE_BACKEND,
        changeOrigin: true,
        ws: true,
        secure: false,
      },
    },
  },
});
