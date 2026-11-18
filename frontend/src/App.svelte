<script lang="ts">
    import { onMount } from 'svelte';
    import {
        BringAllToFront,
        CloseWindow,
        CopyText,
        DefaultSettingsFile,
        GetProductName,
        InstallCommandLineTool,
        KeybindingsFile,
        MinimiseWindow,
        NewWindow,
        OpenFolderInNewWindow,
        OpenThemesFolder,
        PrintDocument,
        RevealFile,
        SetAppearanceMode,
        SetUnsavedChanges,
        SettingsFile,
        ShouldRestoreSession,
        ShowAbout,
        ToggleFullScreen,
        WatchConfig,
        ZoomWindow,
    } from '../bindings/kaze/app';
    import type { WorkspaceInfo } from '../bindings/kaze/models';
    import { Appearance } from './app/appearance.svelte';
    import { WindowContext } from './app/context.svelte';
    import { dispatchCommand, type CommandHandlers } from './app/dispatch';
    import { Documents } from './app/documents.svelte';
    import { subscribeAll } from './app/events';
    import { GitController } from './app/git.svelte';
    import { contextMenuItems as buildContextMenuItems } from './app/menus';
    import { Prompts } from './app/prompts.svelte';
    import { SearchController } from './app/search.svelte';
    import { type CaseTransform } from './case';
    import ChangesView from './ChangesView.svelte';
    import ContextMenu from './ContextMenu.svelte';
    import EditorPane from './EditorPane.svelte';
    import NamePrompt from './NamePrompt.svelte';
    import ImageViewer from './ImageViewer.svelte';
    import Preview from './Preview.svelte';
    import Palette from './Palette.svelte';
    import Prompt from './Prompt.svelte';
    import SearchPanel from './SearchPanel.svelte';
    import SettingsOverlay from './SettingsOverlay.svelte';
    import Sidebar from './Sidebar.svelte';
    import StatusBar from './StatusBar.svelte';
    import TabBar from './TabBar.svelte';
    import TitleBar from './TitleBar.svelte';
    import Welcome from './Welcome.svelte';
    import { INDENT_PALETTE_COMMANDS, rankCommands } from './commands';
    import type { ContextMenuItem } from './contextmenu';
    import {
        readDocument,
        shownPath,
        toBufferDocument,
        type DocumentEOL,
        type LaunchRequest,
        type OpenDocument,
    } from './document';
    import {
        AUTO_LANGUAGE_ID,
        installEditorKeybindings,
        languageChoices,
        languageForDocument,
        languageLabel,
        WELCOME_KEY,
    } from './editor';
    import {
        INDENT_COMMANDS,
        type SelectionCommand,
        type ZoomCommand,
    } from './editorcommands';
    import {
        copyPath,
        createFile,
        createFolder,
        duplicatePath,
        duplicatePathSuggestion,
        entryName,
        movePath,
        parentPath,
        renamePath,
        trashPath,
    } from './filetree';
    import {
        type SidebarView,
    } from './git';
    import type { PaletteItem } from './palette';
    import { previewKindFor, type PreviewKind } from './preview';
    import {
        listFiles,
        rankAll,
        rankCandidates,
        rankFileRows,
        toQuickOpenFiles,
        toRecentItems as toRecentPaletteItems,
        type QuickOpenFile,
    } from './quickOpen';
    import {
        loadEditorSettings,
        loadPreferences,
    } from './settings';
    import {
        applyUISettings,
    } from './theme';
    import {
        addSidebarFolder,
        closeAllSidebarFolders,
        errorMessage,
        loadSidebarFolders,
        openFolderDialog,
        relativePath,
        removeSidebarFolder,
        setSidebarFolders,
        sidebarLabel,
    } from './sidebar';
    import { indentationLabel, noticeFor } from './status';
    import {
        findDocumentByPath,
        setDocumentDirty,
        pathWithin,
        retargetDocumentPaths,
        tabsToClose,
        updateDocument,
    } from './tabs';
    import type { LineTransform } from './text';
    import {
        clearRecentFiles,
        loadWindowSession,
        loadWorkspace,
        loadWorkspaceState,
        openWorkspaceDialog,
        rememberRecentFile,
        saveSession,
        saveSidebarAutoReveal,
        saveSidebarVisible,
        saveWorkspaceAs,
        toWorkspaceSession,
        watchPaths,
        type SavedWindowSession,
        type SavedWorkspaceSession,
    } from './workspace';
    import type { CopyPathKind } from './workspacecommands';
    import { resolveWindowName, setWindowName } from './windowstate';

    // PreviewState is the buffer a preview is showing. It is a snapshot: the
    // editor is hidden while the preview is up, so nothing can change underneath
    // it, and switching tabs clears it.
    interface PreviewState {
        id: string;
        kind: PreviewKind;
        content: string;
    }

    // appName comes from the build config; it is empty only until mount.
    let appName = $state('');

    // activeWorkspace is the file the roots came from. It is session state: the
    // roots themselves are what settings.json restores.
    let activeWorkspace = $state<WorkspaceInfo | null>(null);

    // Git is read-only. The sidebar shows the status marks from gitChanges,
    // while the status bar shows the active file's branch.
    let sidebarView = $state<SidebarView>('explorer');

    let settingsOverlay = $state(false);
    let preview = $state<PreviewState | null>(null);

    let quickOpen = $state(false);
    let quickOpenFiles = $state<QuickOpenFile[]>([]);
    let quickOpenLoading = $state(false);
    let quickOpenTruncated = $state(false);
    let commandPalette = $state(false);
    let recentOpen = $state(false);

    // Workspace state that outlives a window.
    let recentFiles = $state<string[]>([]);
    let sidebarVisible = $state(true);
    let sidebarAutoReveal = $state(true);
    // revealedPath is the row the tree expands to. It follows the active tab
    // only while auto-reveal is on.
    let revealedPath = $state('');
    let expandedPaths = $state<string[]>([]);
    let treeRefreshToken = $state(0);
    let sessionReady = $state(false);
    // Only the window that restored the saved tabs owns session.json's launch
    // entry. Later windows have their own tabs and must not replace it, though
    // they still park their workspace tab sets.
    let sessionOwner = $state(false);
    // sessionSuspended stops the live session writer while a workspace switch
    // or close is moving tabs, so the state between the two tab lists is never
    // written to disk. It is reactive because the save effect returns before it
    // subscribes to the tab list when suspended; clearing it must run the
    // effect again.
    let sessionSuspended = $state(false);

    let imageSize = $state<{ id: string; width: number; height: number } | null>(null);

    // contextMenu is the right-click menu in the tab bar or the tree, anchored
    // where the pointer was. tabId marks a tab; otherwise path is the tree entry
    // under the pointer. isDir decides which items the tree offers.
    let contextMenu = $state<{
        x: number;
        y: number;
        path: string;
        isDir: boolean;
        tabId?: string;
    } | null>(null);

    // clipboardPath is the tree entry Cut or Copy put aside. The operating
    // system clipboard is not used: moving a file is not the same thing as
    // copying its path, and the tree only ever moves one entry.
    let clipboardPath = $state('');
    let clipboardCut = $state(false);

    // busyPath is the entry a tree command is running on, so its row can show
    // that something is happening instead of the command looking ignored.
    let busyPath = $state('');

    let line = $state(1);
    let column = $state(1);
    let selectionLength = $state(0);
    // indentation is what the active buffer indents with. The editor pane
    // reports it when the buffer is attached and whenever its model options
    // change: detection, the settings and the picker all arrive that way.
    let indentation = $state({ insertSpaces: true, tabSize: 4, indentSize: 4 });
    let languagePalette = $state(false);
    // indentationPalette is the list the status bar's indentation item opens;
    // it offers the same actions the command palette groups as Indentation.
    let indentationPalette = $state(false);
    // languageModeItems is built on first use, after Monaco has registered its
    // languages, and reused for every keystroke in the mode palette.
    let languageModeItems: PaletteItem[] | undefined;

    let editorPane = $state<EditorPane>();

    // The window's controllers. They are built here, during component init,
    // and each one is handed the context and the controllers it uses. Host
    // callbacks are the functions that still live in this component.
    const ctx = new WindowContext(() => editorPane);
    const prompts = new Prompts();
    // git and search reach documents through closures: documents is built
    // last because it uses both, and the closures only run after init.
    const git = new GitController(ctx, {
        showDocument: (document, record) => documents.showDocument(document, record),
    });
    const search = new SearchController(ctx, prompts, git, {
        openFile: (path) => documents.openFile(path),
        refreshTree,
    });
    const appearance = new Appearance(ctx, {
        refreshTree,
        sidebarVisible: () => sidebarVisible,
        selectCommand,
    });
    const documents = new Documents(ctx, prompts, git, appearance, {
        appName: () => appName,
        recordRecent,
        refreshTree,
    });

    let imageActive = $derived(ctx.active?.kind === 'image');
    let previewActive = $derived(preview !== null && preview.id === ctx.activeId);
    // Either surface replaces the editor, which stays laid out behind it so a
    // text tab keeps its viewport. The welcome screen is the same layer when no
    // tab is open.
    let viewerActive = $derived(imageActive || previewActive);
    let welcomeActive = $derived(ctx.active === undefined && !ctx.searchActive);
    let paneReplaced = $derived(viewerActive || ctx.searchActive || welcomeActive);
    let language = $derived(
        imageActive
            ? 'Image'
            : ctx.active?.kind === 'diff'
              ? 'Diff'
              : languageLabel(ctx.active?.path ?? '', ctx.active?.content ?? '', ctx.active?.language ?? ''),
    );
    // Monaco keeps its buffers while an image tab is showing, so switching back
    // to a text tab preserves edits. The idle buffer keeps the hidden editor
    // from creating a model for an image path.
    let editorKey = $derived(imageActive ? WELCOME_KEY : (ctx.active?.id ?? WELCOME_KEY));
    let editorContent = $derived(imageActive ? '' : (ctx.active?.content ?? ''));
    // The active file's line ending decides the Monaco model's EOL, so edits
    // and the saved text use the same sequence the file started with.
    let editorEol = $derived(imageActive ? 'lf' : (ctx.active?.eol ?? 'lf'));
    // The status bar prints the active buffer's indentation; a tab with no
    // text buffer has none to configure.
    let indentationText = $derived(
        ctx.active?.kind === 'text'
            ? indentationLabel(indentation.insertSpaces, indentation.indentSize, indentation.tabSize)
            : '',
    );
    // savedContent is what a clean buffer would hold. A moved dirty tab carries
    // a different text than the file on disk, so its tab records the baseline.
    let editorSavedContent = $derived(
        imageActive ? '' : (ctx.active?.savedContent ?? ctx.active?.content ?? ''),
    );
    let openIds = $derived(ctx.tabs.map((tab) => tab.id));
    let folderLabel = $derived(
        activeWorkspace !== null && activeWorkspace.name !== ''
            ? activeWorkspace.name
            : sidebarLabel(ctx.folders),
    );
    let imageSizeLabel = $derived(
        imageSize !== null && imageSize.id === ctx.activeId
            ? `${imageSize.width} × ${imageSize.height}`
            : '',
    );
    let recentItems = $derived(toRecentPaletteItems(recentFiles, ctx.folders));
    // The session records file-backed tabs only: an untitled buffer has no
    // text on disk, and a diff is a view of a file that is already recorded.
    let sessionFiles = $derived(
        ctx.tabs.filter((tab) => tab.kind === 'text' && tab.path !== '').map((tab) => tab.path),
    );
    // A diff tab records the file it shows, so a relaunch restores the file
    // itself as the active tab.
    let sessionActive = $derived(ctx.active?.path || ctx.active?.sourcePath || '');
    let watchedPaths = $derived(Array.from(new Set([...sessionFiles, ...expandedPaths])));
    let quickOpenEmpty = $derived(
        quickOpenLoading
            ? 'Loading files…'
            : quickOpenFiles.length === 0
              ? 'Add a folder to the sidebar to search its files.'
              : 'No matching files.',
    );

    // addFolders saves each root from a launch request and adopts the result.
    // The backend deduplicates, so relaunching with the same folder is a no-op.
    // The first failure is returned so the caller can show it in the status bar.
    async function addFolders(paths: string[]): Promise<string> {
        let failure = '';
        let latest = ctx.folders;
        for (const path of paths) {
            try {
                latest = await addSidebarFolder(ctx.windowName, path);
            } catch (error) {
                if (failure === '') {
                    failure = errorMessage(error);
                }
            }
        }
        ctx.folders = latest;
        void git.refreshGitStatus();
        return failure;
    }

    async function applyLaunch(request: LaunchRequest): Promise<void> {
        let failure = '';
        if (request.replace_folders) {
            // Open Folder in New Window: the new window starts with exactly the
            // chosen folder, and the roots are saved for the next launch.
            try {
                ctx.folders = await setSidebarFolders(ctx.windowName, request.folders);
                void git.refreshGitStatus();
            } catch (error) {
                failure = errorMessage(error);
            }
        } else if (request.folders.length > 0) {
            failure = await addFolders(request.folders);
        }

        // A tab moved from another window arrives as a buffer. Text keeps the
        // text the other window held, so unsaved edits survive the move; an
        // image or diff is read again from its path.
        for (const buffer of request.buffers) {
            if (buffer.kind === 'image' && buffer.path !== '') {
                await documents.openFile(buffer.path);
                continue;
            }
            if (buffer.kind === 'diff' && buffer.source_path !== '') {
                await git.openDiff(buffer.source_path);
                continue;
            }
            documents.showDocument(toBufferDocument(buffer), false);
        }

        for (const path of request.files) {
            const existing = findDocumentByPath(ctx.tabs, path);
            if (existing !== undefined) {
                ctx.activeId = existing.id;
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                if (failure === '') {
                    failure = `Cannot open ${path}`;
                }
                continue;
            }
            documents.showDocument(opened);
        }

        ctx.notice = failure !== '' ? failure : noticeFor(request);
    }

    // switchSidebarView moves between the explorer and the changes list. The
    // switch lives in the status bar, which stays visible when the sidebar does
    // not, so choosing a view brings the sidebar back.
    function switchSidebarView(view: SidebarView): void {
        sidebarView = view;
        if (!sidebarVisible) {
            sidebarVisible = true;
            void saveSidebarVisible(true).catch((error) =>
                console.error('Failed to save the sidebar state.', error),
            );
        }
    }


    // toggleLineEnding switches the active text buffer's line endings. The
    // model edit makes the tab dirty, so the next save writes the chosen
    // sequence instead of the one the file had.
    function toggleLineEnding(): void {
        const tab = ctx.active;
        if (tab === undefined || tab.kind !== 'text') {
            return;
        }
        const eol: DocumentEOL = tab.eol === 'crlf' ? 'lf' : 'crlf';
        editorPane?.setEol(tab.id, eol);
        ctx.tabs = updateDocument(ctx.tabs, tab.id, { eol });
    }

    // newFileFromMenu is File → New File: the file is created inside a sidebar
    // folder, so it exists on disk from the start. The folder is the one the
    // selected entry lives in; with no folders there is nowhere to create one,
    // so the menu opens a buffer instead rather than doing nothing.
    async function newFileFromMenu(): Promise<void> {
        const folder = selectedFolder();
        if (folder === '') {
            ctx.notice = 'Add a folder to the sidebar to create a file.';
            documents.newFile();
            return;
        }
        await newFileInFolder(folder);
    }

    // newFolderFromMenu is File → New Folder, in the selected entry's folder.
    async function newFolderFromMenu(): Promise<void> {
        const folder = selectedFolder();
        if (folder === '') {
            ctx.notice = 'Add a folder to the sidebar to create a folder.';
            return;
        }
        await newFolderInFolder(folder);
    }

    // selectedFolder is the folder a menu command acts on. The tree highlights
    // the entry the command left behind, so a selected file means the folder it
    // lives in and a selected folder means itself; with nothing selected the
    // first root is the workspace's default.
    function selectedFolder(): string {
        const selected = revealedPath;
        if (selected === '') {
            return ctx.folders[0] ?? '';
        }
        const tab = findDocumentByPath(ctx.tabs, selected);
        if (tab !== undefined && tab.path !== '') {
            return parentPath(selected);
        }
        return selected;
    }

    async function openSettings(): Promise<void> {
        try {
            await documents.openFile(await SettingsFile());
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    async function installCommandLineTool(): Promise<void> {
        try {
            await InstallCommandLineTool();
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    function showSettingsOverlay(): void {
        languagePalette = false;
        indentationPalette = false;
        settingsOverlay = true;
    }

    function closeSettingsOverlay(): void {
        settingsOverlay = false;
    }

    async function openDefaultSettings(): Promise<void> {
        try {
            await documents.openFile(await DefaultSettingsFile());
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    async function openThemesFolder(): Promise<void> {
        try {
            await OpenThemesFolder();
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    async function openKeybindings(): Promise<void> {
        try {
            await documents.openFile(await KeybindingsFile());
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    // printActive prints the active buffer: text comes from the editor so an
    // unsaved edit prints as it looks, pictures come from their file.
    async function printActive(): Promise<void> {
        const tab = ctx.active;
        if (tab === undefined) {
            ctx.notice = 'Open a document to print.';
            return;
        }
        try {
            await PrintDocument({
                path: tab.path,
                content: tab.kind === 'image' ? '' : documents.bufferText(tab),
                image: tab.kind === 'image',
            });
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    // togglePreview shows the active buffer as Markdown or SVG. The preview is a
    // snapshot of the buffer; the hidden editor keeps its model untouched, so
    // hiding the preview returns to the same viewport.
    function togglePreview(): void {
        if (previewActive) {
            preview = null;
            return;
        }
        const tab = ctx.active;
        const kind = previewKindFor(tab?.path ?? '');
        if (tab === undefined || kind === '') {
            ctx.notice = 'Preview is available for Markdown and SVG files.';
            return;
        }
        preview = { id: tab.id, kind, content: documents.bufferText(tab) };
        ctx.notice = '';
    }

    function transformSelection(transform: CaseTransform): void {
        editorPane?.transformSelection(transform);
    }

    function trimWhitespace(): void {
        editorPane?.trimTrailingWhitespace();
    }

    function transformLines(command: LineTransform): void {
        editorPane?.transformLines(command);
    }

    function runSelectionCommand(command: SelectionCommand): void {
        editorPane?.runSelectionCommand(command);
    }

    // runEditorAction runs one of Monaco's own editor actions on the active
    // buffer: its Find and Replace widgets and the Go to Line prompt. Those
    // widgets belong to the editor, so the Find in Files results panel gives
    // way first rather than hiding them.
    function runEditorAction(action: string): void {
        if (ctx.active === undefined) {
            ctx.notice = 'Open a document first.';
            return;
        }
        ctx.searchActive = false;
        editorPane?.runEditorAction(action);
    }

    function zoom(command: ZoomCommand): void {
        editorPane?.zoom(command);
    }

    // showQuickOpen opens the palette first and fills it as soon as Go has
    // walked the sidebar roots, so the input is usable while a large tree is
    // still being read.
    async function showQuickOpen(): Promise<void> {
        commandPalette = false;
        recentOpen = false;
        languagePalette = false;
        indentationPalette = false;
        quickOpen = true;
        quickOpenLoading = true;
        quickOpenTruncated = false;
        try {
            const list = await listFiles(ctx.windowName);
            quickOpenFiles = toQuickOpenFiles(list.files, ctx.folders);
            quickOpenTruncated = list.truncated;
        } catch (error) {
            quickOpenFiles = [];
            ctx.notice = errorMessage(error);
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
        await documents.openFile(path);
    }

    // The two palettes share Palette.svelte, so Quick Open rows are mapped onto
    // the common row shape and ranked with the same function.
    function rankFiles(query: string): PaletteItem[] {
        return rankFileRows(query, quickOpenFiles);
    }

    function rankCommandRows(query: string): PaletteItem[] {
        return rankCommands(query);
    }

    // Change Language Mode lists Auto Detect and every language Monaco
    // registered. It is only meaningful for a text tab: a diff has its own
    // fixed syntax and an image has none.
    function rankLanguages(query: string): PaletteItem[] {
        const items = languageModeItems ?? (languageModeItems = languageChoices());
        return rankAll(query, items);
    }

    function showLanguagePalette(): void {
        if (ctx.active === undefined || ctx.active.kind !== 'text') {
            return;
        }
        quickOpen = false;
        recentOpen = false;
        commandPalette = false;
        indentationPalette = false;
        languagePalette = true;
    }

    function chooseLanguage(id: string): void {
        languagePalette = false;
        const tab = ctx.active;
        if (tab === undefined || tab.kind !== 'text') {
            return;
        }
        const override = id === AUTO_LANGUAGE_ID ? undefined : id;
        ctx.tabs = updateDocument(ctx.tabs, tab.id, { language: override });
        const content = editorPane?.value(tab.id) ?? tab.content;
        editorPane?.setLanguage(tab.id, languageForDocument(tab.path, content, override ?? ''));
    }

    // rankIndentation lists the actions the status bar's picker offers. The
    // rows are static, so an empty query shows the whole list.
    function rankIndentation(query: string): PaletteItem[] {
        return rankAll(query, INDENT_PALETTE_COMMANDS);
    }

    function showIndentationPalette(): void {
        if (ctx.active === undefined || ctx.active.kind !== 'text') {
            return;
        }
        quickOpen = false;
        recentOpen = false;
        commandPalette = false;
        languagePalette = false;
        indentationPalette = true;
    }

    // runIndentAction runs one of Monaco's indentation actions on the active
    // buffer. The actions live on the model, so only the showing tab's text
    // and options change.
    function runIndentAction(id: string): void {
        const spec = INDENT_COMMANDS.find((entry) => entry.id === id);
        if (spec === undefined || ctx.active === undefined || ctx.active.kind !== 'text') {
            return;
        }
        ctx.searchActive = false;
        editorPane?.runEditorAction(spec.action);
    }

    function chooseIndentation(id: string): void {
        indentationPalette = false;
        runIndentAction(id);
    }

    function showCommandPalette(): void {
        quickOpen = false;
        recentOpen = false;
        languagePalette = false;
        indentationPalette = false;
        commandPalette = true;
    }

    function closeCommandPalette(): void {
        commandPalette = false;
    }

    function saveActive(saveAs = false): void {
        if (ctx.activeId !== '') {
            void documents.saveDocument(ctx.activeId, saveAs);
        }
    }

    // runWindowCommand reports a failed native window action in the status bar.
    function runWindowCommand(command: () => Promise<void>): void {
        void command().catch((error) => {
            ctx.notice = errorMessage(error);
        });
    }

    // selectCommand closes the palette before running the command, so commands
    // that open another dialog (Quick Open, the folder picker) start clean. The
    // command id table lives in app/dispatch.ts; this object is what each id
    // runs in this window.
    function selectCommand(id: string): void {
        commandPalette = false;
        dispatchCommand(id, commandHandlers);
    }

    // commandHandlers is the window's side of every palette, menu and
    // keybinding command. It is built once because the functions it names are
    // stable for the life of the window.
    const commandHandlers: CommandHandlers = {
        newFile: () => void newFileFromMenu(),
        newFolder: () => void newFolderFromMenu(),
        newBuffer: documents.newFile,
        openFile: () => void documents.openFromDialog(),
        save: () => saveActive(),
        saveAs: () => saveActive(true),
        saveAll: () => void documents.saveAll(),
        print: () => void printActive(),
        trimWhitespace,
        transformSelection,
        transformLines,
        runSelectionCommand,
        runIndentAction,
        runEditorAction,
        toggleViewOption: appearance.toggleViewOption,
        showLanguagePalette,
        showQuickOpen: () => void showQuickOpen(),
        showCommandPalette,
        showRecentFiles,
        clearRecent: () => void clearRecent(),
        showFindInFiles,
        showSettings: showSettingsOverlay,
        showAbout: () => void ShowAbout(),
        stepFindResult: (direction) => void search.stepFindResult(direction),
        togglePreview,
        switchSidebarView,
        zoom,
        addFolder: () => void addFolderFromDialog(),
        closeActiveTab: () => {
            if (ctx.activeId !== '') {
                void documents.closeTab(ctx.activeId);
            }
        },
        goToTab: documents.goToTab,
        stepTab: documents.stepTab,
        reopenClosedTab: () => void documents.reopenClosedTab(),
        closeAllTabs: () => void closeAllTabs(),
        closeAllFolders: () => void closeAllFolders(),
        moveTabToNewWindow: () => void documents.moveTabToNewWindow(),
        openSettingsJson: () => void openSettings(),
        openDefaultSettings: () => void openDefaultSettings(),
        openThemesFolder: () => void openThemesFolder(),
        openKeybindings: () => void openKeybindings(),
        installCommandLineTool: () => void installCommandLineTool(),
        setAppearanceMode: (mode) => void appearance.changeThemeSettings({ ...appearance.themeSettings, mode }),
        openWorkspace: () => void openWorkspaceFromDialog(),
        saveWorkspaceAs: () => void saveWorkspaceAsDialog(),
        closeWorkspace: () => void closeWorkspace(),
        refreshWorkspace: () => void refreshWorkspace(),
        toggleSidebar,
        revealActiveFile: () => void revealActiveFile(),
        copyActivePath: (kind) => void copyActivePath(kind),
        minimiseWindow: () => runWindowCommand(() => MinimiseWindow(ctx.windowName)),
        zoomWindow: () => runWindowCommand(() => ZoomWindow(ctx.windowName)),
        toggleFullScreen: () => runWindowCommand(() => ToggleFullScreen(ctx.windowName)),
        newWindow: () => runWindowCommand(() => NewWindow()),
        openFolderInNewWindow: () => runWindowCommand(() => OpenFolderInNewWindow()),
        closeWindow: () => runWindowCommand(() => CloseWindow(ctx.windowName)),
        bringAllToFront: () => runWindowCommand(() => BringAllToFront()),
    };

    async function addFolderFromDialog(): Promise<void> {
        if (ctx.busy) {
            return;
        }
        ctx.busy = true;
        try {
            const path = await openFolderDialog();
            // Cancelling the dialog is a decision, not a failure.
            if (path === '') {
                return;
            }
            ctx.folders = await addSidebarFolder(ctx.windowName, path);
            void git.refreshGitStatus();
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        } finally {
            ctx.busy = false;
        }
    }

    // restoreWorkspaceTabs opens the file-backed tabs parked under a workspace.
    // Missing files are skipped, so one deleted file does not stop the rest.
    async function restoreWorkspaceTabs(session: SavedWorkspaceSession): Promise<void> {
        for (const path of session.openFiles) {
            if (findDocumentByPath(ctx.tabs, path) !== undefined) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            documents.showDocument(opened, false);
        }
        const restoredActive = findDocumentByPath(ctx.tabs, session.active);
        if (restoredActive !== undefined) {
            ctx.activeId = restoredActive.id;
        }
    }

    // clearWorkspaceTabs drops every document tab and the Find Results tab.
    // The old set has already been parked in session.json, and the Reopen
    // Closed Tab stack is cleared so it cannot pull tabs from another
    // workspace back into this one.
    function clearWorkspaceTabs(): void {
        ctx.tabs = [];
        ctx.activeId = '';
        documents.closedTabs = [];
        if (search.open) {
            search.closeSearchTab();
        }
    }

    // prepareWorkspaceSwitch is the one prompt before a switch replaces the
    // tab set. Saving all is the default because the parked tab set is a
    // convenience, not a place to discard edits.
    async function prepareWorkspaceSwitch(title: string, message: string): Promise<boolean> {
        const dirty = ctx.tabs.filter((tab) => tab.dirty);
        if (dirty.length === 0) {
            return true;
        }
        const answer = await prompts.askUnsaved({
            title,
            message,
            details: dirty.map((tab) => tab.path || tab.name),
            confirmLabel: 'Save All and Continue',
        });
        if (answer === 'cancel') {
            return false;
        }
        if (answer === 'save') {
            return await documents.saveAll();
        }
        return true;
    }

    // switchWorkspace parks the current tabs, replaces the roots and restores
    // the target workspace's parked tabs. The two tab lists never coexist in
    // the live session writer.
    async function switchWorkspace(target: WorkspaceInfo): Promise<boolean> {
        await persistSession();
        if (!(await prepareWorkspaceSwitch(
            'Switch Workspace',
            'Switching workspaces closes the open tabs. Save them first?',
        ))) {
            return false;
        }

        sessionSuspended = true;
        let switched = false;
        try {
            ctx.folders = await setSidebarFolders(ctx.windowName, target.folders ?? []);
            activeWorkspace = target;
            const parked = toWorkspaceSession(target.session);
            expandedPaths = parked.expanded;
            clearWorkspaceTabs();
            await restoreWorkspaceTabs(parked);
            switched = true;
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        } finally {
            sessionSuspended = false;
        }
        if (switched) {
            void git.refreshGitStatus();
            await persistSession();
        }
        return switched;
    }

    // openWorkspaceFromDialog replaces the window's workspace with the picked
    // file. The previous tab set stays parked under its own workspace.
    async function openWorkspaceFromDialog(): Promise<void> {
        if (ctx.busy || prompts.unsaved !== null || prompts.confirm !== null) {
            return;
        }
        ctx.busy = true;
        try {
            const workspace = await openWorkspaceDialog();
            if (workspace.path === '') {
                return;
            }
            await switchWorkspace(workspace);
        } catch (error) {
            ctx.notice = errorMessage(error);
        } finally {
            ctx.busy = false;
        }
    }

    // saveWorkspaceAsDialog writes the current roots to a workspace file, binds
    // the window to it and parks the current tabs under the new name.
    async function saveWorkspaceAsDialog(): Promise<void> {
        if (ctx.busy || prompts.unsaved !== null || prompts.confirm !== null) {
            return;
        }
        ctx.busy = true;
        try {
            const workspace = await saveWorkspaceAs(ctx.folders);
            if (workspace.path === '') {
                return;
            }
            activeWorkspace = workspace;
            await persistSession();
            ctx.notice = `Saved workspace ${workspace.name}`;
        } catch (error) {
            ctx.notice = errorMessage(error);
        } finally {
            ctx.busy = false;
        }
    }

    // closeWorkspace clears the roots and closes the tabs after the save
    // prompt. The parked tab set stays in session.json, so reopening the
    // workspace later restores it.
    async function closeWorkspace(): Promise<void> {
        if (ctx.busy || prompts.unsaved !== null || prompts.confirm !== null) {
            return;
        }
        if (activeWorkspace === null && ctx.folders.length === 0) {
            return;
        }
        ctx.busy = true;
        try {
            await persistSession();
            if (!(await prepareWorkspaceSwitch(
                'Close Workspace',
                'Closing the workspace closes its tabs. Save them first?',
            ))) {
                return;
            }

            sessionSuspended = true;
            try {
                ctx.folders = await closeAllSidebarFolders(ctx.windowName);
                activeWorkspace = null;
                expandedPaths = [];
                clearWorkspaceTabs();
                ctx.notice = '';
            } finally {
                sessionSuspended = false;
            }
            void git.refreshGitStatus();
            await persistSession();
        } catch (error) {
            ctx.notice = errorMessage(error);
        } finally {
            ctx.busy = false;
        }
    }

    // Removing a root only drops it from the sidebar: tabs stay open, because
    // they hold their own buffers and are closed from the tab bar.
    async function removeFolder(path: string): Promise<void> {
        try {
            ctx.folders = await removeSidebarFolder(ctx.windowName, path);
            if (ctx.folders.length === 0) {
                activeWorkspace = null;
            }
            void git.refreshGitStatus();
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    // ---- File tree operations -------------------------------------------------
    //
    // New File, New Folder, Rename, Duplicate, Cut/Copy/Paste, Delete (to the
    // Trash) and drag to move all run against the explorer. Every one of them
    // is a Go binding, because the file system is the backend's business; the
    // frontend only decides which entry, asks for a name and retargets the tabs
    // that follow a path.

    // refreshTree rereads the tree after a command changed what is on disk.
    function refreshTree(): void {
        treeRefreshToken++;
    }

    // applyTreeMove retargets every tab whose file the command moved. A buffer
    // keeps its text, edits and undo history; only the path it saves to changes.
    // The renamed file's git baseline and diff marks are reread, because HEAD
    // still knows the old name.
    function applyTreeMove(from: string, to: string): void {
        if (from === '' || to === '' || from === to) {
            return;
        }
        const previous = ctx.tabs;
        const next = retargetDocumentPaths(ctx.tabs, [{ from, to }]);
        if (next === previous) {
            return;
        }
        ctx.tabs = next;
        for (const tab of next) {
            const before = previous.find((entry) => entry.id === tab.id);
            if (before === undefined || before.path === tab.path) {
                continue;
            }
            // HEAD still knows the old name, so the gutter bars are recomputed
            // against the path the file now has.
            if (tab.kind === 'text') {
                void git.loadBaseline(tab.id, tab.path);
            }
        }
        // A diff tab followed its file, so the snapshot it shows is stale.
        if (next.some((tab) => tab.kind === 'diff')) {
            void git.refreshDiffTabs();
        }
        void git.refreshGitStatus();
    }

    // treeError reports a failed tree command in the status bar, where every
    // other notice appears, and keeps the tree as it was.
    function treeError(error: unknown): void {
        ctx.notice = errorMessage(error);
    }

    // newFileInFolder is New File from the tree: the file is created on disk
    // first, so an empty file exists even if the user never types into it, then
    // opened as a tab.
    async function newFileInFolder(folder: string): Promise<void> {
        await prompts.askName({
            title: 'New File',
            label: `Name for the new file in ${entryName(folder)}`,
            submitLabel: 'Create File',
            run: async (name) => {
                try {
                    const entry = await createFile(folder, name);
                    refreshTree();
                    await documents.openFile(entry.path);
                    ctx.notice = '';
                    return '';
                } catch (error) {
                    ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    }

    // newFolderInFolder is New Folder. The new folder is expanded so the user
    // sees where it went, which matters when it lands between many siblings.
    async function newFolderInFolder(folder: string): Promise<void> {
        await prompts.askName({
            title: 'New Folder',
            label: `Name for the new folder in ${entryName(folder)}`,
            submitLabel: 'Create Folder',
            run: async (name) => {
                try {
                    const entry = await createFolder(folder, name);
                    refreshTree();
                    revealInTree(entry.path);
                    ctx.notice = '';
                    return '';
                } catch (error) {
                    ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    }

    // renameEntry accepts a new name and, when the text carries a folder, a new
    // location too. The field holds the whole path with only the file stem
    // selected, so typing replaces the name and leaves the folder alone. The tab
    // that holds the file follows the new path, so the buffer stays open and
    // keeps its edits and undo history.
    async function renameEntry(path: string): Promise<void> {
        await prompts.askName({
            title: 'Rename',
            value: path,
            selectStem: true,
            submitLabel: 'Rename',
            run: async (target) => {
                try {
                    const entry = await renamePath(path, target);
                    applyTreeMove(path, entry.path);
                    refreshTree();
                    revealInTree(entry.path);
                    // A Cut waiting on the old path would move nothing, so it is
                    // dropped rather than left pointing at a file that is gone.
                    if (clipboardPath === path) {
                        clipboardPath = entry.path;
                    }
                    ctx.notice = '';
                    return '';
                } catch (error) {
                    ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    }

    // duplicateEntry asks for the copy's path, prefilled with the free " copy"
    // name the backend suggests. Accepting it does what Finder does; editing it
    // names the copy before it is written, and a folder in the text puts the
    // copy there. The copy is not opened: the user asked for a second file, not
    // for a second tab.
    async function duplicateEntry(path: string): Promise<void> {
        let suggestion: string;
        try {
            suggestion = await duplicatePathSuggestion(path);
        } catch (error) {
            treeError(error);
            return;
        }
        await prompts.askName({
            title: 'Duplicate',
            value: suggestion,
            selectStem: true,
            submitLabel: 'Duplicate',
            run: async (target) => {
                try {
                    const entry = await duplicatePath(path, target);
                    refreshTree();
                    revealInTree(entry.path);
                    await git.refreshGitStatus();
                    ctx.notice = `Created ${entry.name}`;
                    return '';
                } catch (error) {
                    ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    }

    // cutEntry puts the entry aside for Paste. It changes nothing on disk: the
    // move happens when the user chooses a folder to paste into.
    function cutEntry(path: string): void {
        clipboardPath = path;
        clipboardCut = true;
        ctx.notice = `Ready to move ${entryName(path)}`;
    }

    function copyEntry(path: string): void {
        clipboardPath = path;
        clipboardCut = false;
        ctx.notice = `Ready to copy ${entryName(path)}`;
    }

    // pasteInto copies or moves the entry the clipboard holds into a folder. A
    // cut is cleared after the move, so a second Paste cannot fail on a source
    // that is no longer there.
    async function pasteInto(folder: string): Promise<void> {
        const source = clipboardPath;
        if (source === '') {
            return;
        }
        busyPath = folder;
        try {
            if (clipboardCut) {
                const entry = await movePath(source, folder);
                applyTreeMove(source, entry.path);
                clipboardPath = '';
                clipboardCut = false;
                revealInTree(entry.path);
                ctx.notice = `Moved ${entry.name}`;
            } else {
                const entry = await copyPath(source, folder);
                revealInTree(entry.path);
                ctx.notice = `Copied ${entry.name}`;
            }
            refreshTree();
            await git.refreshGitStatus();
        } catch (error) {
            treeError(error);
        } finally {
            busyPath = '';
        }
    }

    // moveEntry is a drop on a folder row. It is the same move Paste does, with
    // the source coming from the drag rather than the clipboard.
    async function moveEntry(source: string, folder: string): Promise<void> {
        busyPath = folder;
        try {
            const entry = await movePath(source, folder);
            applyTreeMove(source, entry.path);
            finishTreeMove(source, entry.path, entry.name);
        } catch (error) {
            treeError(error);
        } finally {
            busyPath = '';
        }
    }

    // finishTreeMove rereads what a move invalidated: the tree, because the
    // entry left one folder and landed in another, and git, because a repository
    // may have gained or lost a path. The moved entry is revealed so it never
    // lands out of sight.
    function finishTreeMove(source: string, target: string, name: string): void {
        applyTreeMove(source, target);
        refreshTree();
        revealInTree(target);
        void git.refreshGitStatus();
        ctx.notice = `Moved ${name}`;
    }

    // moveTreeEntry is Move…: the field holds the entry's full path with the
    // folder selected, so typing replaces where it lives and the file name stays
    // visible and intact. Move and Rename share the path operation; only the
    // initial selection differs. A refusal reopens the prompt on the same text
    // so a typo is corrected in place.
    async function moveTreeEntry(path: string): Promise<void> {
        await prompts.askName({
            title: 'Move',
            value: path,
            selectFolder: true,
            submitLabel: 'Move',
            run: async (target) => {
                busyPath = path;
                try {
                    const entry = await renamePath(path, target);
                    finishTreeMove(path, entry.path, entry.name);
                    return '';
                } catch (error) {
                    ctx.notice = '';
                    return errorMessage(error);
                } finally {
                    busyPath = '';
                }
            },
        });
    }

    // trashEntry asks first, then moves the entry to the platform's Trash. Its
    // tabs close with it: the files they show no longer exist, and a tab that
    // cannot be saved is worse than no tab. A modified buffer asks about its
    // edits before the file goes, so nothing is lost by accident.
    async function trashEntry(path: string): Promise<void> {
        const affected = ctx.tabs.filter(
            (tab) =>
                (tab.path !== '' && pathWithin(tab.path, path)) ||
                (tab.kind === 'diff' && pathWithin(tab.sourcePath ?? '', path)),
        );
        const dirty = affected.filter((tab) => tab.dirty);
        const confirmed = await prompts.askConfirm(
            'Delete',
            dirty.length > 0
                ? `Delete ${entryName(path)}? It moves to the Trash, and ${dirty.length === 1 ? 'one open buffer has' : `${dirty.length} open buffers have`} unsaved changes that will be lost.`
                : `Delete ${entryName(path)}? It moves to the Trash, so you can put it back from there.`,
            'Delete',
        );
        if (!confirmed) {
            return;
        }
        busyPath = path;
        try {
            await trashPath(path);
            for (const tab of affected) {
                documents.removeTab(tab.id);
            }
            if (pathWithin(clipboardPath, path)) {
                clipboardPath = '';
                clipboardCut = false;
            }
            refreshTree();
            await git.refreshGitStatus();
            ctx.notice = `Moved ${entryName(path)} to the Trash`;
        } catch (error) {
            treeError(error);
        } finally {
            busyPath = '';
        }
    }

    // revealInTree expands the tree to path and scrolls it into view. New
    // entries use it so they never land out of sight.
    function revealInTree(path: string): void {
        revealedPath = path;
        treeRefreshToken++;
    }

    // contextMenuItems builds the row menu for the right-click the menu was
    // opened on. The builders are pure functions in app/menus.ts.
    function contextMenuItems(): ContextMenuItem[] {
        return buildContextMenuItems(contextMenu, ctx.tabs, clipboardPath, search.open);
    }

    // runContextMenu runs the item the row menu picked against the entry it was
    // opened on: a tab command for a tab, a file-tree command for a tree row.
    function runContextMenu(id: string): void {
        const target = contextMenu;
        closeContextMenu();
        if (target === null) {
            return;
        }
        if (target.tabId !== undefined) {
            void runTabContextCommand(id, target.tabId);
            return;
        }
        void runTreeCommand(id, target.path, target.isDir);
    }

    // runTabContextCommand runs a tab menu item against the right-clicked tab.
    async function runTabContextCommand(id: string, tabId: string): Promise<void> {
        const tab = ctx.findTab(tabId);
        if (tab === undefined) {
            return;
        }
        switch (id) {
            case 'reveal':
                await revealPath(tab.path || tab.sourcePath || '');
                break;
            case 'close-tab':
                await documents.closeTab(tabId);
                break;
            case 'move-tab-new-window':
                await documents.moveTabToNewWindow(tabId);
                break;
            case 'close-other-tabs':
                await closeOtherTabs(tabId);
                break;
            case 'close-right-tabs':
                await closeTabsToRight(tabId);
                break;
            case 'close-all-tabs':
                await closeAllTabs();
                break;
            case 'close-deleted-tabs':
                await closeDeletedTabs();
                break;
        }
    }

    // closeOtherTabs leaves only the right-clicked tab, which becomes active.
    // The Find in Files tab counts as another tab, so it closes too.
    async function closeOtherTabs(id: string): Promise<void> {
        if (!(await documents.closeTabsInOrder(tabsToClose(ctx.tabs, id, 'others')))) {
            return;
        }
        if (search.open) {
            search.closeSearchTab();
        }
        ctx.activeId = id;
    }

    // closeTabsToRight closes every tab after the right-clicked one, including
    // the Find in Files tab, which is always rendered after the documents.
    async function closeTabsToRight(id: string): Promise<void> {
        if (ctx.findTab(id) === undefined) {
            return;
        }
        if (!(await documents.closeTabsInOrder(tabsToClose(ctx.tabs, id, 'right')))) {
            return;
        }
        if (search.open) {
            search.closeSearchTab();
        }
    }

    // closeAllTabs closes every document tab and the Find in Files tab. Dirty
    // buffers are asked about one at a time, and a cancelled prompt stops the
    // run with the remaining tabs still open.
    async function closeAllTabs(): Promise<void> {
        if (!(await documents.closeTabsInOrder(tabsToClose(ctx.tabs, '', 'all')))) {
            return;
        }
        if (search.open) {
            search.closeSearchTab();
        }
    }

    // closeDeletedTabs closes the buffers whose files are gone. A dirty one
    // still asks, because closing it discards text that could recreate the file.
    async function closeDeletedTabs(): Promise<void> {
        await documents.closeTabsInOrder(tabsToClose(ctx.tabs, '', 'deleted'));
    }

    async function runTreeCommand(id: string, path: string, isDir: boolean): Promise<void> {
        if (id === 'reveal') {
            await revealPath(path);
            return;
        }
        // Commands that create or paste need a folder: a file row means the
        // folder it lives in.
        const folder = isDir ? path : parentPath(path);
        switch (id) {
            case 'open':
                await openTreeEntry(path, isDir);
                break;
            case 'new-file':
                await newFileInFolder(folder);
                break;
            case 'new-folder':
                await newFolderInFolder(folder);
                break;
            case 'cut':
                cutEntry(path);
                break;
            case 'copy':
                copyEntry(path);
                break;
            case 'paste':
                await pasteInto(folder);
                break;
            case 'copy-name':
                await copyTreeText('name', path);
                break;
            case 'copy-path':
                await copyTreeText('file', path);
                break;
            case 'copy-relative-path':
                await copyTreeText('relative', path);
                break;
            case 'duplicate':
                await duplicateEntry(path);
                break;
            case 'rename':
                await renameEntry(path);
                break;
            case 'move':
                await moveTreeEntry(path);
                break;
            case 'trash':
                await trashEntry(path);
                break;
        }
    }

    // openTreeEntry is the row's primary action: a file opens in the editor, a
    // folder expands in the tree. A file that is already open is focused rather
    // than reread, so edits in its buffer survive.
    async function openTreeEntry(path: string, isDir: boolean): Promise<void> {
        if (isDir) {
            revealInTree(path);
            return;
        }
        await documents.openFile(path);
    }

    // copyTreeText copies a path spelling of one tree entry. The variants match
    // the Copy Path items that act on the active buffer, so the same three
    // strings are reachable from the tree.
    async function copyTreeText(kind: CopyPathKind, path: string): Promise<void> {
        const text =
            kind === 'file'
                ? path
                : kind === 'relative'
                  ? relativePath(path, ctx.folders)
                  : entryName(path);
        try {
            await CopyText(text);
            ctx.notice = `Copied ${text}`;
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    // recordRecent puts an opened file at the top of Open Recent.
    async function recordRecent(path: string): Promise<void> {
        try {
            recentFiles = await rememberRecentFile(path);
        } catch (error) {
            console.error('Failed to remember the recent file.', error);
        }
    }

    function showRecentFiles(): void {
        if (recentFiles.length === 0) {
            ctx.notice = 'No recent files.';
            return;
        }
        quickOpen = false;
        commandPalette = false;
        languagePalette = false;
        indentationPalette = false;
        ctx.searchActive = false;
        recentOpen = true;
    }

    function closeRecentFiles(): void {
        recentOpen = false;
    }

    function rankRecentFiles(query: string): PaletteItem[] {
        return rankCandidates(query, recentItems);
    }

    async function clearRecent(): Promise<void> {
        try {
            recentFiles = await clearRecentFiles();
            ctx.notice = 'Recent files cleared.';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    // revealActiveFileInTree selects and expands to the active file's row. A
    // diff tab names the file it was made from, which is the row to show.
    $effect(() => {
        if (!sidebarAutoReveal) {
            return;
        }
        revealedPath = ctx.active?.path || ctx.active?.sourcePath || '';
    });

    // setSidebarAutoReveal records the choice so the next launch keeps it.
    function setSidebarAutoReveal(autoReveal: boolean): void {
        if (sidebarAutoReveal === autoReveal) {
            return;
        }
        sidebarAutoReveal = autoReveal;
        void saveSidebarAutoReveal(autoReveal).catch((error) =>
            console.error('Failed to save the auto-reveal setting.', error),
        );
    }

    // setSidebarVisible records the choice so the next launch keeps the layout.
    // Toggle Sidebar and the settings overlay both go through it.
    function setSidebarVisible(visible: boolean): void {
        if (sidebarVisible === visible) {
            return;
        }
        sidebarVisible = visible;
        void saveSidebarVisible(visible).catch((error) =>
            console.error('Failed to save the sidebar state.', error),
        );
    }

    function toggleSidebar(): void {
        setSidebarVisible(!sidebarVisible);
    }

    // setExpanded tracks which folders the tree has open, so their contents can
    // be watched for external changes.
    function setExpanded(path: string, open: boolean): void {
        if (open) {
            if (!expandedPaths.includes(path)) {
                expandedPaths = [...expandedPaths, path];
            }
            return;
        }
        expandedPaths = expandedPaths.filter((entry) => entry !== path);
    }

    async function closeAllFolders(): Promise<void> {
        try {
            ctx.folders = await closeAllSidebarFolders(ctx.windowName);
            activeWorkspace = null;
            expandedPaths = [];
            void git.refreshGitStatus();
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    async function refreshWorkspace(): Promise<void> {
        treeRefreshToken++;
        await documents.refreshOpenDocuments();
        await git.refreshDiffTabs();
        await git.refreshGitStatus();
        // HEAD may have moved since the buffers were opened, so the gutter
        // baselines are reread too.
        for (const tab of ctx.tabs) {
            if (tab.kind === 'text' && tab.path !== '') {
                void git.loadBaseline(tab.id, tab.path);
            }
        }
        ctx.notice = '';
    }

    function showFindInFiles(): void {
        quickOpen = false;
        commandPalette = false;
        recentOpen = false;
        languagePalette = false;
        indentationPalette = false;
        search.open = true;
        ctx.searchActive = true;
        search.focusToken++;
    }

    // revealPath shows one file in the platform's file manager.
    async function revealPath(path: string): Promise<void> {
        try {
            await RevealFile(path);
            ctx.notice = '';
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    async function revealActiveFile(): Promise<void> {
        // A diff tab has no path of its own, but it names the file it was made
        // from, which is the file worth revealing.
        const path = ctx.active?.path || ctx.active?.sourcePath || '';
        if (path === '') {
            ctx.notice = 'Open a saved file to reveal it in Finder.';
            return;
        }
        await revealPath(path);
    }

    // showContextMenu opens the right-click menu for path at the pointer. An
    // untitled buffer has nothing on disk, so it gets no menu. A tree row also
    // says whether it is a folder and passes the entries it loaded, which is how
    // the menu knows it may create a file or a folder there.
    function showContextMenu(
        event: MouseEvent,
        path: string,
        options: { isDir?: boolean } = {},
    ): void {
        event.preventDefault();
        event.stopPropagation();
        if (path === '') {
            return;
        }
        contextMenu = {
            x: event.clientX,
            y: event.clientY,
            path,
            isDir: options.isDir ?? false,
        };
    }

    // showTabContextMenu opens the tab's right-click menu. The tab id, not the
    // path, is the target because an untitled buffer has no path to identify it.
    function showTabContextMenu(event: MouseEvent, tab: OpenDocument): void {
        event.preventDefault();
        event.stopPropagation();
        contextMenu = {
            x: event.clientX,
            y: event.clientY,
            path: tab.path || tab.sourcePath || '',
            isDir: false,
            tabId: tab.id,
        };
    }

    // showEmptyAreaMenu opens the tree's background menu for one root, titled by
    // the folder itself: New File and New Folder need somewhere to land, so the
    // menu is the one a folder row would offer.
    function showEmptyAreaMenu(event: MouseEvent, folder: string): void {
        showContextMenu(event, folder, { isDir: true });
    }

    function closeContextMenu(): void {
        contextMenu = null;
    }

    async function copyActivePath(kind: CopyPathKind): Promise<void> {
        const tab = ctx.active;
        // A diff tab has no path of its own, but it names the file it was made
        // from, which is the path worth copying.
        const path = tab?.path || tab?.sourcePath || '';
        if (tab === undefined || path === '') {
            ctx.notice = 'Open a saved file to copy its path.';
            return;
        }
        const text =
            kind === 'file'
                ? path
                : kind === 'relative'
                  ? relativePath(path, ctx.folders)
                  : tab.path === ''
                    ? entryName(path)
                    : tab.name;
        try {
            await CopyText(text);
            ctx.notice = `Copied ${text}`;
        } catch (error) {
            ctx.notice = errorMessage(error);
        }
    }

    // restoreSession reopens the files the previous run left open. Missing
    // files are skipped, so one deleted file does not stop the rest.
    async function restoreSession(session: SavedWindowSession): Promise<void> {
        for (const path of session.openFiles) {
            if (findDocumentByPath(ctx.tabs, path) !== undefined) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            documents.showDocument(opened, false);
        }
        const restoredActive = findDocumentByPath(ctx.tabs, session.active);
        if (restoredActive !== undefined) {
            ctx.activeId = restoredActive.id;
        }
    }

    // persistSession writes the current window state immediately. The live
    // effect below covers ordinary tab changes; workspace switching and closing
    // call this so the old set is on disk before the tab list is replaced.
    async function persistSession(): Promise<void> {
        if (ctx.windowName === '') {
            return;
        }
        try {
            await saveSession(ctx.windowName, {
                workspace: activeWorkspace?.path ?? '',
                folders: [...ctx.folders],
                openFiles: [...sessionFiles],
                active: sessionActive,
                expanded: [...expandedPaths],
            });
        } catch (error) {
            console.error('Failed to save the session.', error);
        }
    }

    // A preview belongs to the tab it was opened on; selecting another tab puts
    // the editor back rather than showing a stale snapshot.
    $effect(() => {
        if (preview !== null && preview.id !== ctx.activeId) {
            preview = null;
        }
    });

    // The watcher only needs paths that exist: open files and expanded folders.
    // It is registered per window, so a change reaches the window that asked
    // for it and closing the window stops its events.
    $effect(() => {
        if (ctx.windowName === '') {
            return;
        }
        const paths = watchedPaths;
        void watchPaths(ctx.windowName, paths).catch((error) =>
            console.error('Failed to watch the workspace.', error),
        );
    });

    // The session is written whenever the file-backed tabs, active file,
    // workspace binding or expanded set changes. Gating on sessionReady keeps
    // startup from overwriting the saved session with the empty list the app
    // begins with. A non-owner window parks its workspace set but never
    // replaces the launch window's tabs.
    $effect(() => {
        if (!sessionReady || sessionSuspended || ctx.windowName === '') {
            return;
        }
        const files = sessionFiles;
        const activePath = sessionActive;
        const workspacePath = activeWorkspace?.path ?? '';
        const currentFolders = ctx.folders;
        const expanded = expandedPaths;
        if (!sessionOwner && workspacePath === '') {
            return;
        }
        void saveSession(ctx.windowName, {
            workspace: workspacePath,
            folders: currentFolders,
            openFiles: files,
            active: activePath,
            expanded,
        }).catch((error) =>
            console.error('Failed to save the session.', error),
        );
    });

    // Go decides whether closing needs a prompt, so it is told whenever the
    // answer changes. Each window reports its own buffers; a failed report only
    // costs a redundant prompt.
    $effect(() => {
        if (ctx.windowName === '') {
            return;
        }
        void SetUnsavedChanges(ctx.windowName, ctx.unsaved).catch((error) =>
            console.error('Failed to report unsaved changes.', error),
        );
    });

    onMount(() => {
        let unsubscribe: (() => void) | undefined;
        let systemTheme: MediaQueryList | undefined;
        let disposed = false;

        // System mode follows the OS appearance without a restart.
        const onSystemTheme = (): void => {
            if (appearance.themeSettings.mode === 'system') {
                void appearance.applyCurrentTheme();
            }
        };

        void (async () => {
            // The window name is the identity every per-window binding and
            // event uses, so it is resolved before anything else happens.
            const name = await resolveWindowName();
            if (disposed) {
                return;
            }
            ctx.windowName = name;
            setWindowName(name);
            appName = await GetProductName();

            // The request this window was created with decides two things: the
            // paths or buffers to open, and whether this is the window that
            // restores and writes the shared session. Only the session flag is
            // read now; the request itself is collected once its listener is
            // installed, so nothing queued meanwhile can be lost.
            sessionOwner = await ShouldRestoreSession(ctx.windowName);

            const workspace = await loadWorkspaceState();
            sidebarVisible = workspace.sidebarVisible;
            sidebarAutoReveal = workspace.autoReveal;
            recentFiles = workspace.recentFiles;
            ctx.folders = await loadSidebarFolders(ctx.windowName);
            // Git status may walk a large repository; the session restore does
            // not wait for it, and the branch appears when the read lands.
            void git.refreshGitStatus();
            if (sessionOwner) {
                const saved = await loadWindowSession(ctx.windowName);
                if (saved.folders.length > 0) {
                    try {
                        ctx.folders = await setSidebarFolders(ctx.windowName, saved.folders);
                    } catch (error) {
                        console.error('Failed to restore the sidebar folders.', error);
                    }
                }
                if (saved.workspace !== '') {
                    try {
                        const info = await loadWorkspace(saved.workspace);
                        activeWorkspace = info;
                        ctx.folders = await setSidebarFolders(ctx.windowName, info.folders ?? []);
                        void git.refreshGitStatus();
                    } catch (error) {
                        // The workspace file may have moved or been deleted;
                        // the saved tabs are still worth reopening.
                        console.error('Failed to restore the workspace.', error);
                    }
                }
                expandedPaths = saved.expanded;
                await restoreSession(saved);
            }
            // From here on tab changes are saved as the session; before this
            // point an empty list is just startup, not the user closing tabs.
            sessionReady = true;

            const loaded = await loadEditorSettings();
            if (disposed) {
                return;
            }
            appearance.editorSettings = loaded;

            const preferences = await loadPreferences();
            if (disposed) {
                return;
            }
            appearance.themeSettings = preferences.theme;
            appearance.uiSettings = preferences.ui;
            appearance.filesExclude = preferences.files.exclude;
            appearance.filesTrimTrailingWhitespace = preferences.files.trimTrailingWhitespace;
            appearance.searchExclude = preferences.searchExclude;
            applyUISettings(appearance.uiSettings);
            await appearance.refreshThemes();
            await appearance.applyCurrentTheme();
            void SetAppearanceMode(appearance.themeSettings.mode).catch((error) =>
                console.error('Failed to update the Appearance menu.', error),
            );

            const off = await subscribeAll(ctx.windowName, {
                launch: (request) => void applyLaunch(request),
                addFolder: () => void addFolderFromDialog(),
                closeTab: () => {
                    if (ctx.activeId !== '') {
                        void documents.closeTab(ctx.activeId);
                    }
                },
                quickOpen: () => void showQuickOpen(),
                commandPalette: showCommandPalette,
                menuCommand: selectCommand,
                fileCommands: {
                    'new-file': () => void newFileFromMenu(),
                    'new-folder': () => void newFolderFromMenu(),
                    'new-buffer': documents.newFile,
                    'open-file': () => void documents.openFromDialog(),
                    'save': () => saveActive(),
                    'save-as': () => saveActive(true),
                    'save-all': () => void documents.saveAll(),
                    'print': () => void printActive(),
                },
                closeRequest: (reason) => void documents.confirmClose(reason),
                editorCommands: {
                    transformSelection,
                    trimWhitespace,
                    selectionCommand: runSelectionCommand,
                    zoom,
                    togglePreview,
                    setViewOption: appearance.setViewOption,
                    find: () => runEditorAction('actions.find'),
                    replace: () => runEditorAction('editor.action.startFindReplaceAction'),
                    goToLine: () => runEditorAction('editor.action.gotoLine'),
                    nextFindResult: () => void search.stepFindResult(1),
                    previousFindResult: () => void search.stepFindResult(-1),
                },
                watchedChanges: (paths) => void documents.handleWatchedChanges(paths),
                workspaceCommands: {
                    openRecent: showRecentFiles,
                    openWorkspace: () => void openWorkspaceFromDialog(),
                    saveWorkspaceAs: () => void saveWorkspaceAsDialog(),
                    closeWorkspace: () => void closeWorkspace(),
                    refresh: () => void refreshWorkspace(),
                    findInFiles: showFindInFiles,
                    toggleSidebar,
                    closeAllFolders: () => void closeAllFolders(),
                    revealFile: () => void revealActiveFile(),
                    copyPath: (kind) => void copyActivePath(kind),
                },
                tabCommands: {
                    next: () => documents.stepTab(1),
                    previous: () => documents.stepTab(-1),
                    reopen: () => void documents.reopenClosedTab(),
                    goToTab: documents.goToTab,
                    moveToNewWindow: () => void documents.moveTabToNewWindow(),
                },
                configChange: (change) => {
                    if (change.themes) {
                        void (async () => {
                            await appearance.refreshThemes();
                            await appearance.applyCurrentTheme();
                        })();
                    }
                    if (change.keybindings) {
                        void appearance.refreshKeybindings();
                    }
                    if (change.settings) {
                        void appearance.reloadSettings();
                    }
                },
                appearance: (mode) => void appearance.changeThemeSettings({ ...appearance.themeSettings, mode }),
                settingsOverlay: showSettingsOverlay,
            });
            if (disposed) {
                off();
                return;
            }
            unsubscribe = off;
            installEditorKeybindings(appearance.runKeybindingCommand);
            await appearance.refreshKeybindings();
            void WatchConfig();

            systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
            systemTheme.addEventListener('change', onSystemTheme);
        })();

        return () => {
            disposed = true;
            unsubscribe?.();
            systemTheme?.removeEventListener('change', onSystemTheme);
            appearance.disposeKeybindings?.();
        };
    });
</script>

<div class="app" data-file-drop-target>
    <TitleBar {appName} filePath={shownPath(ctx.active)} folderLabel={folderLabel} />
    <div class="app__body" class:app__body--sidebar-right={appearance.uiSettings.sidebarPosition === 'right'}>
        <div class="app__sidebar" class:app__sidebar--hidden={!sidebarVisible}>
            {#if sidebarView === 'changes'}
                <ChangesView
                    changes={git.changes}
                    busy={ctx.busy}
                    onOpenDiff={(path) => void git.openDiff(path)}
                    onRefresh={() => void refreshWorkspace()}
                />
            {:else}
                <Sidebar
                    folders={ctx.folders}
                    selectedPath={revealedPath}
                    busy={ctx.busy}
                    refreshToken={treeRefreshToken}
                    {busyPath}
                    statuses={git.statuses}
                    iconStyle={appearance.uiSettings.iconStyle}
                    onAddFolder={() => void addFolderFromDialog()}
                    onRemoveFolder={(path) => void removeFolder(path)}
                    onOpenFile={(path) => void documents.openFile(path)}
                    onContextMenu={showContextMenu}
                    onEmptyAreaMenu={showEmptyAreaMenu}
                    onExpandedChange={setExpanded}
                    onMovePath={(source, folder) => void moveEntry(source, folder)}
                />
            {/if}
        </div>
        <div class="app__editor">
            <TabBar
                tabs={ctx.tabs}
                activeId={ctx.activeId}
                showIcons={appearance.uiSettings.tabIcons}
                iconStyle={appearance.uiSettings.iconStyle}
                searchOpen={search.open}
                searchActive={ctx.searchActive}
                onSelect={(id) => {
                    ctx.activeId = id;
                    ctx.searchActive = false;
                }}
                onClose={(id) => void documents.closeTab(id)}
                onContextMenu={(event, tab) => showTabContextMenu(event, tab)}
                onReorder={documents.reorderTab}
                onSelectSearch={() => {
                    search.open = true;
                    ctx.searchActive = true;
                    search.focusToken++;
                }}
                onCloseSearch={search.closeSearchTab}
            />
            <div class="app__pane">
                <div
                    class="app__pane-layer"
                    class:app__pane-layer--hidden={paneReplaced}
                    inert={paneReplaced}
                >
                    <EditorPane
                        bind:this={editorPane}
                        modelKey={editorKey}
                        content={editorContent}
                        savedContent={editorSavedContent}
                        eol={editorEol}
                        {openIds}
                        settings={appearance.editorSettings}
                        readOnly={ctx.active?.kind === 'diff'}
                        onRevealFile={() => void revealActiveFile()}
                        onCopyPath={(kind) => void copyActivePath(kind)}
                        onDirtyChange={(id, dirty) => {
                            ctx.tabs = setDocumentDirty(ctx.tabs, id, dirty);
                        }}
                        onIndentationChange={(insertSpaces, tabSize, indentSize) => {
                            indentation = { insertSpaces, tabSize, indentSize };
                        }}
                        onCursorMove={(nextLine, nextColumn) => {
                            line = nextLine;
                            column = nextColumn;
                        }}
                        onSelectionChange={(length) => {
                            selectionLength = length;
                        }}
                    />
                </div>
                {#if ctx.searchActive}
                    <SearchPanel
                        bind:query={search.query}
                        bind:include={search.include}
                        bind:exclude={search.excludeGlobs}
                        bind:replacement={search.replacement}
                        bind:regex={search.regex}
                        bind:caseSensitive={search.caseSensitive}
                        groups={search.groups}
                        summary={search.label}
                        error={search.error}
                        busy={search.busy}
                        focusToken={search.focusToken}
                        roots={ctx.folders}
                        onSearch={() => void search.runSearch()}
                        onReplaceAll={() => void search.replaceInFiles()}
                        onOpenMatch={(path, line, column) =>
                            void search.openSearchMatch(path, line, column)}
                    />
                {:else if ctx.active?.kind === 'image'}
                    {#key ctx.active.src}
                        <ImageViewer
                            src={ctx.active.src}
                            name={ctx.active.name}
                            onSize={(width, height) => {
                                imageSize = { id: ctx.activeId, width, height };
                            }}
                        />
                    {/key}
                {:else if previewActive && preview !== null}
                    <Preview
                        kind={preview.kind}
                        content={preview.content}
                        name={ctx.active?.name ?? ''}
                    />
                {:else if welcomeActive}
                    <Welcome onCommand={selectCommand} />
                {/if}
            </div>
        </div>
    </div>
    <StatusBar
        {line}
        {column}
        {selectionLength}
        {language}
        eol={ctx.active?.kind === 'text' ? ctx.active.eol : ''}
        indentation={indentationText}
        invalidUTF8={ctx.active?.kind === 'text' && ctx.active.invalidUTF8}
        notice={ctx.notice}
        {folderLabel}
        imageSize={imageSizeLabel}
        branch={git.branch}
        sidebarView={sidebarView}
        changesCount={git.changes.length}
        onSwitchView={switchSidebarView}
        onToggleEol={toggleLineEnding}
        onPickIndentation={ctx.active?.kind === 'text' ? showIndentationPalette : undefined}
        onChangeLanguage={ctx.active?.kind === 'text' ? showLanguagePalette : undefined}
    />
    {#if quickOpen}
        <Palette
            title="Quick Open"
            placeholder="Search files by name or path"
            rank={rankFiles}
            empty={quickOpenEmpty}
            note={quickOpenTruncated ? 'The file list hit its size limit; not every file is shown.' : ''}
            onSelect={(path) => void openFromQuickOpen(path)}
            onClose={closeQuickOpen}
        />
    {/if}
    {#if recentOpen}
        <Palette
            title="Open Recent"
            placeholder="Search recent files"
            rank={rankRecentFiles}
            empty="No recent files."
            onSelect={(path) => {
                recentOpen = false;
                void documents.openFile(path);
            }}
            onClose={closeRecentFiles}
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
    {#if languagePalette}
        <Palette
            title="Change Language Mode"
            placeholder="Search languages"
            rank={rankLanguages}
            empty="No matching languages."
            onSelect={chooseLanguage}
            onClose={() => (languagePalette = false)}
        />
    {/if}
    {#if indentationPalette}
        <Palette
            title="Select Indentation"
            placeholder="Search indentation actions"
            rank={rankIndentation}
            empty="No matching actions."
            onSelect={chooseIndentation}
            onClose={() => (indentationPalette = false)}
        />
    {/if}
    {#if prompts.unsaved !== null}
        <Prompt
            title={prompts.unsaved.title}
            message={prompts.unsaved.message}
            details={prompts.unsaved.details}
            actions={[
                { id: 'save', label: prompts.unsaved.confirmLabel, primary: true },
                { id: 'discard', label: "Don't Save" },
                { id: 'cancel', label: 'Cancel' },
            ]}
            onSelect={prompts.answerUnsaved}
            onCancel={() => prompts.answerUnsaved('cancel')}
        />
    {/if}
    {#if prompts.confirm !== null}
        <Prompt
            title={prompts.confirm.title}
            message={prompts.confirm.message}
            actions={prompts.confirm.actions}
            onSelect={prompts.answerConfirm}
            onCancel={() => prompts.answerConfirm('cancel')}
        />
    {/if}
    {#if contextMenu !== null}
        <ContextMenu
            x={contextMenu.x}
            y={contextMenu.y}
            items={contextMenuItems()}
            onSelect={runContextMenu}
            onClose={closeContextMenu}
        />
    {/if}
    {#if prompts.name !== null}
        <NamePrompt
            title={prompts.name.title}
            label={prompts.name.label}
            bind:value={prompts.name.value}
            selectStem={prompts.name.selectStem}
            selectFolder={prompts.name.selectFolder}
            error={prompts.name.error}
            onSubmit={prompts.submitName}
            onCancel={prompts.cancelName}
        />
    {/if}
    {#if settingsOverlay}
        <SettingsOverlay
            editor={appearance.editorSettings}
            theme={appearance.themeSettings}
            ui={appearance.uiSettings}
            themes={appearance.themes}
            filesExclude={appearance.filesExclude}
            trimTrailingWhitespace={appearance.filesTrimTrailingWhitespace}
            searchExclude={appearance.searchExclude}
            {sidebarVisible}
            autoReveal={sidebarAutoReveal}
            warning={appearance.settingsWarning}
            onEditor={appearance.changeEditorSettings}
            onTheme={(next) => void appearance.changeThemeSettings(next)}
            onUI={(next) => void appearance.changeUISettings(next)}
            onFilesExclude={(next) => void appearance.changeFilesExclude(next)}
            onTrimTrailingWhitespace={(next) => void appearance.changeFilesTrimTrailingWhitespace(next)}
            onSearchExclude={(next) => void appearance.changeSearchExclude(next)}
            onSidebarVisible={setSidebarVisible}
            onAutoReveal={setSidebarAutoReveal}
            onClose={closeSettingsOverlay}
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

    .app__body--sidebar-right {
        flex-direction: row-reverse;
    }

    .app__body--sidebar-right .app__sidebar :global(.sidebar),
    .app__body--sidebar-right .app__sidebar :global(.changes) {
        border-right: none;
        border-left: 1px solid var(--border);
    }

    .app__sidebar {
        display: flex;
        flex: 0 0 auto;
        min-height: 0;
    }

    .app__sidebar--hidden {
        display: none;
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
