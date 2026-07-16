import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const BACKEND_TARGET =
  "http://127.0.0.1:8080";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],

  server: {
    // Bisa diakses melalui localhost dan jaringan LAN.
    host: "0.0.0.0",

    port: 5173,

    // Jangan berpindah otomatis ke 5174 atau port lain.
    strictPort: true,

    /**
     * Mengizinkan hostname Quick Tunnel Cloudflare.
     *
     * Contoh:
     * random-name.trycloudflare.com
     */
    allowedHosts: [
      ".trycloudflare.com",
    ],

    /**
     * Semua request yang diawali /api diteruskan
     * oleh Vite ke backend Go.
     *
     * Browser:
     * https://public-url.trycloudflare.com/api/plants
     *
     * Diteruskan menjadi:
     * http://127.0.0.1:8080/api/plants
     */
    proxy: {
      "/api": {
        target: BACKEND_TARGET,

        changeOrigin: true,

        secure: false,
      },
    },
  },

  preview: {
    host: "0.0.0.0",

    port: 4173,

    strictPort: true,
  },
});