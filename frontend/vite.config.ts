import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
    plugins: [svelte()],
    server: {
        port: 5173,
        strictPort: true,
    },
    build: {
        target: 'esnext',
    },
    // Monaco ships its own language workers; pre-bundling breaks their
    // relative worker URLs in dev.
    optimizeDeps: {
        exclude: ['monaco-editor'],
    },
});
