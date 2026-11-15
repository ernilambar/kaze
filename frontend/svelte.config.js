import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
    preprocess: vitePreprocess(),
    compilerOptions: {
        // Runes are the Svelte 5 default; pinning it keeps legacy syntax from
        // creeping in silently.
        runes: true,
    },
};
