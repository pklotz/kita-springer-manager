import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // Don't precache index.html — keep it network-first so backend changes
        // (auth, new endpoints) propagate immediately. Precaching index.html
        // makes the SW serve a stale shell for one+ load after every deploy.
        globPatterns: ['**/*.{js,css,ico,png,svg}'],
        navigateFallback: null,
        skipWaiting: true,
        clientsClaim: true,
        // /api/* and the index navigation must always go to the network so
        // basic-auth challenges and fresh data show up.
        navigateFallbackDenylist: [/^\/api\//],
        runtimeCaching: [
          {
            urlPattern: ({ request }) => request.mode === 'navigate',
            handler: 'NetworkFirst',
            options: {
              cacheName: 'pages',
              networkTimeoutSeconds: 5,
            },
          },
          {
            urlPattern: ({ url }) => url.pathname.startsWith('/api/'),
            handler: 'NetworkOnly',
          },
          {
            urlPattern: /^https:\/\/transport\.opendata\.ch\/.*/i,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'transit-api',
              expiration: { maxEntries: 50, maxAgeSeconds: 3600 },
            },
          },
        ],
      },
      manifest: {
        name: 'Kita Springer Manager',
        short_name: 'KitaSpringer',
        description: 'Einsatzplanung für Kita-Springerinnen in Bern',
        theme_color: '#2563eb',
        background_color: '#ffffff',
        display: 'standalone',
        orientation: 'portrait',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any maskable' },
        ],
      },
    }),
  ],
  server: {
    proxy: {
      '/api': 'http://localhost:9092',
    },
  },
})
