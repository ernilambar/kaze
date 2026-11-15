<script lang="ts">
    import { onMount } from 'svelte';
    import EditorPane from './EditorPane.svelte';
    import Palette from './Palette.svelte';
    import Sidebar from './Sidebar.svelte';
    import StatusBar from './StatusBar.svelte';
    import TabBar from './TabBar.svelte';
    import TitleBar from './TitleBar.svelte';
    import { listenForCommandPalette, rankCommands } from './commands';
    import {
        listenForLaunch,
        readDocument,
        shownName,
        type LaunchRequest,
        type OpenDocument,
    } from './document';
    import { WELCOME_KEY, WELCOME_TEXT } from './editor';
    import { InstallCommandLineTool, OpenSettings } from '../wailsjs/go/main/App';
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
    import { closeDocumentTab, listenForCloseTab, openDocumentTab } from './tabs';

    const APP_NAME = 'Kova';
    const LANGUAGE = 'Plain Text';

    let tabs = $state<OpenDocument[]>([]);
    let activePath = $state('');
    let folders = $state<string[]>([]);
    let notice = $state('');
    let busy = $state(false);

    let quickOpen = $state(false);
    let quickOpenFiles = $state<QuickOpenFile[]>([]);
    let quickOpenLoading = $state(false);
    let commandPalette = $state(false);

    let line = $state(1);
    let column = $state(1);

    let active = $derived(tabs.find((tab) => tab.path === activePath));
    let openPaths = $derived(tabs.map((tab) => tab.path));
    let folderLabel = $derived(sidebarLabel(folders));
    let quickOpenEmpty = $derived(
        quickOpenLoading
            ? 'Loading files…'
            : quickOpenFiles.length === 0
              ? 'Add a folder to the sidebar to search its files.'
              : 'No matching files.',
    );

    // showDocument adds a tab for the document unless its path is already open,
    // then makes it active.
    function showDocument(document: OpenDocument): void {
        tabs = openDocumentTab(tabs, document);
        activePath = document.path;
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
            if (tabs.some((tab) => tab.path === path)) {
                activePath = path;
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
        if (tabs.some((tab) => tab.path === path)) {
            activePath = path;
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

    function closeTab(path: string): void {
        const result = closeDocumentTab(tabs, path, activePath);
        tabs = result.tabs;
        activePath = result.activePath;
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

    // selectCommand closes the palette before running the command, so commands
    // that open another dialog (Quick Open, the folder picker) start clean.
    function selectCommand(id: string): void {
        commandPalette = false;
        switch (id) {
            case 'quick-open':
                void showQuickOpen();
                break;
            case 'add-folder':
                void addFolderFromDialog();
                break;
            case 'close-tab':
                if (activePath !== '') {
                    closeTab(activePath);
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

    onMount(() => {
        let unsubscribe: (() => void) | undefined;
        let unsubscribeMenu: (() => void) | undefined;
        let unsubscribeClose: (() => void) | undefined;
        let unsubscribeQuickOpen: (() => void) | undefined;
        let unsubscribeCommandPalette: (() => void) | undefined;
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
                if (activePath !== '') {
                    closeTab(activePath);
                }
            });
            unsubscribeQuickOpen = listenForQuickOpen(() => void showQuickOpen());
            unsubscribeCommandPalette = listenForCommandPalette(showCommandPalette);
        })();

        return () => {
            disposed = true;
            unsubscribe?.();
            unsubscribeMenu?.();
            unsubscribeClose?.();
            unsubscribeQuickOpen?.();
            unsubscribeCommandPalette?.();
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
                activePath={activePath}
                onSelect={(path) => {
                    activePath = path;
                }}
                onClose={closeTab}
            />
            <EditorPane
                modelKey={active?.path ?? WELCOME_KEY}
                content={active?.content ?? WELCOME_TEXT}
                {openPaths}
                onCursorMove={(nextLine, nextColumn) => {
                    line = nextLine;
                    column = nextColumn;
                }}
            />
        </div>
    </div>
    <StatusBar {line} {column} language={LANGUAGE} {notice} {folderLabel} />
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
</style>
