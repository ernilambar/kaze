<script lang="ts">
    import { onMount } from 'svelte';
    import EditorPane from './EditorPane.svelte';
    import Sidebar from './Sidebar.svelte';
    import StatusBar from './StatusBar.svelte';
    import TitleBar from './TitleBar.svelte';
    import {
        listenForLaunch,
        readDocument,
        shownName,
        type LaunchRequest,
        type OpenDocument,
    } from './document';
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

    const APP_NAME = 'Kova';
    const LANGUAGE = 'Plain Text';

    let current = $state<OpenDocument>();
    let folders = $state<string[]>([]);
    let notice = $state('');
    let busy = $state(false);

    let line = $state(1);
    let column = $state(1);

    let folderLabel = $derived(sidebarLabel(folders));

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
        notice = failure !== '' ? failure : noticeFor(request);

        const path = request.files[0];
        if (path === undefined) {
            return;
        }
        const opened = await readDocument(path);
        if (opened !== undefined) {
            current = opened;
        }
    }

    async function openFile(path: string): Promise<void> {
        const opened = await readDocument(path);
        if (opened === undefined) {
            notice = `Cannot open ${path}`;
            return;
        }
        current = opened;
        notice = '';
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

    // Removing a root only drops it from the sidebar: an open file stays open,
    // because the editor holds its content and there is no tab to close.
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
        })();

        return () => {
            disposed = true;
            unsubscribe?.();
            unsubscribeMenu?.();
        };
    });
</script>

<div class="app">
    <TitleBar appName={APP_NAME} fileName={shownName(current)} folderLabel={folderLabel} />
    <div class="app__body">
        <Sidebar
            {folders}
            selectedPath={current?.path ?? ''}
            {busy}
            onAddFolder={() => void addFolderFromDialog()}
            onRemoveFolder={(path) => void removeFolder(path)}
            onOpenFile={(path) => void openFile(path)}
        />
        <EditorPane
            content={current?.content}
            onCursorMove={(nextLine, nextColumn) => {
                line = nextLine;
                column = nextColumn;
            }}
        />
    </div>
    <StatusBar {line} {column} language={LANGUAGE} {notice} {folderLabel} />
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
</style>
