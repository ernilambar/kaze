<script lang="ts">
    import { onMount } from 'svelte';
    import {
        InstallCommandLineTool,
        OpenFileDialog,
        OpenSettings,
        PrintDocument,
        ResolveClose,
        SaveDocument,
        SaveFileDialog,
        SetUnsavedChanges,
    } from '../bindings/kova/app';
    import type { SaveResult } from '../bindings/kova/models';
    import EditorPane from './EditorPane.svelte';
    import ImageViewer from './ImageViewer.svelte';
    import Palette from './Palette.svelte';
    import Prompt from './Prompt.svelte';
    import Sidebar from './Sidebar.svelte';
    import StatusBar from './StatusBar.svelte';
    import TabBar from './TabBar.svelte';
    import TitleBar from './TitleBar.svelte';
    import { listenForCommandPalette, rankCommands } from './commands';
    import {
        createUntitledDocument,
        listenForLaunch,
        readDocument,
        shownName,
        type LaunchRequest,
        type OpenDocument,
    } from './document';
    import { WELCOME_KEY, WELCOME_TEXT } from './editor';
    import { listenForCloseRequest, listenForFileCommands } from './filecommands';
    import type { PaletteItem } from './palette';
    import {
        listFiles,
        listenForQuickOpen,
        rankQuickOpen,
        toQuickOpenFiles,
        type QuickOpenFile,
    } from './quickOpen';
    import {
        addSidebarFolder,
        errorMessage,
        listenForAddFolder,
        loadSidebarFolders,
        openFolderDialog,
        removeSidebarFolder,
        sidebarLabel,
    } from './sidebar';
    import { noticeFor } from './status';
    import {
        closeDocumentTab,
        listenForCloseTab,
        openDocumentTab,
        setDocumentDirty,
        updateDocument,
    } from './tabs';

    // UnsavedPrompt is the dialog asking what to do with modified buffers
    // before they are closed. answer resolves the promise the caller is
    // awaiting, so a save or close sequence reads top to bottom.
    interface UnsavedPrompt {
        title: string;
        message: string;
        details: string[];
        confirmLabel: string;
        answer: (answer: 'save' | 'discard' | 'cancel') => void;
    }

    // ConflictPrompt asks whether a buffer may overwrite a file another program
    // changed while it was open.
    interface ConflictPrompt {
        message: string;
        answer: (overwrite: boolean) => void;
    }

    const APP_NAME = 'Kova';
    const LANGUAGE = 'Plain Text';

    let tabs = $state<OpenDocument[]>([]);
    let activeId = $state('');
    let folders = $state<string[]>([]);
    let notice = $state('');
    let busy = $state(false);

    let quickOpen = $state(false);
    let quickOpenFiles = $state<QuickOpenFile[]>([]);
    let quickOpenLoading = $state(false);
    let commandPalette = $state(false);

    let unsavedPrompt = $state<UnsavedPrompt | null>(null);
    let conflictPrompt = $state<ConflictPrompt | null>(null);

    let line = $state(1);
    let column = $state(1);

    let editorPane = $state<EditorPane>();

    let active = $derived(tabs.find((tab) => tab.id === activeId));
    let imageActive = $derived(active?.kind === 'image');
    // Monaco keeps its buffers while an image tab is showing, so switching back
    // to a text tab preserves edits. The welcome buffer keeps the hidden editor
    // from creating a model for an image path.
    let editorKey = $derived(imageActive ? WELCOME_KEY : (active?.id ?? WELCOME_KEY));
    let editorContent = $derived(imageActive ? WELCOME_TEXT : (active?.content ?? WELCOME_TEXT));
    let openIds = $derived(tabs.map((tab) => tab.id));
    let unsaved = $derived(tabs.some((tab) => tab.dirty));
    let folderLabel = $derived(sidebarLabel(folders));
    let quickOpenEmpty = $derived(
        quickOpenLoading
            ? 'Loading files…'
            : quickOpenFiles.length === 0
              ? 'Add a folder to the sidebar to search its files.'
              : 'No matching files.',
    );

    function findTab(id: string): OpenDocument | undefined {
        return tabs.find((tab) => tab.id === id);
    }

    // showDocument adds a tab for the document unless its file is already open,
    // then makes it active.
    function showDocument(document: OpenDocument): void {
        const opened = openDocumentTab(tabs, document);
        tabs = opened.tabs;
        activeId = opened.activeId;
    }

    // addFolders saves each root from a launch request and adopts the result.
    // The backend deduplicates, so relaunching with the same folder is a no-op.
    // The first failure is returned so the caller can show it in the status bar.
    async function addFolders(paths: string[]): Promise<string> {
        let failure = '';
        let latest = folders;
        for (const path of paths) {
            try {
                latest = await addSidebarFolder(path);
            } catch (error) {
                if (failure === '') {
                    failure = errorMessage(error);
                }
            }
        }
        folders = latest;
        return failure;
    }

    async function applyLaunch(request: LaunchRequest): Promise<void> {
        let failure = '';
        if (request.folders.length > 0) {
            failure = await addFolders(request.folders);
        }

        for (const path of request.files) {
            const existing = tabs.find((tab) => tab.path !== '' && tab.path === path);
            if (existing !== undefined) {
                activeId = existing.id;
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                if (failure === '') {
                    failure = `Cannot open ${path}`;
                }
                continue;
            }
            showDocument(opened);
        }

        notice = failure !== '' ? failure : noticeFor(request);
    }

    async function openFile(path: string): Promise<void> {
        // Clicking a file that is already open focuses its tab instead of
        // rereading the file, which would discard edits made in the buffer.
        const existing = tabs.find((tab) => tab.path !== '' && tab.path === path);
        if (existing !== undefined) {
            activeId = existing.id;
            notice = '';
            return;
        }
        const opened = await readDocument(path);
        if (opened === undefined) {
            notice = `Cannot open ${path}`;
            return;
        }
        showDocument(opened);
        notice = '';
    }

    // newFile opens an empty buffer with no path yet. Save turns it into a
    // Save As, which is how most editors treat an untitled document.
    function newFile(): void {
        const document = createUntitledDocument(tabs);
        tabs = [...tabs, document];
        activeId = document.id;
        notice = '';
    }

    async function openFromDialog(): Promise<void> {
        if (busy) {
            return;
        }
        busy = true;
        try {
            const path = await OpenFileDialog();
            // Cancelling the dialog is a decision, not a failure.
            if (path === '') {
                return;
            }
            await openFile(path);
        } catch (error) {
            notice = errorMessage(error);
        } finally {
            busy = false;
        }
    }

    function removeTab(id: string): void {
        const result = closeDocumentTab(tabs, id, activeId);
        tabs = result.tabs;
        activeId = result.activeId;
    }

    // bufferText is what the editor holds right now. A tab that has never been
    // displayed has no model yet, so its loaded content is the answer.
    function bufferText(tab: OpenDocument): string {
        return editorPane?.value(tab.id) ?? tab.content;
    }

    // askUnsaved shows the prompt and resolves with the action the user picked.
    function askUnsaved(options: Omit<UnsavedPrompt, 'answer'>): Promise<'save' | 'discard' | 'cancel'> {
        return new Promise((resolve) => {
            unsavedPrompt = { ...options, answer: resolve };
        });
    }

    function answerUnsaved(answer: string): void {
        if (answer !== 'save' && answer !== 'discard' && answer !== 'cancel') {
            return;
        }
        const prompt = unsavedPrompt;
        unsavedPrompt = null;
        prompt?.answer(answer);
    }

    function askConflict(message: string): Promise<boolean> {
        return new Promise((resolve) => {
            conflictPrompt = { message, answer: resolve };
        });
    }

    function answerConflict(overwrite: boolean): void {
        const prompt = conflictPrompt;
        conflictPrompt = null;
        prompt?.answer(overwrite);
    }

    async function chooseSavePath(tab: OpenDocument): Promise<string> {
        try {
            return await SaveFileDialog(tab.path !== '' ? tab.path : tab.name);
        } catch (error) {
            notice = errorMessage(error);
            return '';
        }
    }

    function applySaved(id: string, result: SaveResult, content: string): void {
        // The editor decides whether the buffer is clean: typing during the
        // write leaves newer text in the model than the file now holds.
        const clean = editorPane?.markSaved(id, content) ?? true;
        tabs = updateDocument(tabs, id, {
            path: result.path,
            name: result.name,
            modified: result.modified,
            content,
            dirty: !clean,
        });
        notice = '';
    }

    // saveDocument writes one buffer, asking for a path when it has none. A file
    // another program changed is reported as a conflict first, so its content is
    // never overwritten without the user saying so. False means nothing was
    // written: a cancelled dialog, a conflict the user declined, or an error.
    async function saveDocument(id: string, saveAs = false): Promise<boolean> {
        const tab = findTab(id);
        // Pictures are read-only in Kova, so there is nothing to write back.
        if (tab === undefined || tab.kind === 'image') {
            return true;
        }

        const content = bufferText(tab);
        let path = tab.path;
        let expected = tab.modified;
        if (saveAs || path === '') {
            path = await chooseSavePath(tab);
            if (path === '') {
                return false;
            }
            expected = 0;
            // Two tabs writing one file would disagree about its content, so
            // Save As refuses a path that is already open.
            if (tabs.some((other) => other.id !== id && other.path === path)) {
                notice = `${path} is already open.`;
                return false;
            }
        }

        try {
            const result = await SaveDocument({
                path,
                content,
                expected_modified: expected,
                overwrite: false,
            });
            if (result.conflict) {
                const overwrite = await askConflict(
                    `${result.name} changed on disk since it was opened. Overwrite it with your version?`,
                );
                if (!overwrite) {
                    return false;
                }
                const forced = await SaveDocument({
                    path: result.path,
                    content,
                    expected_modified: expected,
                    overwrite: true,
                });
                applySaved(id, forced, content);
                return true;
            }
            applySaved(id, result, content);
            return true;
        } catch (error) {
            notice = errorMessage(error);
            return false;
        }
    }

    // saveAll writes every modified buffer in tab order. A cancelled Save As or
    // a failed write stops the run, so the rest are left exactly as they were.
    async function saveAll(): Promise<boolean> {
        for (const tab of tabs.filter((entry) => entry.dirty)) {
            if (!(await saveDocument(tab.id))) {
                return false;
            }
        }
        return true;
    }

    // closeTab drops a buffer. A modified buffer asks first, because closing it
    // is also what discards its undo stack.
    async function closeTab(id: string): Promise<void> {
        const tab = findTab(id);
        if (tab === undefined) {
            return;
        }
        if (tab.dirty) {
            const answer = await askUnsaved({
                title: 'Close Tab',
                message: `Save changes to ${tab.name} before closing?`,
                details: [],
                confirmLabel: 'Save',
            });
            if (answer === 'cancel') {
                return;
            }
            if (answer === 'save' && !(await saveDocument(id))) {
                return;
            }
        }
        removeTab(id);
    }

    async function resolveClose(quit: boolean): Promise<void> {
        try {
            await ResolveClose(quit);
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // confirmClose answers Go's close request. It is the one place that decides
    // whether the window may go away, so the close button and Cmd+Q behave the
    // same and share one prompt.
    async function confirmClose(): Promise<void> {
        // Another prompt is already on screen; the user is deciding, so a
        // second Cmd+Q must not stack a dialog on top of it. Answering the
        // request with "cancel" clears Go's pending flag, so the next Cmd+Q
        // asks again instead of being swallowed.
        if (unsavedPrompt !== null || conflictPrompt !== null) {
            await resolveClose(false);
            return;
        }

        const dirty = tabs.filter((tab) => tab.dirty);
        if (dirty.length === 0) {
            await resolveClose(true);
            return;
        }

        const answer = await askUnsaved({
            title: 'Quit Kova',
            message:
                dirty.length === 1
                    ? 'Save changes before quitting?'
                    : `${dirty.length} files have unsaved changes.`,
            details: dirty.map((tab) => tab.name),
            confirmLabel: 'Save All',
        });
        if (answer === 'cancel') {
            await resolveClose(false);
            return;
        }
        if (answer === 'save' && !(await saveAll())) {
            // Saving failed or was cancelled: stay open rather than drop the
            // buffers the user just tried to keep.
            await resolveClose(false);
            return;
        }
        await resolveClose(true);
    }

    async function openSettings(): Promise<void> {
        try {
            await OpenSettings();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function installCommandLineTool(): Promise<void> {
        try {
            await InstallCommandLineTool();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // printActive prints the active buffer: text comes from the editor so an
    // unsaved edit prints as it looks, pictures come from their file.
    async function printActive(): Promise<void> {
        const tab = active;
        if (tab === undefined) {
            notice = 'Open a document to print.';
            return;
        }
        try {
            await PrintDocument({
                path: tab.path,
                content: tab.kind === 'image' ? '' : bufferText(tab),
                image: tab.kind === 'image',
            });
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // showQuickOpen opens the palette first and fills it as soon as Go has
    // walked the sidebar roots, so the input is usable while a large tree is
    // still being read.
    async function showQuickOpen(): Promise<void> {
        commandPalette = false;
        quickOpen = true;
        quickOpenLoading = true;
        try {
            quickOpenFiles = toQuickOpenFiles(await listFiles(), folders);
        } catch (error) {
            quickOpenFiles = [];
            notice = errorMessage(error);
        } finally {
            quickOpenLoading = false;
        }
    }

    function closeQuickOpen(): void {
        quickOpen = false;
    }

    // Picking a file from the palette uses the same path as a sidebar click, so
    // an already-open tab is focused instead of reread.
    async function openFromQuickOpen(path: string): Promise<void> {
        quickOpen = false;
        await openFile(path);
    }

    // The two palettes share Palette.svelte, so Quick Open rows are mapped onto
    // the common row shape and ranked with the same function.
    function rankFiles(query: string): PaletteItem[] {
        return rankQuickOpen(query, quickOpenFiles).map(toPaletteItem);
    }

    function toPaletteItem(file: QuickOpenFile): PaletteItem {
        return { id: file.path, name: file.name, detail: file.detail };
    }

    function rankCommandRows(query: string): PaletteItem[] {
        return rankCommands(query);
    }

    function showCommandPalette(): void {
        quickOpen = false;
        commandPalette = true;
    }

    function closeCommandPalette(): void {
        commandPalette = false;
    }

    function saveActive(saveAs = false): void {
        if (activeId !== '') {
            void saveDocument(activeId, saveAs);
        }
    }

    // selectCommand closes the palette before running the command, so commands
    // that open another dialog (Quick Open, the folder picker) start clean.
    function selectCommand(id: string): void {
        commandPalette = false;
        switch (id) {
            case 'new-file':
                newFile();
                break;
            case 'open-file':
                void openFromDialog();
                break;
            case 'save':
                saveActive();
                break;
            case 'save-as':
                saveActive(true);
                break;
            case 'save-all':
                void saveAll();
                break;
            case 'print':
                void printActive();
                break;
            case 'quick-open':
                void showQuickOpen();
                break;
            case 'add-folder':
                void addFolderFromDialog();
                break;
            case 'close-tab':
                if (activeId !== '') {
                    void closeTab(activeId);
                }
                break;
            case 'settings':
                void openSettings();
                break;
            case 'install-command-line-tool':
                void installCommandLineTool();
                break;
        }
    }

    async function addFolderFromDialog(): Promise<void> {
        if (busy) {
            return;
        }
        busy = true;
        try {
            const path = await openFolderDialog();
            // Cancelling the dialog is a decision, not a failure.
            if (path === '') {
                return;
            }
            folders = await addSidebarFolder(path);
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        } finally {
            busy = false;
        }
    }

    // Removing a root only drops it from the sidebar: tabs stay open, because
    // they hold their own buffers and are closed from the tab bar.
    async function removeFolder(path: string): Promise<void> {
        try {
            folders = await removeSidebarFolder(path);
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // Go decides whether closing needs a prompt, so it is told whenever the
    // answer changes. A failed report only costs a redundant prompt.
    $effect(() => {
        void SetUnsavedChanges(unsaved).catch((error) =>
            console.error('Failed to report unsaved changes.', error),
        );
    });

    onMount(() => {
        let unsubscribe: (() => void) | undefined;
        let unsubscribeMenu: (() => void) | undefined;
        let unsubscribeClose: (() => void) | undefined;
        let unsubscribeQuickOpen: (() => void) | undefined;
        let unsubscribeCommandPalette: (() => void) | undefined;
        let unsubscribeFileCommands: (() => void) | undefined;
        let unsubscribeCloseRequest: (() => void) | undefined;
        let disposed = false;

        void (async () => {
            folders = await loadSidebarFolders();
            const off = await listenForLaunch((request) => void applyLaunch(request));
            if (disposed) {
                off();
                return;
            }
            unsubscribe = off;
            unsubscribeMenu = listenForAddFolder(() => void addFolderFromDialog());
            unsubscribeClose = listenForCloseTab(() => {
                if (activeId !== '') {
                    void closeTab(activeId);
                }
            });
            unsubscribeQuickOpen = listenForQuickOpen(() => void showQuickOpen());
            unsubscribeCommandPalette = listenForCommandPalette(showCommandPalette);
            unsubscribeFileCommands = listenForFileCommands({
                'new-file': newFile,
                'open-file': () => void openFromDialog(),
                'save': () => saveActive(),
                'save-as': () => saveActive(true),
                'save-all': () => void saveAll(),
                'print': () => void printActive(),
            });
            unsubscribeCloseRequest = listenForCloseRequest(() => void confirmClose());
        })();

        return () => {
            disposed = true;
            unsubscribe?.();
            unsubscribeMenu?.();
            unsubscribeClose?.();
            unsubscribeQuickOpen?.();
            unsubscribeCommandPalette?.();
            unsubscribeFileCommands?.();
            unsubscribeCloseRequest?.();
        };
    });
</script>

<div class="app">
    <TitleBar appName={APP_NAME} fileName={shownName(active)} folderLabel={folderLabel} />
    <div class="app__body">
        <Sidebar
            {folders}
            selectedPath={active?.path ?? ''}
            {busy}
            onAddFolder={() => void addFolderFromDialog()}
            onRemoveFolder={(path) => void removeFolder(path)}
            onOpenFile={(path) => void openFile(path)}
        />
        <div class="app__editor">
            <TabBar
                {tabs}
                activeId={activeId}
                onSelect={(id) => {
                    activeId = id;
                }}
                onClose={(id) => void closeTab(id)}
            />
            <div class="app__pane">
                <div
                    class="app__pane-layer"
                    class:app__pane-layer--hidden={imageActive}
                    inert={imageActive}
                >
                    <EditorPane
                        bind:this={editorPane}
                        modelKey={editorKey}
                        content={editorContent}
                        {openIds}
                        onDirtyChange={(id, dirty) => {
                            tabs = setDocumentDirty(tabs, id, dirty);
                        }}
                        onCursorMove={(nextLine, nextColumn) => {
                            line = nextLine;
                            column = nextColumn;
                        }}
                    />
                </div>
                {#if active?.kind === 'image'}
                    {#key active.src}
                        <ImageViewer src={active.src} name={active.name} />
                    {/key}
                {/if}
            </div>
        </div>
    </div>
    <StatusBar {line} {column} language={imageActive ? 'Image' : LANGUAGE} {notice} {folderLabel} />
    {#if quickOpen}
        <Palette
            title="Quick Open"
            placeholder="Search files by name or path"
            rank={rankFiles}
            empty={quickOpenEmpty}
            onSelect={(path) => void openFromQuickOpen(path)}
            onClose={closeQuickOpen}
        />
    {/if}
    {#if commandPalette}
        <Palette
            title="Command Palette"
            placeholder="Type a command"
            rank={rankCommandRows}
            empty="No matching commands."
            onSelect={selectCommand}
            onClose={closeCommandPalette}
        />
    {/if}
    {#if unsavedPrompt !== null}
        <Prompt
            title={unsavedPrompt.title}
            message={unsavedPrompt.message}
            details={unsavedPrompt.details}
            actions={[
                { id: 'save', label: unsavedPrompt.confirmLabel, primary: true },
                { id: 'discard', label: "Don't Save" },
                { id: 'cancel', label: 'Cancel' },
            ]}
            onSelect={answerUnsaved}
            onCancel={() => answerUnsaved('cancel')}
        />
    {/if}
    {#if conflictPrompt !== null}
        <Prompt
            title="File Changed on Disk"
            message={conflictPrompt.message}
            actions={[
                { id: 'overwrite', label: 'Overwrite', primary: true },
                { id: 'cancel', label: 'Cancel' },
            ]}
            onSelect={(id) => answerConflict(id === 'overwrite')}
            onCancel={() => answerConflict(false)}
        />
    {/if}
</div>

<style>
    .app {
        display: flex;
        flex-direction: column;
        height: 100vh;
    }

    .app__body {
        display: flex;
        flex: 1 1 auto;
        min-height: 0;
    }

    .app__editor {
        display: flex;
        flex-direction: column;
        flex: 1 1 auto;
        min-width: 0;
        min-height: 0;
    }

    .app__pane {
        position: relative;
        flex: 1 1 auto;
        min-height: 0;
    }

    /* The editor stays laid out behind the viewer; visibility keeps Monaco's
       size stable so returning to a text tab restores the same viewport. */
    .app__pane-layer {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
    }

    .app__pane-layer--hidden {
        visibility: hidden;
        pointer-events: none;
    }
</style>
