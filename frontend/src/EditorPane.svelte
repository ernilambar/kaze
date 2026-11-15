<script lang="ts">
    import { onMount } from 'svelte';
    import { createEditor, WELCOME_TEXT } from './editor';
    import { loadEditorSettings } from './settings';

    interface Props {
        content?: string | undefined;
        onCursorMove?: (line: number, column: number) => void;
    }

    let { content, onCursorMove }: Props = $props();

    let container: HTMLElement;
    let editor = $state<ReturnType<typeof createEditor>>();

    onMount(() => {
        let disposed = false;
        let cursorSubscription: { dispose(): void } | undefined;

        void loadEditorSettings().then((settings) => {
            // The pane can unmount while settings are still loading.
            if (disposed) {
                return;
            }
            editor = createEditor(container, settings, content ?? WELCOME_TEXT);
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

    // Opening a file replaces the buffer. The editor is kept, so its font
    // settings and layout survive; only the content and view state reset.
    $effect(() => {
        const next = content;
        if (next === undefined || editor === undefined) {
            return;
        }
        if (editor.getValue() === next) {
            return;
        }
        editor.setValue(next);
        editor.setScrollTop(0);
        editor.setPosition({ lineNumber: 1, column: 1 });
    });
</script>

<main class="editor" bind:this={container}></main>

<style>
    .editor {
        flex: 1 1 auto;
        min-height: 0;
    }
</style>
