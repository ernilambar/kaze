<script lang="ts">
    import * as monaco from 'monaco-editor';
    import { onMount } from 'svelte';
    import { transformCase, type CaseTransform } from './case';
    import { diffLineKinds, changedLines, type DiffLineKind } from './diff';
    import { DIFF_PREFIX, normalizeEol, type DocumentEOL } from './document';
    import {
        createEditor,
        createModel,
        EDITOR_VERTICAL_PADDING,
        editorOptions,
        WELCOME_KEY,
    } from './editor';
    import {
        SELECTION_COMMANDS,
        type SelectionCommand,
        type ZoomCommand,
    } from './editorcommands';
    import type { EditorSettings } from './settings';
    import {
        appendFinalNewline,
        transformLines as transformLineText,
        trimTrailingWhitespace as trimLineEnd,
        type LineTransform,
    } from './text';
    import type { CopyPathKind } from './workspacecommands';

    // FILE_MENU_GROUP is the trailing section of the editor's right-click menu.
    // Monaco sorts groups lexically with 'navigation' pinned first, so a group
    // that starts with 9_ and sorts after 9_cutcopypaste follows every built-in
    // item: the file commands come last, as they do in other editors.
    const FILE_MENU_GROUP = '9_kaze';

    interface Props {
        // modelKey identifies the active buffer: a document tab's id, or
        // WELCOME_KEY for the idle screen.
        modelKey?: string;
        // content is only read the first time a model is created, so a tab
        // keeps its edits when the user switches away and back.
        content?: string;
        // eol is the active document's line ending. A model adopts it when it
        // is created, so its text already matches what a save writes.
        eol?: string;
        // savedContent is what the buffer last held on disk. It differs from
        // content for a tab moved from another window with unsaved edits, so
        // the moved buffer keeps its dirty state and its undo cannot look saved.
        savedContent?: string;
        // openIds is every tab currently open. Models for closed tabs are
        // disposed so reopening a file starts from its content on disk.
        openIds?: string[];
        // settings is the editor configuration from settings.json. Changing it
        // re-applies the options to the live editor and its buffers.
        settings: EditorSettings;
        // readOnly is set for diff tabs, whose text is a generated snapshot.
        readOnly?: boolean;
        onCursorMove?: (line: number, column: number) => void;
        // onSelectionChange reports how many characters are selected. The
        // status bar appends it to the cursor position.
        onSelectionChange?: (length: number) => void;
        // onRevealFile asks the app to show the active file in the platform's
        // file manager; it is wired to the editor's context menu.
        onRevealFile?: () => void;
        // onCopyPath asks the app to copy the active file's path to the
        // clipboard. It is wired to the editor's context menu as well.
        onCopyPath?: (kind: CopyPathKind) => void;
        // onDirtyChange reports when a buffer stops or starts matching the file
        // it came from, so its tab can be marked.
        onDirtyChange?: (id: string, dirty: boolean) => void;
        // onIndentationChange reports what the active buffer indents with, so
        // the status bar can print it. It fires when a buffer is attached and
        // whenever its model options change: content detection, the settings,
        // and the indentation picker all arrive as model options.
        onIndentationChange?: (insertSpaces: boolean, tabSize: number, indentSize: number) => void;
    }

    let {
        modelKey = WELCOME_KEY,
        content = '',
        savedContent = '',
        eol = 'lf',
        openIds = [],
        settings,
        readOnly = false,
        onCursorMove,
        onSelectionChange,
        onRevealFile,
        onCopyPath,
        onDirtyChange,
        onIndentationChange,
    }: Props = $props();

    // Zoom moves the font size in session steps around the configured size;
    // Actual Size returns to it. The offset is never written to settings.json.
    const MIN_FONT_SIZE = 6;
    const MAX_FONT_SIZE = 72;

    let container: HTMLElement;
    let editor = $state<monaco.editor.IStandaloneCodeEditor>();
    let zoomOffset = $state(0);

    // One model per document key, plus the view state to restore when the user
    // returns to a tab.
    const models = new Map<string, monaco.editor.ITextModel>();
    const viewStates = new Map<string, monaco.editor.ICodeEditorViewState | null>();
    // savedTexts holds the text each model had when it was loaded or saved: a
    // buffer is clean exactly while its text still matches.
    const savedTexts = new Map<string, string>();
    // baselines holds each buffer's committed text. A key is present only when
    // the file belongs to a repository, so files outside git show no gutter
    // marks instead of looking entirely new.
    const baselines = new Map<string, string>();
    const decorations = new Map<string, string[]>();
    const decorationTimers = new Map<string, ReturnType<typeof setTimeout>>();
    // A model's options change events are per model, so each buffer keeps its
    // own listener; only the active one reaches the status bar.
    const optionSubscriptions = new Map<string, monaco.IDisposable>();
    let activeKey = WELCOME_KEY;

    // setBaseline starts comparing a buffer with the text HEAD has for it. It
    // is called when a file is opened and after Save As gives a buffer a path.
    export function setBaseline(key: string, text: string): void {
        baselines.set(key, text);
        updateDecorations(key);
    }

    // clearBaseline removes the gutter marks for a buffer outside a repository.
    export function clearBaseline(key: string): void {
        baselines.delete(key);
        updateDecorations(key);
    }

    function clearDecorations(key: string): void {
        const model = models.get(key);
        const current = decorations.get(key);
        if (model === undefined || current === undefined || current.length === 0) {
            return;
        }
        decorations.set(key, model.deltaDecorations(current, []));
    }

    // cssColor reads the active theme's git colors. Monaco decorations need a
    // literal color, so the CSS variable is resolved when the bar is drawn.
    function cssColor(name: string, fallback: string): string {
        const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
        return value === '' ? fallback : value;
    }

    // DIFF_LINE_CLASS names the background class for each kind of diff line.
    const DIFF_LINE_CLASS: Record<DiffLineKind, string> = {
        added: 'diff-line-added',
        deleted: 'diff-line-deleted',
        header: 'diff-line-header',
    };

    // updateDiffDecorations tints the added, deleted and hunk-header lines of a
    // diff tab. The text is a read-only snapshot, so it only changes on reload.
    function updateDiffDecorations(key: string, model: monaco.editor.ITextModel): void {
        const next = diffLineKinds(model.getValue()).map<monaco.editor.IModelDeltaDecoration>(
            (entry) => ({
                range: {
                    startLineNumber: entry.line,
                    startColumn: 1,
                    endLineNumber: entry.line,
                    endColumn: 1,
                },
                options: { isWholeLine: true, className: DIFF_LINE_CLASS[entry.kind] },
            }),
        );
        decorations.set(key, model.deltaDecorations(decorations.get(key) ?? [], next));
    }

    // updateDecorations recomputes the changed-line bars for one model. The
    // model owns its decorations, so switching tabs keeps every buffer's marks
    // and only the visible one is painted.
    function updateDecorations(key: string): void {
        const model = models.get(key);
        if (model === undefined) {
            return;
        }
        if (key.startsWith(DIFF_PREFIX)) {
            updateDiffDecorations(key, model);
            return;
        }
        const baseline = baselines.get(key);
        if (baseline === undefined || key === WELCOME_KEY) {
            clearDecorations(key);
            return;
        }

        const changed = changedLines(baseline, model.getValue());
        const addedColor = cssColor('--git-added', '#2ea043');
        const modifiedColor = cssColor('--git-modified', '#1f6feb');
        const next = changed.map<monaco.editor.IModelDeltaDecoration>((entry) => ({
            range: {
                startLineNumber: entry.line,
                startColumn: 1,
                endLineNumber: entry.line,
                endColumn: 1,
            },
            options: {
                isWholeLine: true,
                linesDecorationsClassName: `git-gutter-${entry.kind}`,
                overviewRuler: {
                    color: entry.kind === 'added' ? addedColor : modifiedColor,
                    position: monaco.editor.OverviewRulerLane.Left,
                },
            },
        }));
        decorations.set(key, model.deltaDecorations(decorations.get(key) ?? [], next));
    }

    // scheduleDecorations coalesces keystrokes into one diff. Typing a word is
    // several changes, and only the line numbers after the last one matter.
    function scheduleDecorations(key: string): void {
        const pending = decorationTimers.get(key);
        if (pending !== undefined) {
            clearTimeout(pending);
        }
        decorationTimers.set(
            key,
            setTimeout(() => {
                decorationTimers.delete(key);
                updateDecorations(key);
            }, 120),
        );
    }

    // value is a buffer's current text. The caller saves this rather than the
    // tab's loaded content, so an unsaved edit is what reaches the file.
    export function value(key: string): string | undefined {
        return models.get(key)?.getValue();
    }

    // replaceInBuffer applies a full-text replacement to one open buffer as a
    // single undoable operation. The tab stays dirty: Replace All never writes
    // to disk itself, so the user keeps the edits and can undo them in Monaco.
    // Returns the new text, or undefined when the buffer is not open.
    export function replaceInBuffer(key: string, text: string): string | undefined {
        const model = models.get(key);
        if (model === undefined) {
            return undefined;
        }
        const fullRange = model.getFullModelRange();
        model.pushStackElement();
        model.pushEditOperations(
            [],
            [{ range: fullRange, text, forceMoveMarkers: true }],
            () => null,
        );
        model.pushStackElement();
        scheduleDecorations(key);
        // The editor's change subscription only sees the active model, so a
        // buffer edited while a different tab is showing reports its own dirty
        // state here.
        reportDirty(key);
        // The model's value is the source of truth: it normalizes EOL, so use
        // what it holds rather than the text we handed it.
        return model.getValue();
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
        // The file holds saved now, whatever the buffer did meanwhile, so it is
        // the baseline: undoing back to the older text must read as modified.
        savedTexts.set(key, saved);
        if (model.getValue() !== saved) {
            return false;
        }
        onDirtyChange?.(key, false);
        return true;
    }

    // reload replaces a buffer's text with what is now on disk and makes it the
    // clean baseline. The caller only does this for a tab with no unsaved
    // edits, so the external text cannot overwrite the user's work. The file's
    // line ending may have changed too, so the model follows it.
    export function reload(key: string, text: string, eol: string): void {
        const model = models.get(key);
        if (model === undefined) {
            return;
        }
        const crlf = eol === 'crlf';
        if ((model.getEOL() === '\r\n') !== crlf) {
            model.setEOL(
                crlf
                    ? monaco.editor.EndOfLineSequence.CRLF
                    : monaco.editor.EndOfLineSequence.LF,
            );
        }
        if (model.getValue() !== text) {
            model.setValue(text);
        }
        // The model normalizes mixed endings, so its own value is the baseline
        // the next keystroke is compared with.
        savedTexts.set(key, model.getValue());
        scheduleDecorations(key);
        onDirtyChange?.(key, false);
    }

    // revealLine puts the cursor on a line and column and scrolls it into view.
    // It is only meaningful for the buffer the editor is currently showing, so
    // a search result calls it after its tab has been activated.
    export function revealLine(key: string, line: number, column: number): void {
        const current = editor;
        if (current === undefined || activeKey !== key) {
            return;
        }
        const position = { lineNumber: line, column };
        current.setPosition(position);
        current.setSelection(new monaco.Range(line, column, line, column));
        current.revealLineInCenter(line, monaco.editor.ScrollType.Smooth);
        current.focus();
    }

    // setLanguage re-highlights a buffer after Save As gave it a real path or
    // Change Language Mode picked an override. The caller resolves the id; this
    // only touches the model.
    export function setLanguage(key: string, languageId: string): void {
        const model = models.get(key);
        if (model !== undefined) {
            monaco.editor.setModelLanguage(model, languageId);
        }
    }

    // setEol switches a buffer's line endings in place. Monaco records it as an
    // edit, so the buffer becomes dirty and Save writes the new sequence.
    export function setEol(key: string, eol: DocumentEOL): void {
        const model = models.get(key);
        if (model === undefined) {
            return;
        }
        const crlf = eol === 'crlf';
        if ((model.getEOL() === '\r\n') === crlf) {
            return;
        }
        model.setEOL(
            crlf ? monaco.editor.EndOfLineSequence.CRLF : monaco.editor.EndOfLineSequence.LF,
        );
    }

    // transformSelection converts the selection, or the whole buffer when
    // nothing is selected. It is one undo step and leaves the cursor in place.
    export function transformSelection(transform: CaseTransform): void {
        const current = editor;
        const model = current?.getModel() ?? null;
        if (current === undefined || model === null) {
            return;
        }
        const selection = current.getSelection();
        if (selection === null) {
            return;
        }
        const range = selection.isEmpty() ? model.getFullModelRange() : selection;
        const text = model.getValueInRange(range);
        const converted = transformCase(text, transform);
        if (converted === text) {
            return;
        }
        current.pushUndoStop();
        current.executeEdits('transform-case', [
            { range, text: converted, forceMoveMarkers: true },
        ]);
        current.pushUndoStop();
        current.focus();
    }

    // trimmedEdits lists the deletions that remove trailing spaces and tabs
    // from each affected line. Editing only the line ends leaves cursors and
    // the rest of the buffer untouched.
    function trimmedEdits(
        model: monaco.editor.ITextModel,
    ): monaco.editor.IIdentifiedSingleEditOperation[] {
        const edits: monaco.editor.IIdentifiedSingleEditOperation[] = [];
        for (let line = 1; line <= model.getLineCount(); line++) {
            const lineContent = model.getLineContent(line);
            const trimmed = trimLineEnd(lineContent);
            if (trimmed !== lineContent) {
                edits.push({
                    range: new monaco.Range(line, trimmed.length + 1, line, lineContent.length + 1),
                    text: '',
                });
            }
        }
        return edits;
    }

    // trimModel applies one undoable trim to a buffer, whether or not the
    // editor is showing it.
    function trimModel(model: monaco.editor.ITextModel): boolean {
        const edits = trimmedEdits(model);
        if (edits.length === 0) {
            return false;
        }
        model.pushStackElement();
        model.pushEditOperations([], edits, () => null);
        model.pushStackElement();
        return true;
    }

    // trimTrailingWhitespace edits the active buffer's line ends.
    export function trimTrailingWhitespace(): void {
        const current = editor;
        const model = current?.getModel() ?? null;
        if (current === undefined || model === null) {
            return;
        }
        if (!trimModel(model)) {
            return;
        }
        current.focus();
    }

    // trimTrailingWhitespaceInBuffer trims a buffer before its text is saved,
    // including one the editor is not currently showing.
    export function trimTrailingWhitespaceInBuffer(key: string): void {
        const model = models.get(key);
        if (model !== undefined) {
            trimModel(model);
        }
    }

    // transformLines applies a whole-line command to the selection, or to the
    // entire buffer when nothing is selected. It is one undo step and leaves
    // the cursor in place.
    export function transformLines(transform: LineTransform): void {
        const current = editor;
        const model = current?.getModel() ?? null;
        if (current === undefined || model === null) {
            return;
        }
        const selection = current.getSelection();
        if (selection === null) {
            return;
        }
        const range = selection.isEmpty() ? model.getFullModelRange() : selection;
        const text = model.getValueInRange(range);
        const converted = transformLineText(text, transform);
        if (converted === text) {
            return;
        }
        current.pushUndoStop();
        current.executeEdits('transform-lines', [
            { range, text: converted, forceMoveMarkers: true },
        ]);
        current.pushUndoStop();
        current.focus();
    }

    // ensureFinalNewline appends the newline the save setting promises, editing
    // the buffer itself so the saved text and the model still agree afterwards.
    export function ensureFinalNewline(key: string): void {
        const model = models.get(key);
        if (model === undefined) {
            return;
        }
        const text = model.getValue();
        if (appendFinalNewline(text) === text) {
            return;
        }
        const line = model.getLineCount();
        const column = model.getLineMaxColumn(line);
        model.pushEditOperations(
            [],
            [{ range: new monaco.Range(line, column, line, column), text: '\n' }],
            () => null,
        );
    }

    // runSelectionCommand runs Monaco's multi-cursor and line action for one
    // Selection menu command. Column selection is an editor option rather than
    // an action, so it toggles that option itself.
    export function runSelectionCommand(command: SelectionCommand): void {
        const current = editor;
        if (current === undefined) {
            return;
        }
        const spec = SELECTION_COMMANDS.find((entry) => entry.id === command);
        if (spec === undefined) {
            return;
        }
        current.focus();
        if (spec.action !== undefined) {
            void current.getAction(spec.action)?.run();
            return;
        }
        current.updateOptions({
            columnSelection: !current.getOption(monaco.editor.EditorOption.columnSelection),
        });
    }

    // runEditorAction runs a Monaco action by id, focusing the editor first so
    // Find, Replace and Go to Line act on this buffer rather than on whatever
    // had focus.
    export function runEditorAction(action: string): void {
        const current = editor;
        if (current === undefined) {
            return;
        }
        current.focus();
        void current.getAction(action)?.run();
    }

    // zoom changes the effective font size around the configured one.
    export function zoom(command: ZoomCommand): void {
        if (editor === undefined) {
            return;
        }
        if (command === 'reset') {
            zoomOffset = 0;
            return;
        }
        const size = settings.fontSize + zoomOffset;
        const next = Math.min(
            Math.max(command === 'in' ? size + 1 : size - 1, MIN_FONT_SIZE),
            MAX_FONT_SIZE,
        );
        zoomOffset = next - settings.fontSize;
    }

    // applyScrollBeyond places the empty space below the last line. Monaco only
    // knows "one page" or none, so half_page pays for its half viewport with
    // bottom padding, which counts toward the scroll height. The guard stops a
    // layout change caused by the padding itself from looping.
    let appliedScrollBeyond: EditorSettings['scrollBeyondLastLine'] | undefined;
    let appliedPaddingBottom = 0;

    function applyScrollBeyond(
        current: monaco.editor.IStandaloneCodeEditor,
        force = false,
    ): void {
        const mode = settings.scrollBeyondLastLine;
        const height = current.getLayoutInfo().height;
        const bottom =
            mode === 'half_page'
                ? Math.max(EDITOR_VERTICAL_PADDING, Math.round(height / 2))
                : EDITOR_VERTICAL_PADDING;
        if (!force && mode === appliedScrollBeyond && bottom === appliedPaddingBottom) {
            return;
        }
        appliedScrollBeyond = mode;
        appliedPaddingBottom = bottom;
        current.updateOptions({
            scrollBeyondLastLine: mode === 'one_page',
            padding: { top: EDITOR_VERTICAL_PADDING, bottom },
        });
    }

    function modelFor(key: string, text: string, documentEol: string): monaco.editor.ITextModel {
        let model = models.get(key);
        if (model === undefined) {
            // createModel runs Monaco's indentation detection with the settings
            // as the fallback, so nothing is forced on the model here.
            model = createModel(key, text, documentEol);
            models.set(key, model);
            optionSubscriptions.set(
                key,
                model.onDidChangeOptions(() => {
                    if (key === activeKey) {
                        reportIndentation(key);
                    }
                }),
            );
            savedTexts.set(key, normalizeEol(savedContent, documentEol));
            // A baseline may have arrived before the tab was first shown.
            updateDecorations(key);
        }
        return model;
    }

    // reportIndentation tells the status bar what one buffer indents with.
    // Monaco keeps tab size and space insertion on the model, so the resolved
    // options are read rather than tracked: an indentSize of `tabSize` is
    // already resolved to a number.
    function reportIndentation(key: string): void {
        const options = models.get(key)?.getOptions();
        if (options === undefined) {
            return;
        }
        onIndentationChange?.(options.insertSpaces, options.tabSize, options.indentSize);
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

    // reportSelection counts the characters selected across every cursor. The
    // status bar shows it beside the cursor position; an empty selection
    // reports zero and the bar drops it.
    function reportSelection(): void {
        const current = editor;
        if (current === undefined) {
            onSelectionChange?.(0);
            return;
        }
        const model = current.getModel();
        if (model === null) {
            onSelectionChange?.(0);
            return;
        }
        let length = 0;
        for (const selection of current.getSelections() ?? []) {
            if (!selection.isEmpty()) {
                length += model.getValueLengthInRange(selection);
            }
        }
        onSelectionChange?.(length);
    }

    // showModel swaps the editor to key's buffer, creating it on first use.
    // Each buffer keeps its own undo stack, so switching tabs never rewrites
    // text or loses history.
    function showModel(key: string, text: string, documentEol: string): void {
        if (editor === undefined) {
            return;
        }
        if (key === activeKey && editor.getModel() !== null) {
            // The visible buffer did not change, but this is also the first
            // pass after the editor is created, and the status bar starts
            // empty-handed.
            reportIndentation(key);
            return;
        }
        if (editor.getModel() !== null) {
            viewStates.set(activeKey, editor.saveViewState());
        }
        editor.setModel(modelFor(key, text, documentEol));
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
        reportIndentation(key);
        reportSelection();
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
                baselines.delete(key);
                decorations.delete(key);
                optionSubscriptions.get(key)?.dispose();
                optionSubscriptions.delete(key);
                const pending = decorationTimers.get(key);
                if (pending !== undefined) {
                    clearTimeout(pending);
                    decorationTimers.delete(key);
                }
            }
        }
    }

    $effect(() => {
        showModel(modelKey, content, eol);
        pruneModels(openIds, modelKey);
    });

    // A settings change re-applies the options to the live editor. Indentation
    // needs no pass over the buffers: the tab size and space insertion in the
    // options reach the models through Monaco's configuration, which keeps
    // each buffer's detected indentation and re-detects only when the settings
    // values themselves change.
    $effect(() => {
        const current = editor;
        if (current === undefined) {
            return;
        }
        current.updateOptions(editorOptions(settings, settings.fontSize + zoomOffset));
        applyScrollBeyond(current, true);
    });

    // Diff tabs are snapshots: Monaco refuses edits, so the model cannot drift
    // from what git printed.
    $effect(() => {
        editor?.updateOptions({ readOnly });
    });

    onMount(() => {
        const created = createEditor(container, settings, modelFor(modelKey, content, eol));
        editor = created;
        applyScrollBeyond(created);
        activeKey = modelKey;
        const cursorSubscription = created.onDidChangeCursorPosition((event) => {
            onCursorMove?.(event.position.lineNumber, event.position.column);
        });
        const selectionSubscription = created.onDidChangeCursorSelection(() => {
            reportSelection();
        });
        const contentSubscription = created.onDidChangeModelContent(() => {
            reportDirty(activeKey);
            scheduleDecorations(activeKey);
        });
        const layoutSubscription = created.onDidLayoutChange(() => applyScrollBeyond(created));
        const revealAction = created.addAction({
            id: 'reveal-in-finder',
            label: 'Reveal in Finder',
            contextMenuGroupId: FILE_MENU_GROUP,
            contextMenuOrder: 1,
            run: () => onRevealFile?.(),
        });
        const copyPathAction = created.addAction({
            id: 'copy-path',
            label: 'Copy Path',
            contextMenuGroupId: FILE_MENU_GROUP,
            contextMenuOrder: 2,
            run: () => onCopyPath?.('file'),
        });
        const copyRelativePathAction = created.addAction({
            id: 'copy-relative-path',
            label: 'Copy Relative Path',
            contextMenuGroupId: FILE_MENU_GROUP,
            contextMenuOrder: 3,
            run: () => onCopyPath?.('relative'),
        });

        return () => {
            cursorSubscription.dispose();
            selectionSubscription.dispose();
            contentSubscription.dispose();
            layoutSubscription.dispose();
            revealAction.dispose();
            copyPathAction.dispose();
            copyRelativePathAction.dispose();
            created.dispose();
            for (const model of models.values()) {
                model.dispose();
            }
            for (const pending of decorationTimers.values()) {
                clearTimeout(pending);
            }
            for (const subscription of optionSubscriptions.values()) {
                subscription.dispose();
            }
            models.clear();
            viewStates.clear();
            savedTexts.clear();
            baselines.clear();
            decorations.clear();
            decorationTimers.clear();
            optionSubscriptions.clear();
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
