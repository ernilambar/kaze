<script lang="ts">
    import { onMount } from 'svelte';
    import { createEditor } from './editor';
    import { loadEditorSettings } from './settings';

    interface Props {
        onCursorMove?: (line: number, column: number) => void;
    }

    let { onCursorMove }: Props = $props();

    let container: HTMLElement;

    onMount(() => {
        let disposed = false;
        let editor: ReturnType<typeof createEditor> | undefined;
        let cursorSubscription: { dispose(): void } | undefined;

        void loadEditorSettings().then((settings) => {
            // The pane can unmount while settings are still loading.
            if (disposed) {
                return;
            }
            editor = createEditor(container, settings);
            cursorSubscription = editor.onDidChangeCursorPosition((event) => {
                onCursorMove?.(event.position.lineNumber, event.position.column);
            });
        });

        return () => {
            disposed = true;
            cursorSubscription?.dispose();
            editor?.dispose();
        };
    });
</script>

<main class="editor" bind:this={container}></main>

<style>
    .editor {
        flex: 1 1 auto;
        min-height: 0;
    }
</style>
