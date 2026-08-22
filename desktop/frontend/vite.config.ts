import { defineConfig } from 'vite'

// The Wails v3 runtime is served by the Go asset server at /wails/runtime.js.
// We alias the bare "@wailsio/runtime" specifier used by the generated bindings
// to that URL and mark it external so the import survives bundling as a live
// URL import the webview can resolve at runtime.
export default defineConfig({
  server: {
    port: 34115,
    strictPort: true,
  },
  resolve: {
    alias: {
      '@wailsio/runtime': '/wails/runtime.js',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      external: ['/wails/runtime.js'],
    },
  },
})
