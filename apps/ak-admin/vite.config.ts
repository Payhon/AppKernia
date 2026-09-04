import react from '@vitejs/plugin-react'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import { readFile } from 'node:fs/promises'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig, type Plugin } from 'vite'

const canonicalOpenApiPath = resolve(import.meta.dirname, '../../server/openapi/openapi.yaml')
const apiProxyTarget = process.env['AK_API_PROXY_TARGET'] ?? 'http://127.0.0.1:8080'

function canonicalOpenApiPlugin(): Plugin {
  let command: 'build' | 'serve' = 'serve'
  return {
    name: 'appkernia-canonical-openapi',
    configResolved(config) {
      command = config.command
    },
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        const pathname = new URL(request.url ?? '/', 'http://appkernia.local').pathname
        if (pathname === '/openapi') {
          response.statusCode = 308
          response.setHeader('Location', '/openapi/')
          response.end()
          return
        }
        if (pathname.startsWith('/openapi/')) {
          response.setHeader('Cache-Control', 'no-cache, must-revalidate')
          response.setHeader('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
          response.setHeader('Permissions-Policy', 'camera=(), geolocation=(), microphone=()')
          response.setHeader('Referrer-Policy', 'no-referrer')
          response.setHeader('X-Content-Type-Options', 'nosniff')
          response.setHeader('X-Frame-Options', 'DENY')
        }
        if (pathname === '/openapi/openapi.yaml') {
          if (request.method !== 'GET' && request.method !== 'HEAD') {
            response.statusCode = 405
            response.setHeader('Allow', 'GET, HEAD')
            response.end()
            return
          }
          const source = readFileSync(canonicalOpenApiPath)
          response.statusCode = 200
          response.setHeader('Content-Type', 'application/yaml; charset=utf-8')
          response.end(request.method === 'HEAD' ? undefined : source)
          return
        }
        if (pathname.startsWith('/internal/') && !/^\/internal\/v1\/health\/(live|ready)$/.test(pathname)) {
          response.statusCode = 404
          response.end()
          return
        }
        next()
      })
    },
    async buildStart() {
      if (command !== 'build') return
      this.emitFile({
        fileName: 'openapi/openapi.yaml',
        source: await readFile(canonicalOpenApiPath),
        type: 'asset',
      })
    },
  }
}

export default defineConfig({
  plugins: [canonicalOpenApiPlugin(), tanstackRouter({ target: 'react', autoCodeSplitting: true }), react()],
  server: {
    port: 4173,
    proxy: {
      '/admin-api': apiProxyTarget,
      '/api': apiProxyTarget,
      '^/internal/v1/health/(live|ready)$': apiProxyTarget,
    },
    strictPort: true,
  },
  build: {
    manifest: true,
    rollupOptions: {
      input: {
        admin: resolve(import.meta.dirname, 'index.html'),
        openapi: resolve(import.meta.dirname, 'openapi/index.html'),
      },
    },
    sourcemap: true,
    target: 'es2023',
  },
})
