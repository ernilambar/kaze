import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

export default defineConfig({
    // Svelte components are not directly under test yet, but importing one from
    // a test would need the plugin to compile it.
    plugins: [svelte()],
    test: {
        environment: 'happy-dom',
        include: ['src/**/*.test.ts'],
    },
});
