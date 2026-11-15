import { svelte } from '@sveltejs/vite-plugin-svelte';
import wails from '@wailsio/runtime/plugins/vite';
import { defineConfig } from 'vite';

export default defineConfig({
    plugins: [svelte(), wails('./bindings')],
    server: {
        host: '127.0.0.1',
        port: Number(process.env.WAILS_VITE_PORT) || 9245,
        strictPort: true,
    },
    build: {
        target: 'esnext',
        // Monaco's editor core is intentionally bundled in the app chunk, so it
        // is large by design; the warning would fire on every build otherwise.
        chunkSizeWarningLimit: 4500,
    },
    // Monaco ships its own language workers; pre-bundling breaks their
    // relative worker URLs in dev.
    optimizeDeps: {
        exclude: ['monaco-editor'],
    },
});
