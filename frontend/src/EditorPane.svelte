<script lang="ts">
    import type * as monaco from 'monaco-editor';
    import { onMount } from 'svelte';
    import { createEditor, createModel, WELCOME_KEY, WELCOME_TEXT } from './editor';
    import { loadEditorSettings } from './settings';

    interface Props {
        // modelKey identifies the active buffer: a document tab's id, or
        // WELCOME_KEY for the welcome screen.
        modelKey?: string;
        // content is only read the first time a model is created, so a tab
        // keeps its edits when the user switches away and back.
        content?: string;
        // openIds is every tab currently open. Models for closed tabs are
        // disposed so reopening a file starts from its content on disk.
        openIds?: string[];
        onCursorMove?: (line: number, column: number) => void;
        // onDirtyChange reports when a buffer stops or starts matching the file
        // it came from, so its tab can be marked.
        onDirtyChange?: (id: string, dirty: boolean) => void;
    }

    let {
        modelKey = WELCOME_KEY,
        content = WELCOME_TEXT,
        openIds = [],
        onCursorMove,
        onDirtyChange,
    }: Props = $props();

    let container: HTMLElement;
    let editor = $state<monaco.editor.IStandaloneCodeEditor>();

    // One model per document key, plus the view state to restore when the user
    // returns to a tab.
    const models = new Map<string, monaco.editor.ITextModel>();
    const viewStates = new Map<string, monaco.editor.ICodeEditorViewState | null>();
    // savedTexts holds the text each model had when it was loaded or saved: a
    // buffer is clean exactly while its text still matches.
    const savedTexts = new Map<string, string>();
    let activeKey = WELCOME_KEY;

    // value is a buffer's current text. The caller saves this rather than the
    // tab's loaded content, so an unsaved edit is what reaches the file.
    export function value(key: string): string | undefined {
        return models.get(key)?.getValue();
    }

    // markSaved re-baselines a buffer after its text reached the disk. It
    // reports whether the buffer is now clean: one edited while the write was in
    // flight stays modified, because the file holds the earlier text.
    export function markSaved(key: string, saved: string): boolean {
        const model = models.get(key);
        if (model === undefined) {
            // A buffer that was never displayed has no model to disagree with.
            return true;
        }
        if (model.getValue() !== saved) {
            return false;
        }
        savedTexts.set(key, saved);
        onDirtyChange?.(key, false);
        return true;
    }

    function modelFor(key: string, text: string): monaco.editor.ITextModel {
        let model = models.get(key);
        if (model === undefined) {
            model = createModel(key, text);
            models.set(key, model);
            savedTexts.set(key, text);
        }
        return model;
    }

    // reportDirty compares the buffer with the text that is on disk. The length
    // is checked first: it differs after almost every keystroke, so the full
    // comparison only runs when an edit leaves a buffer the same size as its
    // file, such as undoing back to it.
    function reportDirty(key: string): void {
        const model = models.get(key);
        const saved = savedTexts.get(key);
        if (model === undefined || saved === undefined) {
            return;
        }
        const dirty = model.getValueLength() !== saved.length || model.getValue() !== saved;
        onDirtyChange?.(key, dirty);
    }

    // showModel swaps the editor to key's buffer, creating it on first use.
    // Each buffer keeps its own undo stack, so switching tabs never rewrites
    // text or loses history.
    function showModel(key: string, text: string): void {
        if (editor === undefined) {
            return;
        }
        if (key === activeKey && editor.getModel() !== null) {
            return;
        }
        if (editor.getModel() !== null) {
            viewStates.set(activeKey, editor.saveViewState());
        }
        editor.setModel(modelFor(key, text));
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
                savedTexts.delete(key);
            }
        }
    }

    $effect(() => {
        showModel(modelKey, content);
        pruneModels(openIds, modelKey);
    });

    onMount(() => {
        let disposed = false;
        let cursorSubscription: { dispose(): void } | undefined;
        let contentSubscription: { dispose(): void } | undefined;

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
            contentSubscription = editor.onDidChangeModelContent(() => reportDirty(activeKey));
        });

        return () => {
            disposed = true;
            cursorSubscription?.dispose();
            contentSubscription?.dispose();
            editor?.dispose();
            for (const model of models.values()) {
                model.dispose();
            }
            models.clear();
            viewStates.clear();
            savedTexts.clear();
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
