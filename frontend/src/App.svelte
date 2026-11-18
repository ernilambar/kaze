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
        SetAppearanceMode,
        SetUnsavedChanges,
        SettingsFile,
        ShouldRestoreSession,
        ShowAbout,
        ToggleFullScreen,
        WatchConfig,
        ZoomWindow,
    } from '../bindings/kaze/app';
    import { Appearance } from './app/appearance.svelte';
    import { WindowContext } from './app/context.svelte';
    import { dispatchCommand, type CommandHandlers } from './app/dispatch';
    import { Documents } from './app/documents.svelte';
    import { subscribeAll } from './app/events';
    import { GitController } from './app/git.svelte';
    import { contextMenuItems as buildContextMenuItems } from './app/menus';
    import { Prompts } from './app/prompts.svelte';
    import { SearchController } from './app/search.svelte';
    import { TreeOps } from './app/treeops.svelte';
    import { WorkspaceController } from './app/workspace.svelte';
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
    import {
        shownPath,
        type DocumentEOL,
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
        entryName,
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
        errorMessage,
        loadSidebarFolders,
        relativePath,
        setSidebarFolders,
        sidebarLabel,
    } from './sidebar';
    import { indentationLabel } from './status';
    import {
        setDocumentDirty,
        tabsToClose,
        updateDocument,
    } from './tabs';
    import type { LineTransform } from './text';
    import {
        loadWindowSession,
        loadWorkspace,
        loadWorkspaceState,
        saveSession,
        saveSidebarAutoReveal,
        saveSidebarVisible,
        watchPaths,
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
    let sidebarVisible = $state(true);
    let sidebarAutoReveal = $state(true);

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
    // Controllers built later are reached through closures, which only run
    // after init: git and search need documents, and documents, search and
    // appearance need the tree and the workspace.
    const git = new GitController(ctx, {
        showDocument: (document, record) => documents.showDocument(document, record),
    });
    const search = new SearchController(ctx, prompts, git, {
        openFile: (path): Promise<string> => documents.openFile(path),
        refreshTree: () => treeops.refreshTree(),
    });
    const appearance = new Appearance(ctx, {
        refreshTree: () => treeops.refreshTree(),
        sidebarVisible: () => sidebarVisible,
        selectCommand,
    });
    const documents = new Documents(ctx, prompts, git, appearance, {
        appName: () => appName,
        recordRecent: (path): Promise<void> => workspaces.recordRecent(path),
        refreshTree: () => treeops.refreshTree(),
    });
    const treeops = new TreeOps(ctx, prompts, git, documents);
    const workspaces = new WorkspaceController(ctx, prompts, git, search, documents, treeops);

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
        workspaces.activeWorkspace !== null && workspaces.activeWorkspace.name !== ''
            ? workspaces.activeWorkspace.name
            : sidebarLabel(ctx.folders),
    );
    let imageSizeLabel = $derived(
        imageSize !== null && imageSize.id === ctx.activeId
            ? `${imageSize.width} × ${imageSize.height}`
            : '',
    );
    let recentItems = $derived(toRecentPaletteItems(workspaces.recentFiles, ctx.folders));
    let watchedPaths = $derived(Array.from(new Set([...workspaces.sessionFiles, ...treeops.expandedPaths])));
    let quickOpenEmpty = $derived(
        quickOpenLoading
            ? 'Loading files…'
            : quickOpenFiles.length === 0
              ? 'Add a folder to the sidebar to search its files.'
              : 'No matching files.',
    );

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
        newFile: () => void treeops.newFileFromMenu(),
        newFolder: () => void treeops.newFolderFromMenu(),
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
        clearRecent: () => void workspaces.clearRecent(),
        showFindInFiles,
        showSettings: showSettingsOverlay,
        showAbout: () => void ShowAbout(),
        stepFindResult: (direction) => void search.stepFindResult(direction),
        togglePreview,
        switchSidebarView,
        zoom,
        addFolder: () => void workspaces.addFolderFromDialog(),
        closeActiveTab: () => {
            if (ctx.activeId !== '') {
                void documents.closeTab(ctx.activeId);
            }
        },
        goToTab: documents.goToTab,
        stepTab: documents.stepTab,
        reopenClosedTab: () => void documents.reopenClosedTab(),
        closeAllTabs: () => void closeAllTabs(),
        closeAllFolders: () => void workspaces.closeAllFolders(),
        moveTabToNewWindow: () => void documents.moveTabToNewWindow(),
        openSettingsJson: () => void openSettings(),
        openDefaultSettings: () => void openDefaultSettings(),
        openThemesFolder: () => void openThemesFolder(),
        openKeybindings: () => void openKeybindings(),
        installCommandLineTool: () => void installCommandLineTool(),
        setAppearanceMode: (mode) => void appearance.changeThemeSettings({ ...appearance.themeSettings, mode }),
        openWorkspace: () => void workspaces.openWorkspaceFromDialog(),
        saveWorkspaceAs: () => void workspaces.saveWorkspaceAsDialog(),
        closeWorkspace: () => void workspaces.closeWorkspace(),
        refreshWorkspace: () => void workspaces.refreshWorkspace(),
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

    // contextMenuItems is the row menu for the right-click the menu was opened
    // on. The builders are pure functions in app/menus.ts. It is empty when the
    // right-clicked tab closed while the menu was up, and the menu then closes
    // instead of drawing an empty box.
    let contextMenuItems = $derived(
        buildContextMenuItems(contextMenu, ctx.tabs, treeops.clipboardPath, search.open, ctx.folders),
    );

    // A menu with nothing to offer is closed rather than kept hidden, so a tab
    // reopened under the same id cannot bring the old menu back.
    $effect(() => {
        if (contextMenu !== null && contextMenuItems.length === 0) {
            contextMenu = null;
        }
    });

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
        void treeops.runTreeCommand(id, target.path, target.isDir);
    }

    // runTabContextCommand runs a tab menu item against the right-clicked tab.
    async function runTabContextCommand(id: string, tabId: string): Promise<void> {
        const tab = ctx.findTab(tabId);
        if (tab === undefined) {
            return;
        }
        switch (id) {
            case 'reveal':
                await treeops.revealPath(tab.path || tab.sourcePath || '');
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

    function showRecentFiles(): void {
        if (workspaces.recentFiles.length === 0) {
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

    // revealActiveFileInTree selects and expands to the active file's row. A
    // diff tab names the file it was made from, which is the row to show.
    $effect(() => {
        if (!sidebarAutoReveal) {
            return;
        }
        treeops.revealedPath = ctx.active?.path || ctx.active?.sourcePath || '';
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

    async function revealActiveFile(): Promise<void> {
        // A diff tab has no path of its own, but it names the file it was made
        // from, which is the file worth revealing.
        const path = ctx.active?.path || ctx.active?.sourcePath || '';
        if (path === '') {
            ctx.notice = 'Open a saved file to reveal it in Finder.';
            return;
        }
        await treeops.revealPath(path);
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
        if (!workspaces.sessionReady || workspaces.sessionSuspended || ctx.windowName === '') {
            return;
        }
        const files = workspaces.sessionFiles;
        const activePath = workspaces.sessionActive;
        const workspacePath = workspaces.activeWorkspace?.path ?? '';
        const currentFolders = ctx.folders;
        const expanded = treeops.expandedPaths;
        if (!workspaces.sessionOwner && workspacePath === '') {
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
            workspaces.sessionOwner = await ShouldRestoreSession(ctx.windowName);

            const workspace = await loadWorkspaceState();
            sidebarVisible = workspace.sidebarVisible;
            sidebarAutoReveal = workspace.autoReveal;
            workspaces.recentFiles = workspace.recentFiles;
            ctx.folders = await loadSidebarFolders(ctx.windowName);
            // Git status may walk a large repository; the session restore does
            // not wait for it, and the branch appears when the read lands.
            void git.refreshGitStatus();
            if (workspaces.sessionOwner) {
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
                        workspaces.activeWorkspace = info;
                        ctx.folders = await setSidebarFolders(ctx.windowName, info.folders ?? []);
                        void git.refreshGitStatus();
                    } catch (error) {
                        // The workspace file may have moved or been deleted;
                        // the saved tabs are still worth reopening.
                        console.error('Failed to restore the workspace.', error);
                    }
                }
                treeops.expandedPaths = saved.expanded;
                await workspaces.restoreSession(saved);
            }
            // From here on tab changes are saved as the session; before this
            // point an empty list is just startup, not the user closing tabs.
            workspaces.sessionReady = true;

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
                launch: (request) => void workspaces.applyLaunch(request),
                addFolder: () => void workspaces.addFolderFromDialog(),
                closeTab: () => {
                    if (ctx.activeId !== '') {
                        void documents.closeTab(ctx.activeId);
                    }
                },
                quickOpen: () => void showQuickOpen(),
                commandPalette: showCommandPalette,
                menuCommand: selectCommand,
                fileCommands: {
                    'new-file': () => void treeops.newFileFromMenu(),
                    'new-folder': () => void treeops.newFolderFromMenu(),
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
                    openWorkspace: () => void workspaces.openWorkspaceFromDialog(),
                    saveWorkspaceAs: () => void workspaces.saveWorkspaceAsDialog(),
                    closeWorkspace: () => void workspaces.closeWorkspace(),
                    refresh: () => void workspaces.refreshWorkspace(),
                    findInFiles: showFindInFiles,
                    toggleSidebar,
                    closeAllFolders: () => void workspaces.closeAllFolders(),
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
                    onRefresh={() => void workspaces.refreshWorkspace()}
                />
            {:else}
                <Sidebar
                    folders={ctx.folders}
                    selectedPath={treeops.revealedPath}
                    busy={ctx.busy}
                    refreshToken={treeops.treeRefreshToken}
                    busyPath={treeops.busyPath}
                    statuses={git.statuses}
                    iconStyle={appearance.uiSettings.iconStyle}
                    onAddFolder={() => void workspaces.addFolderFromDialog()}
                    onRemoveFolder={(path) => void workspaces.removeFolder(path)}
                    onOpenFile={(path) => void documents.openFile(path)}
                    onContextMenu={showContextMenu}
                    onEmptyAreaMenu={showEmptyAreaMenu}
                    onExpandedChange={treeops.setExpanded}
                    onMovePath={(source, folder) => void treeops.moveEntry(source, folder)}
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
    {#if contextMenu !== null && contextMenuItems.length > 0}
        <ContextMenu
            x={contextMenu.x}
            y={contextMenu.y}
            items={contextMenuItems}
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
