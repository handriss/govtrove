import { defineConfig } from 'vite'
import type { Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

function asyncCss(): Plugin {
  return {
    name: 'async-css',
    enforce: 'post',
    transformIndexHtml(html) {
      return html.replace(
        /<link rel="stylesheet" crossorigin href="(\/assets\/[^"]+\.css)">/g,
        (_match, href) =>
          `<link rel="preload" href="${href}" as="style" onload="this.onload=null;this.rel='stylesheet'" crossorigin>\n    <noscript><link rel="stylesheet" href="${href}" crossorigin></noscript>`,
      )
    },
  }
}

export default defineConfig({
  plugins: [react(), tailwindcss(), asyncCss()],
  envDir: '..',
  server: {
    proxy: {
      '/api': 'http://localhost:3000',
      '/health': 'http://localhost:3000',
    },
  },
  build: {
    target: 'es2022',
    rollupOptions: {
      output: {
        manualChunks: {
          recharts: ['recharts'],
          vendor: ['react', 'react-dom', 'react-router-dom', '@workos-inc/authkit-react'],
        },
      },
    },
  },
})
