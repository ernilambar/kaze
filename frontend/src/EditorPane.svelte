<script lang="ts">
    import type * as monaco from 'monaco-editor';
    import { onMount } from 'svelte';
    import { createEditor, createModel, WELCOME_KEY, WELCOME_TEXT } from './editor';
    import { loadEditorSettings } from './settings';

    interface Props {
        // modelKey identifies the active buffer: a document path, or
        // WELCOME_KEY for the welcome screen.
        modelKey?: string;
        // content is only read the first time a model is created, so a tab
        // keeps its edits when the user switches away and back.
        content?: string;
        // openPaths is every tab currently open. Models for closed tabs are
        // disposed so reopening a file starts from its content on disk.
        openPaths?: string[];
        onCursorMove?: (line: number, column: number) => void;
    }

    let {
        modelKey = WELCOME_KEY,
        content = WELCOME_TEXT,
        openPaths = [],
        onCursorMove,
    }: Props = $props();

    let container: HTMLElement;
    let editor = $state<monaco.editor.IStandaloneCodeEditor>();

    // One model per document key, plus the view state to restore when the user
    // returns to a tab.
    const models = new Map<string, monaco.editor.ITextModel>();
    const viewStates = new Map<string, monaco.editor.ICodeEditorViewState | null>();
    let activeKey = WELCOME_KEY;

    function modelFor(key: string, value: string): monaco.editor.ITextModel {
        let model = models.get(key);
        if (model === undefined) {
            model = createModel(key, value);
            models.set(key, model);
        }
        return model;
    }

    // showModel swaps the editor to key's buffer, creating it on first use.
    // Each buffer keeps its own undo stack, so switching tabs never rewrites
    // text or loses history.
    function showModel(key: string, value: string): void {
        if (editor === undefined) {
            return;
        }
        if (key === activeKey && editor.getModel() !== null) {
            return;
        }
        if (editor.getModel() !== null) {
            viewStates.set(activeKey, editor.saveViewState());
        }
        editor.setModel(modelFor(key, value));
        const state = viewStates.get(key);
        if (state) {
            editor.restoreViewState(state);
        }
        activeKey = key;

        // setModel does not emit onDidChangeCursorPosition, so tell the status
        // bar where the restored tab's cursor is.
        const position = editor.getPosition();
        if (position !== null) {
            onCursorMove?.(position.lineNumber, position.column);
        }
    }

    // pruneModels disposes buffers for tabs that were closed. The active model
    // is skipped because it is the one the editor is displaying.
    function pruneModels(keys: string[], active: string): void {
        const keep = new Set(keys);
        keep.add(WELCOME_KEY);
        for (const [key, model] of models) {
            if (key !== active && !keep.has(key)) {
                model.dispose();
                models.delete(key);
                viewStates.delete(key);
            }
        }
    }

    $effect(() => {
        showModel(modelKey, content);
        pruneModels(openPaths, modelKey);
    });

    onMount(() => {
        let disposed = false;
        let cursorSubscription: { dispose(): void } | undefined;

        void loadEditorSettings().then((settings) => {
            // The pane can unmount while settings are still loading.
            if (disposed) {
                return;
            }
            editor = createEditor(container, settings, modelFor(modelKey, content));
            activeKey = modelKey;
            cursorSubscription = editor.onDidChangeCursorPosition((event) => {
                onCursorMove?.(event.position.lineNumber, event.position.column);
            });
        });

        return () => {
            disposed = true;
            cursorSubscription?.dispose();
            editor?.dispose();
            for (const model of models.values()) {
                model.dispose();
            }
            models.clear();
            viewStates.clear();
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
