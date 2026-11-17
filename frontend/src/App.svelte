<script lang="ts">
    import * as monaco from 'monaco-editor';
    import { onMount, tick } from 'svelte';
    import {
        BringAllToFront,
        CloseWindow,
        CopyText,
        FileExists,
        GetKeybindings,
        GetProductName,
        InstallCommandLineTool,
        MinimiseWindow,
        MoveTabToNewWindow,
        NewWindow,
        OpenDefaultSettings,
        OpenFileDialog,
        OpenFolderInNewWindow,
        OpenKeybindings,
        OpenSettings,
        OpenThemesFolder,
        PrintDocument,
        ReplaceInProject,
        ResolveClose,
        RevealFile,
        SaveDocument,
        SaveFileDialog,
        SearchProject,
        SetAppearanceMode,
        SetUnsavedChanges,
        SetViewOption,
        ShouldRestoreSession,
        ShowAbout,
        ToggleFullScreen,
        WatchConfig,
        ZoomWindow,
    } from '../bindings/kaze/app';
    import type { SaveResult, SearchMatch, WorkspaceInfo } from '../bindings/kaze/models';
    import { CASE_TRANSFORMS, type CaseTransform } from './case';
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
    import { COMMANDS, INDENT_PALETTE_COMMANDS, listenForCommandPalette, listenForMenuCommand, rankCommands } from './commands';
    import { treeMenuItems, type ContextMenuItem } from './contextmenu';
    import {
        createDiffDocument,
        createUntitledDocument,
        diffTabId,
        listenForLaunch,
        normalizeEol,
        readDocument,
        shownPath,
        toBufferDocument,
        type DocumentEOL,
        type LaunchBuffer,
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
        listenForEditorCommands,
        INDENT_COMMANDS,
        SELECTION_COMMANDS,
        VIEW_OPTIONS,
        type SelectionCommand,
        type ViewOption,
        type ZoomCommand,
    } from './editorcommands';
    import { listenForCloseRequest, listenForFileCommands, type CloseReason } from './filecommands';
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
        branchLabel,
        loadGitStatus,
        readGitBaseline,
        readGitDiff,
        repoForPath,
        type GitChange,
        type GitRepo,
        type GitStatusKind,
        type SidebarView,
    } from './git';
    import type { PaletteItem } from './palette';
    import { previewKindFor, type PreviewKind } from './preview';
    import {
        listFiles,
        listenForQuickOpen,
        rankCandidates,
        rankQuickOpen,
        toQuickOpenFile,
        toQuickOpenFiles,
        type QuickOpenFile,
    } from './quickOpen';
    import {
        expectedModified,
        groupSearchMatches,
        replaceConfirmMessage,
        replaceNotice,
        replaceText,
        searchExpression,
        searchSummary,
        stepSearchResult,
    } from './search';
    import {
        defaultEditorSettings,
        defaultThemeSettings,
        defaultUISettings,
        loadEditorSettings,
        loadPreferences,
        saveEditorSettings,
        saveFilesExclude,
        saveSearchExclude,
        saveThemeSettings,
        saveUISettings,
        type EditorSettings,
        type ThemeSettings,
        type UISettings,
    } from './settings';
    import {
        installKeybindings,
        isUsableBinding,
        withDefaultKeybindings,
        type KeyContext,
    } from './keybindings';
    import {
        applyThemeCSS,
        applyUISettings,
        cacheStartupTheme,
        listThemes,
        listenForAppearance,
        listenForConfigChanges,
        listenForSettingsOverlay,
        loadTheme,
        monacoThemeName,
        resolveThemeName,
        systemPrefersDark,
        themeData,
        type ThemeInfo,
        type ThemePalette,
    } from './theme';
    import {
        addSidebarFolder,
        closeAllSidebarFolders,
        errorMessage,
        listenForAddFolder,
        loadSidebarFolders,
        openFolderDialog,
        relativePath,
        removeSidebarFolder,
        setSidebarFolders,
        sidebarLabel,
    } from './sidebar';
    import { indentationLabel, noticeFor } from './status';
    import {
        closeDocumentTab,
        findDocumentByPath,
        listenForCloseTab,
        listenForTabCommands,
        moveDocumentTab,
        openDocumentTab,
        setDocumentDirty,
        pathWithin,
        retargetDocumentPaths,
        stepTabId,
        tabIdAt,
        tabsToClose,
        updateDocument,
    } from './tabs';
    import { appendFinalNewline } from './text';
    import {
        clearRecentFiles,
        listenForWatchedChanges,
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
    import { listenForWorkspaceCommands, type CopyPathKind } from './workspacecommands';
    import { resolveWindowName, setWindowName } from './windowstate';

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

    // ConfirmPrompt asks whether a save may go ahead when it would lose
    // something: another program's newer file, or bytes that were never valid
    // UTF-8 and are now replacement characters. A conflict offers Reload as a
    // third answer, so the prompt carries its actions instead of a boolean.
    interface ConfirmPrompt {
        title: string;
        message: string;
        actions: { id: string; label: string; primary?: boolean }[];
        answer: (id: string) => void;
    }

    // ClosedTab is one entry in the Reopen Closed Tab stack. The buffer text is
    // kept because an untitled tab has no file to read again.
    interface ClosedTab {
        document: OpenDocument;
        content: string;
    }

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

    let tabs = $state<OpenDocument[]>([]);
    let activeId = $state('');
    let folders = $state<string[]>([]);
    // activeWorkspace is the file the roots came from. It is session state: the
    // roots themselves are what settings.json restores.
    let activeWorkspace = $state<WorkspaceInfo | null>(null);
    let notice = $state('');
    let busy = $state(false);
    // windowName is this window's Wails identity. Every per-window backend call
    // and event listener uses it, so a menu command or close prompt lands in
    // the window it belongs to.
    let windowName = $state('');

    // Git is read-only. The sidebar shows the status marks from gitChanges,
    // while the status bar shows the active file's branch.
    let sidebarView = $state<SidebarView>('explorer');
    let gitRepos = $state<GitRepo[]>([]);
    let gitChanges = $state<GitChange[]>([]);

    let editorSettings = $state<EditorSettings>({ ...defaultEditorSettings });
    // Editor saves are chained so the last change wins; the version identifies
    // the newest request, whose normalized result is the one applied.
    let editorSaveQueue: Promise<void> = Promise.resolve();
    let editorSaveVersion = 0;
    let themeSettings = $state<ThemeSettings>({ ...defaultThemeSettings });
    let uiSettings = $state<UISettings>({ ...defaultUISettings });
    // Excludes are the two glob lists settings.json keeps. The tree and Find in
    // Files both read them in Go; the settings overlay edits them.
    let filesExclude = $state<string[]>([]);
    let searchExclude = $state<string[]>([]);
    let themes = $state<ThemeInfo[]>([]);
    let themeErrors = $state<string[]>([]);
    let settingsOverlay = $state(false);
    let keybindingError = $state('');
    let disposeKeybindings: (() => void) | undefined;
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
    // closedTabs is the Reopen Closed Tab stack. It is never rendered, so it
    // stays a plain array instead of reactive state.
    let closedTabs: ClosedTab[] = [];

    let imageSize = $state<{ id: string; width: number; height: number } | null>(null);

    // Find in Files state. The results tab is open while results exist, even
    // when another tab is active.
    let searchOpen = $state(false);
    let searchActive = $state(false);
    let searchQuery = $state('');
    let searchInclude = $state('');
    let searchExcludeGlobs = $state('');
    let searchReplacement = $state('');
    let searchRegex = $state(false);
    let searchCaseSensitive = $state(false);
    let searchMatches = $state<SearchMatch[]>([]);
    let searchTruncated = $state(false);
    // searchResultIndex is the position F4 last opened in searchMatches, or -1
    // before the first step. A new search resets it.
    let searchResultIndex = $state(-1);
    // searchedParams records the query that produced searchMatches. Replace
    // All uses these rather than the live fields, so changing the query without
    // searching again cannot replace text the results do not describe.
    let searchedParams = $state({
        query: '',
        regex: false,
        caseSensitive: false,
        include: '',
        exclude: '',
    });
    let searchBusy = $state(false);
    let searchError = $state('');
    let searchFocusToken = $state(0);

    let unsavedPrompt = $state<UnsavedPrompt | null>(null);
    let confirmPrompt = $state<ConfirmPrompt | null>(null);
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

    // NamePromptState is the one-line dialog that asks for a name: a new file, a
    // new folder, a rename or a duplicate. answer resolves what the operation
    // awaiting it needs, so each command reads top to bottom.
    interface NamePromptState {
        title: string;
        label?: string;
        value: string;
        selectStem: boolean;
        selectFolder: boolean;
        submitLabel: string;
        error: string;
        answer: (value: string) => void;
        cancel: () => void;
    }

    let namePrompt = $state<NamePromptState | null>(null);

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

    let active = $derived(tabs.find((tab) => tab.id === activeId));
    let imageActive = $derived(active?.kind === 'image');
    let previewActive = $derived(preview !== null && preview.id === activeId);
    // Either surface replaces the editor, which stays laid out behind it so a
    // text tab keeps its viewport. The welcome screen is the same layer when no
    // tab is open.
    let viewerActive = $derived(imageActive || previewActive);
    let welcomeActive = $derived(active === undefined && !searchActive);
    let paneReplaced = $derived(viewerActive || searchActive || welcomeActive);
    let language = $derived(
        imageActive
            ? 'Image'
            : active?.kind === 'diff'
              ? 'Diff'
              : languageLabel(active?.path ?? '', active?.content ?? '', active?.language ?? ''),
    );
    // Monaco keeps its buffers while an image tab is showing, so switching back
    // to a text tab preserves edits. The idle buffer keeps the hidden editor
    // from creating a model for an image path.
    let editorKey = $derived(imageActive ? WELCOME_KEY : (active?.id ?? WELCOME_KEY));
    let editorContent = $derived(imageActive ? '' : (active?.content ?? ''));
    // The active file's line ending decides the Monaco model's EOL, so edits
    // and the saved text use the same sequence the file started with.
    let editorEol = $derived(imageActive ? 'lf' : (active?.eol ?? 'lf'));
    // The status bar prints the active buffer's indentation; a tab with no
    // text buffer has none to configure.
    let indentationText = $derived(
        active?.kind === 'text'
            ? indentationLabel(indentation.insertSpaces, indentation.indentSize, indentation.tabSize)
            : '',
    );
    // savedContent is what a clean buffer would hold. A moved dirty tab carries
    // a different text than the file on disk, so its tab records the baseline.
    let editorSavedContent = $derived(
        imageActive ? '' : (active?.savedContent ?? active?.content ?? ''),
    );
    let openIds = $derived(tabs.map((tab) => tab.id));
    let unsaved = $derived(tabs.some((tab) => tab.dirty));
    let folderLabel = $derived(
        activeWorkspace !== null && activeWorkspace.name !== ''
            ? activeWorkspace.name
            : sidebarLabel(folders),
    );
    // The tree looks statuses up by path; the changes list keeps the ordered
    // records. Both come from the same snapshot.
    let gitStatuses = $derived.by(() => {
        const statuses: Record<string, GitStatusKind> = {};
        for (const change of gitChanges) {
            statuses[change.path] = change.status;
        }
        return statuses;
    });
    let activeRepo = $derived(
        repoForPath(gitRepos, active?.path || active?.sourcePath || ''),
    );
    let branch = $derived(branchLabel(activeRepo));
    let settingsWarning = $derived(
        keybindingError !== '' ? keybindingError : (themeErrors[0] ?? ''),
    );
    let imageSizeLabel = $derived(
        imageSize !== null && imageSize.id === activeId
            ? `${imageSize.width} × ${imageSize.height}`
            : '',
    );
    let searchGroups = $derived(groupSearchMatches(searchMatches));
    let searchLabel = $derived(
        searchSummary(searchGroups.length, searchMatches.length, searchTruncated),
    );
    let recentItems = $derived(toRecentItems(recentFiles));
    // The session records file-backed tabs only: an untitled buffer has no
    // text on disk, and a diff is a view of a file that is already recorded.
    let sessionFiles = $derived(
        tabs.filter((tab) => tab.kind === 'text' && tab.path !== '').map((tab) => tab.path),
    );
    // A diff tab records the file it shows, so a relaunch restores the file
    // itself as the active tab.
    let sessionActive = $derived(active?.path || active?.sourcePath || '');
    let watchedPaths = $derived(Array.from(new Set([...sessionFiles, ...expandedPaths])));
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
    // then makes it active. Restoring a session passes record=false, so a
    // relaunch does not reshuffle the Open Recent list.
    function showDocument(document: OpenDocument, record = true): void {
        const opened = openDocumentTab(tabs, document);
        tabs = opened.tabs;
        activeId = opened.activeId;
        searchActive = false;
        if (record && document.path !== '') {
            void recordRecent(document.path);
        }
        if (document.kind === 'text' && document.path !== '') {
            void loadBaseline(document.id, document.path);
        }
    }

    // addFolders saves each root from a launch request and adopts the result.
    // The backend deduplicates, so relaunching with the same folder is a no-op.
    // The first failure is returned so the caller can show it in the status bar.
    async function addFolders(paths: string[]): Promise<string> {
        let failure = '';
        let latest = folders;
        for (const path of paths) {
            try {
                latest = await addSidebarFolder(windowName, path);
            } catch (error) {
                if (failure === '') {
                    failure = errorMessage(error);
                }
            }
        }
        folders = latest;
        void refreshGitStatus();
        return failure;
    }

    async function applyLaunch(request: LaunchRequest): Promise<void> {
        let failure = '';
        if (request.replace_folders) {
            // Open Folder in New Window: the new window starts with exactly the
            // chosen folder, and the roots are saved for the next launch.
            try {
                folders = await setSidebarFolders(windowName, request.folders);
                void refreshGitStatus();
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
                await openFile(buffer.path);
                continue;
            }
            if (buffer.kind === 'diff' && buffer.source_path !== '') {
                await openDiff(buffer.source_path);
                continue;
            }
            showDocument(toBufferDocument(buffer), false);
        }

        for (const path of request.files) {
            const existing = findDocumentByPath(tabs, path);
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

    // openFile opens one file and returns the tab that now holds it, or an
    // empty id when it could not be read. The id is the path spelling the tab
    // was created with, which can differ from the one passed in when the file
    // is open through a symlink.
    async function openFile(path: string): Promise<string> {
        // Clicking a file that is already open focuses its tab instead of
        // rereading the file, which would discard edits made in the buffer.
        const existing = findDocumentByPath(tabs, path);
        if (existing !== undefined) {
            activeId = existing.id;
            notice = '';
            return existing.id;
        }
        const opened = await readDocument(path);
        if (opened === undefined) {
            notice = `Cannot open ${path}`;
            return '';
        }
        showDocument(opened);
        notice = '';
        return activeId;
    }

    // refreshGitStatus rereads the repositories behind the sidebar roots. It
    // runs when the workspace changes and after the app writes a file; a folder
    // outside git simply contributes nothing. A newer request wins, so adding
    // a root while an older read is still running cannot roll the result back.
    let gitRequest = 0;
    async function refreshGitStatus(): Promise<void> {
        const request = ++gitRequest;
        try {
            const state = await loadGitStatus(folders);
            if (request !== gitRequest) {
                return;
            }
            gitRepos = state.repos;
            gitChanges = state.changes;
        } catch (error) {
            console.error('Failed to read git status.', error);
        }
    }

    // loadBaseline gives one buffer its HEAD text so the gutter can mark the
    // lines that differ while the user types. An untracked file's baseline is
    // empty, which makes every line read as added.
    async function loadBaseline(id: string, path: string): Promise<void> {
        if (path === '') {
            return;
        }
        try {
            const baseline = await readGitBaseline(path);
            if (!baseline.inRepo) {
                editorPane?.clearBaseline(id);
                return;
            }
            editorPane?.setBaseline(id, baseline.content);
        } catch (error) {
            console.error('Failed to read the file baseline.', error);
            editorPane?.clearBaseline(id);
        }
    }

    // openDiff shows one file's changes against HEAD in a read-only tab. A
    // diff that is already open is focused instead of rebuilt.
    async function openDiff(path: string): Promise<void> {
        const id = diffTabId(path);
        if (tabs.some((tab) => tab.id === id)) {
            activeId = id;
            searchActive = false;
            return;
        }
        try {
            const diff = await readGitDiff(path);
            showDocument(createDiffDocument(path, diff), false);
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // reloadDiffTab replaces a diff's snapshot with a fresh one. The model is
    // read-only, so this is the only way its text changes.
    async function reloadDiffTab(id: string, path: string): Promise<void> {
        try {
            const document = createDiffDocument(path, await readGitDiff(path));
            tabs = updateDocument(tabs, id, { content: document.content });
            editorPane?.reload(id, document.content, document.eol);
        } catch (error) {
            console.error('Failed to refresh the diff.', error);
        }
    }

    async function refreshDiffTabs(): Promise<void> {
        for (const tab of tabs) {
            if (tab.kind === 'diff' && tab.sourcePath !== undefined) {
                await reloadDiffTab(tab.id, tab.sourcePath);
            }
        }
    }

    // refreshGitAfterSave updates the marks and any diff of the file that was
    // just written, so the sidebar does not describe the previous content.
    async function refreshGitAfterSave(path: string): Promise<void> {
        await refreshGitStatus();
        for (const tab of tabs) {
            if (tab.kind === 'diff' && tab.sourcePath === path) {
                await reloadDiffTab(tab.id, path);
            }
        }
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
        const tab = active;
        if (tab === undefined || tab.kind !== 'text') {
            return;
        }
        const eol: DocumentEOL = tab.eol === 'crlf' ? 'lf' : 'crlf';
        editorPane?.setEol(tab.id, eol);
        tabs = updateDocument(tabs, tab.id, { eol });
    }

    // newFile opens an empty buffer with no path yet. Save turns it into a
    // Save As, which is how most editors treat an untitled document.
    function newFile(): void {
        const document = createUntitledDocument(tabs);
        tabs = [...tabs, document];
        activeId = document.id;
        notice = '';
    }

    // newFileFromMenu is File → New File: the file is created inside a sidebar
    // folder, so it exists on disk from the start. The folder is the one the
    // selected entry lives in; with no folders there is nowhere to create one,
    // so the menu opens a buffer instead rather than doing nothing.
    async function newFileFromMenu(): Promise<void> {
        const folder = selectedFolder();
        if (folder === '') {
            notice = 'Add a folder to the sidebar to create a file.';
            newFile();
            return;
        }
        await newFileInFolder(folder);
    }

    // newFolderFromMenu is File → New Folder, in the selected entry's folder.
    async function newFolderFromMenu(): Promise<void> {
        const folder = selectedFolder();
        if (folder === '') {
            notice = 'Add a folder to the sidebar to create a folder.';
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
            return folders[0] ?? '';
        }
        const tab = findDocumentByPath(tabs, selected);
        if (tab !== undefined && tab.path !== '') {
            return parentPath(selected);
        }
        return selected;
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

    function askConfirm(
        title: string,
        message: string,
        confirmLabel: string,
    ): Promise<boolean> {
        return new Promise((resolve) => {
            confirmPrompt = {
                title,
                message,
                actions: [
                    { id: 'confirm', label: confirmLabel, primary: true },
                    { id: 'cancel', label: 'Cancel' },
                ],
                answer: (id) => resolve(id === 'confirm'),
            };
        });
    }

    // askConflict is the save prompt for a file another program changed. Reload
    // is a separate answer because the disk copy may be the newer one, and
    // discarding the buffer is then the right move.
    function askConflict(
        title: string,
        message: string,
    ): Promise<'overwrite' | 'reload' | 'cancel'> {
        return new Promise((resolve) => {
            confirmPrompt = {
                title,
                message,
                actions: [
                    { id: 'overwrite', label: 'Overwrite', primary: true },
                    { id: 'reload', label: 'Reload from Disk' },
                    { id: 'cancel', label: 'Cancel' },
                ],
                answer: (id) =>
                    resolve(id === 'overwrite' || id === 'reload' ? id : 'cancel'),
            };
        });
    }

    function answerConfirm(id: string): void {
        const prompt = confirmPrompt;
        confirmPrompt = null;
        prompt?.answer(id);
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
        const previousPath = findTab(id)?.path ?? '';
        // The editor decides whether the buffer is clean: typing during the
        // write leaves newer text in the model than the file now holds.
        const clean = editorPane?.markSaved(id, content) ?? true;
        tabs = updateDocument(tabs, id, {
            path: result.path,
            realPath: result.real_path || result.path,
            name: result.name,
            modified: result.modified,
            content,
            savedContent: undefined,
            dirty: !clean,
            // The file on disk is valid UTF-8 now; the replacement characters
            // it holds are what the user agreed to write.
            invalidUTF8: false,
            deleted: false,
        });
        // Save As can give an untitled buffer a real path, and with it a
        // language the model should start highlighting. An explicit override
        // still wins over the new extension.
        const saved = findTab(id);
        editorPane?.setLanguage(
            id,
            languageForDocument(
                result.path,
                editorPane?.value(id) ?? saved?.content ?? '',
                saved?.language ?? '',
            ),
        );
        if (previousPath !== result.path) {
            void loadBaseline(id, result.path);
        }
        void refreshGitAfterSave(result.path).catch((error) =>
            console.error('Failed to refresh the git status.', error),
        );
        notice = '';
    }

    // saveDocument writes one buffer, asking for a path when it has none. A file
    // another program changed is reported as a conflict first, so its content is
    // never overwritten without the user saying so. False means nothing was
    // written: a cancelled dialog, a conflict the user declined, or an error.
    async function saveDocument(id: string, saveAs = false): Promise<boolean> {
        const tab = findTab(id);
        // Pictures are read-only here, and a diff is a generated view, so
        // there is nothing to write back for either.
        if (tab === undefined || tab.kind !== 'text') {
            return true;
        }

        // ensure_final_newline edits the buffer before its text is read, so the
        // write and the model still agree and the tab does not stay dirty.
        if (editorSettings.ensureFinalNewline) {
            editorPane?.ensureFinalNewline(tab.id);
        }
        let content = bufferText(tab);
        if (editorSettings.ensureFinalNewline) {
            content = appendFinalNewline(content);
        }
        // The model's text is already normalized to the document's EOL, but a
        // buffer that was never shown comes straight from the tab. Normalizing
        // here keeps the baseline markSaved compares with the model.
        content = normalizeEol(content, tab.eol);
        let path = tab.path;
        let expected = tab.modified;
        if (saveAs || path === '') {
            path = await chooseSavePath(tab);
            if (path === '') {
                return false;
            }
            expected = 0;
            // Two tabs writing one file would disagree about its content, so
            // Save As refuses a path that is already open. The dialog returns a
            // resolved path, so an alias of an open file counts too.
            if (
                tabs.some(
                    (other) =>
                        other.id !== id && (other.path === path || other.realPath === path),
                )
            ) {
                notice = `${path} is already open.`;
                return false;
            }
        }

        let overwrite = false;
        let allowInvalidUTF8 = false;
        try {
            for (;;) {
                const result = await SaveDocument({
                    path,
                    content,
                    expected_modified: expected,
                    overwrite,
                    bom: tab.bom,
                    eol: tab.eol,
                    invalid_utf8: tab.invalidUTF8,
                    allow_invalid_utf8: allowInvalidUTF8,
                });
                if (result.invalid_utf8) {
                    const proceed = await askConfirm(
                        'Invalid UTF-8',
                        `${result.name} is not valid UTF-8. Saving replaces the bytes that could not be decoded with U+FFFD. Continue?`,
                        'Save Anyway',
                    );
                    if (!proceed) {
                        return false;
                    }
                    allowInvalidUTF8 = true;
                    continue;
                }
                if (result.conflict) {
                    const choice = await askConflict(
                        'File Changed on Disk',
                        `${result.name} changed on disk since it was opened. Overwrite it with your version, or reload it from disk?`,
                    );
                    if (choice === 'cancel') {
                        return false;
                    }
                    if (choice === 'reload') {
                        // The user chose the disk copy: replace the buffer and
                        // forget its edits. The tab is clean afterwards, so a
                        // save during close can go on to close it.
                        const opened = await readDocument(tab.path);
                        if (opened === undefined) {
                            notice = `Could not reload ${tab.name} from disk.`;
                            return false;
                        }
                        applyReload(id, opened);
                        void loadBaseline(id, tab.path);
                        void refreshDiffTabs();
                        void refreshGitStatus();
                        return true;
                    }
                    overwrite = true;
                    continue;
                }
                applySaved(id, result, content);
                return true;
            }
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
    // is also what discards its undo stack. The result is false when the user
    // cancelled or the save failed, so a group command can stop there.
    async function closeTab(id: string): Promise<boolean> {
        const tab = findTab(id);
        if (tab === undefined) {
            return false;
        }
        if (tab.dirty) {
            const answer = await askUnsaved({
                title: 'Close Tab',
                message: `Save changes to ${tab.name} before closing?`,
                details: [],
                confirmLabel: 'Save',
            });
            if (answer === 'cancel') {
                return false;
            }
            if (answer === 'save' && !(await saveDocument(id))) {
                return false;
            }
        }
        rememberClosedTab(tab);
        removeTab(id);
        return true;
    }

    // closeTabsInOrder closes a group one tab at a time, stopping at the first
    // cancelled prompt or failed save so the remaining tabs stay exactly as
    // they were.
    async function closeTabsInOrder(candidates: readonly OpenDocument[]): Promise<boolean> {
        for (const tab of candidates) {
            if (!(await closeTab(tab.id))) {
                return false;
            }
        }
        return true;
    }

    async function resolveClose(quit: boolean): Promise<void> {
        try {
            await ResolveClose(windowName, quit);
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // confirmClose answers Go's close request for this window. It is the one
    // place that decides whether the window may go away, so the close button
    // and Cmd+Q behave the same and share one prompt. The reason changes the
    // wording: closing one window is not quitting the app.
    async function confirmClose(reason: CloseReason = 'quit'): Promise<void> {
        // Another prompt is already on screen; the user is deciding, so a
        // second Cmd+Q must not stack a dialog on top of it. Answering the
        // request with "cancel" clears Go's pending flag, so the next Cmd+Q
        // asks again instead of being swallowed.
        if (unsavedPrompt !== null || confirmPrompt !== null) {
            await resolveClose(false);
            return;
        }

        const dirty = tabs.filter((tab) => tab.dirty);
        if (dirty.length === 0) {
            await resolveClose(true);
            return;
        }

        const answer = await askUnsaved({
            title: reason === 'close' ? 'Close Window' : `Quit ${appName}`,
            message:
                reason === 'close'
                    ? dirty.length === 1
                        ? 'Save changes before closing this window?'
                        : `${dirty.length} files in this window have unsaved changes.`
                    : dirty.length === 1
                      ? 'Save changes before quitting?'
                      : `${dirty.length} files have unsaved changes.`,
            details: dirty.map((tab) => tab.name),
            confirmLabel: dirty.length === 1 ? 'Save' : 'Save All',
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

    // Theme application: CSS variables cover the app chrome; Monaco gets its
    // own theme definition. A stable name per palette lets a live reload
    // replace the data and re-select the same theme.
    function applyThemePalette(palette: ThemePalette): void {
        applyThemeCSS(palette);
        cacheStartupTheme(palette);
        monaco.editor.defineTheme(monacoThemeName(palette.name), themeData(palette));
        monaco.editor.setTheme(monacoThemeName(palette.name));
    }

    async function applyCurrentTheme(): Promise<void> {
        const prefersDark = systemPrefersDark();
        const name = resolveThemeName(themeSettings, prefersDark);
        try {
            applyThemePalette(await loadTheme(name));
            return;
        } catch (error) {
            console.error('Failed to apply the selected theme.', error);
        }
        const fallback = prefersDark ? defaultThemeSettings.dark : defaultThemeSettings.light;
        try {
            applyThemePalette(await loadTheme(fallback));
            notice = `Theme "${name}" is unavailable.`;
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // refreshThemes lists installed themes and repairs a name that no longer
    // exists. The first theme for an appearance is the fallback, so deleting a
    // theme file does not leave the editor stuck on the old palette.
    async function refreshThemes(): Promise<void> {
        try {
            const catalog = await listThemes();
            themes = catalog.themes;
            themeErrors = catalog.errors;
            const lightFallback = catalog.themes.find(
                (entry) => entry.appearance === 'light',
            )?.name;
            const darkFallback = catalog.themes.find(
                (entry) => entry.appearance === 'dark',
            )?.name;
            const names = new Set(catalog.themes.map((entry) => entry.name));
            let next = themeSettings;
            if (!names.has(next.light) && lightFallback !== undefined) {
                next = { ...next, light: lightFallback };
            }
            if (!names.has(next.dark) && darkFallback !== undefined) {
                next = { ...next, dark: darkFallback };
            }
            if (next.light !== themeSettings.light || next.dark !== themeSettings.dark) {
                themeSettings = next;
                themeSettings = await saveThemeSettings(next);
            }
            if (themeErrors.length > 0) {
                notice = themeErrors[0];
            }
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function changeThemeSettings(next: ThemeSettings): Promise<void> {
        themeSettings = next;
        void SetAppearanceMode(next.mode).catch((error) =>
            console.error('Failed to update the Appearance menu.', error),
        );
        await applyCurrentTheme();
        try {
            themeSettings = await saveThemeSettings(next);
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function changeUISettings(next: UISettings): Promise<void> {
        uiSettings = next;
        applyUISettings(next);
        try {
            uiSettings = await saveUISettings(next);
            applyUISettings(uiSettings);
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // changeFilesExclude saves the explorer's glob list and rereads the tree:
    // the level already on screen was opened with the old matcher.
    async function changeFilesExclude(next: string[]): Promise<void> {
        filesExclude = next;
        try {
            filesExclude = await saveFilesExclude(next);
            treeRefreshToken++;
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function changeSearchExclude(next: string[]): Promise<void> {
        searchExclude = next;
        try {
            searchExclude = await saveSearchExclude(next);
        } catch (error) {
            notice = errorMessage(error);
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
            await OpenDefaultSettings();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function openThemesFolder(): Promise<void> {
        try {
            await OpenThemesFolder();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function openKeybindings(): Promise<void> {
        try {
            await OpenKeybindings();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // keybindingContext is what a `when` expression can test. Focus inside
    // Monaco's textarea still counts as editor focus.
    function keybindingContext(): KeyContext {
        const focused = document.activeElement;
        return {
            editorFocus: focused instanceof Element && focused.closest('.monaco-editor') !== null,
            sidebarVisible,
            hasActiveTab: activeId !== '',
            hasDirtyTab: unsaved,
        };
    }

    function knownCommand(id: string): boolean {
        return COMMANDS.some((command) => command.id === id);
    }

    function runKeybindingCommand(id: string): void {
        if (!knownCommand(id)) {
            notice = `Unknown keybinding command: ${id}`;
            return;
        }
        selectCommand(id);
    }

    async function refreshKeybindings(): Promise<void> {
        let state;
        try {
            state = await GetKeybindings();
        } catch (error) {
            keybindingError = errorMessage(error);
            return;
        }
        keybindingError = state.error;
        // A broken file did not decode; keep the previous listener (or start
        // with the defaults) and native menu rather than silently dropping
        // every shortcut. A file that
        // decoded has an authoritative list even when entries were rejected.
        if (!state.parsed) {
            disposeKeybindings ??= installKeybindings(withDefaultKeybindings([]), {
                run: runKeybindingCommand,
                context: keybindingContext,
            });
            return;
        }
        const bindings = state.bindings ?? [];
        const actionable = bindings.filter((binding) => !binding.command.startsWith('-'));
        const usable = actionable.filter(isUsableBinding);
        if (usable.length < actionable.length) {
            const dropped = actionable.length - usable.length;
            keybindingError = `${keybindingError === '' ? '' : `${keybindingError}; `}${dropped} binding(s) need a modifier or a non-printing key`;
        }
        disposeKeybindings?.();
        // installKeybindings skips "-command" removals and unusable entries.
        disposeKeybindings = installKeybindings(withDefaultKeybindings(bindings), {
            run: runKeybindingCommand,
            context: keybindingContext,
        });
        if (keybindingError !== '') {
            notice = keybindingError;
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

    // togglePreview shows the active buffer as Markdown or SVG. The preview is a
    // snapshot of the buffer; the hidden editor keeps its model untouched, so
    // hiding the preview returns to the same viewport.
    function togglePreview(): void {
        if (previewActive) {
            preview = null;
            return;
        }
        const tab = active;
        const kind = previewKindFor(tab?.path ?? '');
        if (tab === undefined || kind === '') {
            notice = 'Preview is available for Markdown and SVG files.';
            return;
        }
        preview = { id: tab.id, kind, content: bufferText(tab) };
        notice = '';
    }

    function transformSelection(transform: CaseTransform): void {
        editorPane?.transformSelection(transform);
    }

    function trimWhitespace(): void {
        editorPane?.trimTrailingWhitespace();
    }

    function runSelectionCommand(command: SelectionCommand): void {
        editorPane?.runSelectionCommand(command);
    }

    // runEditorAction runs one of Monaco's own editor actions on the active
    // buffer: its Find and Replace widgets and the Go to Line prompt. Those
    // widgets belong to the editor, so the Find in Files results panel gives
    // way first rather than hiding them.
    function runEditorAction(action: string): void {
        if (active === undefined) {
            notice = 'Open a document first.';
            return;
        }
        searchActive = false;
        editorPane?.runEditorAction(action);
    }

    function zoom(command: ZoomCommand): void {
        editorPane?.zoom(command);
    }

    // changeEditorSettings applies the whole editor section, saves it, and moves
    // the View menu checkmarks. Saves are chained and only the newest result is
    // applied, so two quick changes cannot write the older values last.
    function changeEditorSettings(next: EditorSettings): void {
        editorSettings = next;
        const version = ++editorSaveVersion;
        editorSaveQueue = editorSaveQueue
            .catch(() => {})
            .then(async () => {
                const saved = await saveEditorSettings(next);
                if (version !== editorSaveVersion) {
                    return;
                }
                editorSettings = saved;
                for (const option of VIEW_OPTIONS) {
                    void SetViewOption(option.id, saved[option.id]).catch((error) =>
                        console.error('Failed to update the View menu.', error),
                    );
                }
            })
            .catch((error) => {
                if (version === editorSaveVersion) {
                    notice = errorMessage(error);
                }
            });
    }

    // setViewOption applies a view toggle, saves it, and moves the View menu's
    // checkmark when the toggle came from the command palette instead.
    function setViewOption(option: ViewOption, checked: boolean): void {
        if (editorSettings[option] === checked) {
            return;
        }
        changeEditorSettings({ ...editorSettings, [option]: checked });
    }

    function toggleViewOption(option: ViewOption): void {
        setViewOption(option, !editorSettings[option]);
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
            const list = await listFiles(windowName);
            quickOpenFiles = toQuickOpenFiles(list.files, folders);
            quickOpenTruncated = list.truncated;
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

    // Change Language Mode lists Auto Detect and every language Monaco
    // registered. It is only meaningful for a text tab: a diff has its own
    // fixed syntax and an image has none.
    function rankLanguages(query: string): PaletteItem[] {
        const items = languageModeItems ?? (languageModeItems = languageChoices());
        return rankCandidates(query, items, items.length);
    }

    function showLanguagePalette(): void {
        if (active === undefined || active.kind !== 'text') {
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
        const tab = active;
        if (tab === undefined || tab.kind !== 'text') {
            return;
        }
        const override = id === AUTO_LANGUAGE_ID ? undefined : id;
        tabs = updateDocument(tabs, tab.id, { language: override });
        const content = editorPane?.value(tab.id) ?? tab.content;
        editorPane?.setLanguage(tab.id, languageForDocument(tab.path, content, override ?? ''));
    }

    // rankIndentation lists the actions the status bar's picker offers. The
    // rows are static, so an empty query shows the whole list.
    function rankIndentation(query: string): PaletteItem[] {
        return rankCandidates(query, INDENT_PALETTE_COMMANDS, INDENT_PALETTE_COMMANDS.length);
    }

    function showIndentationPalette(): void {
        if (active === undefined || active.kind !== 'text') {
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
        if (spec === undefined || active === undefined || active.kind !== 'text') {
            return;
        }
        searchActive = false;
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
        if (activeId !== '') {
            void saveDocument(activeId, saveAs);
        }
    }

    // runWindowCommand reports a failed native window action in the status bar.
    function runWindowCommand(command: () => Promise<void>): void {
        void command().catch((error) => {
            notice = errorMessage(error);
        });
    }

    // moveTabToNewWindow hands one tab to a new window and drops it from this
    // one. The buffer text travels with the request, so unsaved edits are not
    // lost; the tab is removed only after the call succeeds. Without an id the
    // active tab is moved, which is what the menu command does.
    async function moveTabToNewWindow(id: string = activeId): Promise<void> {
        const tab = findTab(id);
        if (tab === undefined) {
            notice = 'Open a tab to move to a new window.';
            return;
        }
        const buffer: LaunchBuffer = {
            name: tab.name,
            path: tab.kind === 'diff' ? '' : tab.path,
            real_path: tab.kind === 'diff' ? '' : tab.realPath,
            kind: tab.kind,
            content: tab.kind === 'text' ? bufferText(tab) : tab.content,
            saved_content: tab.content,
            modified: tab.modified,
            dirty: tab.kind === 'text' ? tab.dirty : false,
            source_path: tab.sourcePath ?? '',
            bom: tab.bom,
            eol: tab.eol,
            invalid_utf8: tab.invalidUTF8,
        };
        try {
            await MoveTabToNewWindow(buffer);
            removeTab(tab.id);
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // selectCommand closes the palette before running the command, so commands
    // that open another dialog (Quick Open, the folder picker) start clean.
    function selectCommand(id: string): void {
        commandPalette = false;

        // The case and view toggles are generated from their shared tables, so a
        // new transform needs no second switch arm here.
        const transform = CASE_TRANSFORMS.find((entry) => `case-${entry.id}` === id);
        if (transform !== undefined) {
            transformSelection(transform.id);
            return;
        }
        const viewOption = VIEW_OPTIONS.find((entry) => `toggle-${entry.id}` === id);
        if (viewOption !== undefined) {
            toggleViewOption(viewOption.id);
            return;
        }
        const selectionCommand = SELECTION_COMMANDS.find((entry) => entry.id === id);
        if (selectionCommand !== undefined) {
            runSelectionCommand(selectionCommand.id);
            return;
        }
        const indentCommand = INDENT_COMMANDS.find((entry) => entry.id === id);
        if (indentCommand !== undefined) {
            runIndentAction(indentCommand.id);
            return;
        }

        if (id.startsWith('go-to-tab-')) {
            goToTab(Number(id.slice('go-to-tab-'.length)));
            return;
        }

        switch (id) {
            case 'new-file':
                void newFileFromMenu();
                break;
            case 'new-folder':
                void newFolderFromMenu();
                break;
            case 'new-buffer':
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
            case 'trim-whitespace':
                trimWhitespace();
                break;
            case 'change-language-mode':
                showLanguagePalette();
                break;
            case 'find':
                runEditorAction('actions.find');
                break;
            case 'replace':
                runEditorAction('editor.action.startFindReplaceAction');
                break;
            case 'go-to-line':
                runEditorAction('editor.action.gotoLine');
                break;
            case 'find-next-result':
                void stepFindResult(1);
                break;
            case 'find-previous-result':
                void stepFindResult(-1);
                break;
            case 'toggle-preview':
                togglePreview();
                break;
            case 'show-explorer':
                switchSidebarView('explorer');
                break;
            case 'show-changes':
                switchSidebarView('changes');
                break;
            case 'zoom-in':
                zoom('in');
                break;
            case 'zoom-out':
                zoom('out');
                break;
            case 'zoom-reset':
                zoom('reset');
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
            case 'command-palette':
                showCommandPalette();
                break;
            case 'settings':
                showSettingsOverlay();
                break;
            case 'about':
                void ShowAbout();
                break;
            case 'open-settings-json':
                void openSettings();
                break;
            case 'open-default-settings':
                void openDefaultSettings();
                break;
            case 'open-themes-folder':
                void openThemesFolder();
                break;
            case 'open-keybindings':
                void openKeybindings();
                break;
            case 'appearance-system':
            case 'appearance-light':
            case 'appearance-dark':
                void changeThemeSettings({
                    ...themeSettings,
                    mode: id.slice('appearance-'.length) as ThemeSettings['mode'],
                });
                break;
            case 'install-command-line-tool':
                void installCommandLineTool();
                break;
            case 'open-recent':
                showRecentFiles();
                break;
            case 'clear-recent':
                void clearRecent();
                break;
            case 'open-workspace':
                void openWorkspaceFromDialog();
                break;
            case 'save-workspace-as':
                void saveWorkspaceAsDialog();
                break;
            case 'close-workspace':
                void closeWorkspace();
                break;
            case 'minimize-window':
                runWindowCommand(() => MinimiseWindow(windowName));
                break;
            case 'zoom-window':
                runWindowCommand(() => ZoomWindow(windowName));
                break;
            case 'toggle-full-screen':
                runWindowCommand(() => ToggleFullScreen(windowName));
                break;
            case 'new-window':
                runWindowCommand(() => NewWindow());
                break;
            case 'open-folder-new-window':
                runWindowCommand(() => OpenFolderInNewWindow());
                break;
            case 'close-window':
                runWindowCommand(() => CloseWindow(windowName));
                break;
            case 'move-tab-new-window':
                void moveTabToNewWindow();
                break;
            case 'bring-all-to-front':
                runWindowCommand(() => BringAllToFront());
                break;
            case 'close-all-folders':
                void closeAllFolders();
                break;
            case 'close-all-tabs':
                void closeAllTabs();
                break;
            case 'reopen-closed-tab':
                void reopenClosedTab();
                break;
            case 'next-tab':
                stepTab(1);
                break;
            case 'previous-tab':
                stepTab(-1);
                break;
            case 'find-in-files':
                showFindInFiles();
                break;
            case 'refresh':
                void refreshWorkspace();
                break;
            case 'toggle-sidebar':
                toggleSidebar();
                break;
            case 'reveal-file':
                void revealActiveFile();
                break;
            case 'copy-file-path':
                void copyActivePath('file');
                break;
            case 'copy-relative-path':
                void copyActivePath('relative');
                break;
            case 'copy-file-name':
                void copyActivePath('name');
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
            folders = await addSidebarFolder(windowName, path);
            void refreshGitStatus();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        } finally {
            busy = false;
        }
    }

    // restoreWorkspaceTabs opens the file-backed tabs parked under a workspace.
    // Missing files are skipped, so one deleted file does not stop the rest.
    async function restoreWorkspaceTabs(session: SavedWorkspaceSession): Promise<void> {
        for (const path of session.openFiles) {
            if (findDocumentByPath(tabs, path) !== undefined) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            showDocument(opened, false);
        }
        const restoredActive = findDocumentByPath(tabs, session.active);
        if (restoredActive !== undefined) {
            activeId = restoredActive.id;
        }
    }

    // clearWorkspaceTabs drops every document tab and the Find Results tab.
    // The old set has already been parked in session.json, and the Reopen
    // Closed Tab stack is cleared so it cannot pull tabs from another
    // workspace back into this one.
    function clearWorkspaceTabs(): void {
        tabs = [];
        activeId = '';
        closedTabs = [];
        if (searchOpen) {
            closeSearchTab();
        }
    }

    // prepareWorkspaceSwitch is the one prompt before a switch replaces the
    // tab set. Saving all is the default because the parked tab set is a
    // convenience, not a place to discard edits.
    async function prepareWorkspaceSwitch(title: string, message: string): Promise<boolean> {
        const dirty = tabs.filter((tab) => tab.dirty);
        if (dirty.length === 0) {
            return true;
        }
        const answer = await askUnsaved({
            title,
            message,
            details: dirty.map((tab) => tab.path || tab.name),
            confirmLabel: 'Save All and Continue',
        });
        if (answer === 'cancel') {
            return false;
        }
        if (answer === 'save') {
            return await saveAll();
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
            folders = await setSidebarFolders(windowName, target.folders ?? []);
            activeWorkspace = target;
            const parked = toWorkspaceSession(target.session);
            expandedPaths = parked.expanded;
            clearWorkspaceTabs();
            await restoreWorkspaceTabs(parked);
            switched = true;
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        } finally {
            sessionSuspended = false;
        }
        if (switched) {
            void refreshGitStatus();
            await persistSession();
        }
        return switched;
    }

    // openWorkspaceFromDialog replaces the window's workspace with the picked
    // file. The previous tab set stays parked under its own workspace.
    async function openWorkspaceFromDialog(): Promise<void> {
        if (busy || unsavedPrompt !== null || confirmPrompt !== null) {
            return;
        }
        busy = true;
        try {
            const workspace = await openWorkspaceDialog();
            if (workspace.path === '') {
                return;
            }
            await switchWorkspace(workspace);
        } catch (error) {
            notice = errorMessage(error);
        } finally {
            busy = false;
        }
    }

    // saveWorkspaceAsDialog writes the current roots to a workspace file, binds
    // the window to it and parks the current tabs under the new name.
    async function saveWorkspaceAsDialog(): Promise<void> {
        if (busy || unsavedPrompt !== null || confirmPrompt !== null) {
            return;
        }
        busy = true;
        try {
            const workspace = await saveWorkspaceAs(folders);
            if (workspace.path === '') {
                return;
            }
            activeWorkspace = workspace;
            await persistSession();
            notice = `Saved workspace ${workspace.name}`;
        } catch (error) {
            notice = errorMessage(error);
        } finally {
            busy = false;
        }
    }

    // closeWorkspace clears the roots and closes the tabs after the save
    // prompt. The parked tab set stays in session.json, so reopening the
    // workspace later restores it.
    async function closeWorkspace(): Promise<void> {
        if (busy || unsavedPrompt !== null || confirmPrompt !== null) {
            return;
        }
        if (activeWorkspace === null && folders.length === 0) {
            return;
        }
        busy = true;
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
                folders = await closeAllSidebarFolders(windowName);
                activeWorkspace = null;
                expandedPaths = [];
                clearWorkspaceTabs();
                notice = '';
            } finally {
                sessionSuspended = false;
            }
            void refreshGitStatus();
            await persistSession();
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
            folders = await removeSidebarFolder(windowName, path);
            if (folders.length === 0) {
                activeWorkspace = null;
            }
            void refreshGitStatus();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // rememberClosedTab pushes a tab onto the Reopen Closed Tab stack, newest
    // last. The stack is capped so a long session cannot grow without bound.
    function rememberClosedTab(tab: OpenDocument): void {
        closedTabs.push({ document: { ...tab }, content: bufferText(tab) });
        if (closedTabs.length > 20) {
            closedTabs.shift();
        }
    }

    async function reopenClosedTab(): Promise<void> {
        const closed = closedTabs.pop();
        if (closed === undefined) {
            notice = 'No closed tab to reopen.';
            return;
        }

        const saved = closed.document;
        // A clean file tab is reopened from disk. Closing a dirty tab already
        // discarded its edits, so there is nothing newer to restore.
        if (saved.kind === 'text' && saved.path !== '' && !saved.dirty) {
            const opened = await readDocument(saved.path);
            if (opened !== undefined) {
                showDocument(opened);
                notice = '';
                return;
            }
        }

        // An untitled buffer lives only in memory, so its text comes from the
        // stack. A file that disappeared falls back to the same snapshot.
        let restored = saved;
        if (saved.kind === 'text' && saved.path === '' && tabs.some((tab) => tab.id === saved.id)) {
            // A New File made since the close may already own this name; give
            // the restored buffer a fresh id so the two do not share a model.
            const replacement = createUntitledDocument(tabs);
            restored = { ...saved, id: replacement.id, name: replacement.name };
        }
        showDocument({
            ...restored,
            content: closed.content,
            // The model's clean baseline is the file's text (nothing for an
            // untitled buffer), not the edited snapshot, or undo could mark
            // the reopened edits clean.
            savedContent:
                restored.path === '' ? '' : (restored.savedContent ?? restored.content),
            dirty:
                restored.kind === 'text' &&
                (restored.dirty || (restored.path === '' && closed.content !== '')),
        });
        notice = '';
    }

    function stepTab(direction: 1 | -1): void {
        if (tabs.length === 0) {
            return;
        }
        searchActive = false;
        activeId = stepTabId(tabs, activeId, direction);
    }

    function goToTab(position: number): void {
        const id = tabIdAt(tabs, position);
        if (id === '') {
            return;
        }
        searchActive = false;
        activeId = id;
    }

    // reorderTab keeps the dragged tab on the same side of the target the
    // insertion line showed. The tab list order is the saved session order.
    function reorderTab(id: string, targetId: string, placeAfter: boolean): void {
        tabs = moveDocumentTab(tabs, id, targetId, placeAfter);
    }

    // ---- File tree operations -------------------------------------------------
    //
    // New File, New Folder, Rename, Duplicate, Cut/Copy/Paste, Delete (to the
    // Trash) and drag to move all run against the explorer. Every one of them
    // is a Go binding, because the file system is the backend's business; the
    // frontend only decides which entry, asks for a name and retargets the tabs
    // that follow a path.

    // askName opens the one-line dialog and resolves with the name the user
    // submitted, or an empty string when they cancelled. The dialog stays open
    // while a refusal is shown in it, so a taken name is corrected in place.
    function askName(options: {
        title: string;
        label?: string;
        value?: string;
        selectStem?: boolean;
        selectFolder?: boolean;
        submitLabel?: string;
        run: (value: string) => Promise<string>;
    }): Promise<boolean> {
        return new Promise((resolve) => {
            const answer = (value: string): void => {
                // The dialog was already closed by the caller's submit path, so
                // the state is only read to know what to do next.
                void (async () => {
                    const refusal = await options.run(value);
                    if (refusal === '') {
                        resolve(true);
                        return;
                    }
                    namePrompt = {
                        title: options.title,
                        label: options.label ?? '',
                        value,
                        selectStem: options.selectStem ?? false,
                        selectFolder: options.selectFolder ?? false,
                        submitLabel: options.submitLabel ?? options.title,
                        error: refusal,
                        answer,
                        cancel: () => {
                            namePrompt = null;
                            resolve(false);
                        },
                    };
                })();
            };
            namePrompt = {
                title: options.title,
                label: options.label ?? '',
                value: options.value ?? '',
                selectStem: options.selectStem ?? false,
                selectFolder: options.selectFolder ?? false,
                submitLabel: options.submitLabel ?? options.title,
                error: '',
                answer,
                cancel: () => {
                    namePrompt = null;
                    resolve(false);
                },
            };
        });
    }

    // submitName closes the dialog and hands the value to the pending command,
    // which reopens it with the refusal if the backend says no.
    function submitName(value: string): void {
        const prompt = namePrompt;
        namePrompt = null;
        prompt?.answer(value);
    }

    function cancelName(): void {
        const prompt = namePrompt;
        namePrompt = null;
        prompt?.cancel();
    }

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
        const previous = tabs;
        const next = retargetDocumentPaths(tabs, [{ from, to }]);
        if (next === previous) {
            return;
        }
        tabs = next;
        for (const tab of next) {
            const before = previous.find((entry) => entry.id === tab.id);
            if (before === undefined || before.path === tab.path) {
                continue;
            }
            // HEAD still knows the old name, so the gutter bars are recomputed
            // against the path the file now has.
            if (tab.kind === 'text') {
                void loadBaseline(tab.id, tab.path);
            }
        }
        // A diff tab followed its file, so the snapshot it shows is stale.
        if (next.some((tab) => tab.kind === 'diff')) {
            void refreshDiffTabs();
        }
        void refreshGitStatus();
    }

    // treeError reports a failed tree command in the status bar, where every
    // other notice appears, and keeps the tree as it was.
    function treeError(error: unknown): void {
        notice = errorMessage(error);
    }

    // newFileInFolder is New File from the tree: the file is created on disk
    // first, so an empty file exists even if the user never types into it, then
    // opened as a tab.
    async function newFileInFolder(folder: string): Promise<void> {
        await askName({
            title: 'New File',
            label: `Name for the new file in ${entryName(folder)}`,
            submitLabel: 'Create File',
            run: async (name) => {
                try {
                    const entry = await createFile(folder, name);
                    refreshTree();
                    await openFile(entry.path);
                    notice = '';
                    return '';
                } catch (error) {
                    notice = '';
                    return errorMessage(error);
                }
            },
        });
    }

    // newFolderInFolder is New Folder. The new folder is expanded so the user
    // sees where it went, which matters when it lands between many siblings.
    async function newFolderInFolder(folder: string): Promise<void> {
        await askName({
            title: 'New Folder',
            label: `Name for the new folder in ${entryName(folder)}`,
            submitLabel: 'Create Folder',
            run: async (name) => {
                try {
                    const entry = await createFolder(folder, name);
                    refreshTree();
                    revealInTree(entry.path);
                    notice = '';
                    return '';
                } catch (error) {
                    notice = '';
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
        await askName({
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
                    notice = '';
                    return '';
                } catch (error) {
                    notice = '';
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
        await askName({
            title: 'Duplicate',
            value: suggestion,
            selectStem: true,
            submitLabel: 'Duplicate',
            run: async (target) => {
                try {
                    const entry = await duplicatePath(path, target);
                    refreshTree();
                    revealInTree(entry.path);
                    await refreshGitStatus();
                    notice = `Created ${entry.name}`;
                    return '';
                } catch (error) {
                    notice = '';
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
        notice = `Ready to move ${entryName(path)}`;
    }

    function copyEntry(path: string): void {
        clipboardPath = path;
        clipboardCut = false;
        notice = `Ready to copy ${entryName(path)}`;
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
                notice = `Moved ${entry.name}`;
            } else {
                const entry = await copyPath(source, folder);
                revealInTree(entry.path);
                notice = `Copied ${entry.name}`;
            }
            refreshTree();
            await refreshGitStatus();
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
        void refreshGitStatus();
        notice = `Moved ${name}`;
    }

    // moveTreeEntry is Move…: the field holds the entry's full path with the
    // folder selected, so typing replaces where it lives and the file name stays
    // visible and intact. Move and Rename share the path operation; only the
    // initial selection differs. A refusal reopens the prompt on the same text
    // so a typo is corrected in place.
    async function moveTreeEntry(path: string): Promise<void> {
        await askName({
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
                    notice = '';
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
        const affected = tabs.filter(
            (tab) =>
                (tab.path !== '' && pathWithin(tab.path, path)) ||
                (tab.kind === 'diff' && pathWithin(tab.sourcePath ?? '', path)),
        );
        const dirty = affected.filter((tab) => tab.dirty);
        const confirmed = await askConfirm(
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
                removeTab(tab.id);
            }
            if (pathWithin(clipboardPath, path)) {
                clipboardPath = '';
                clipboardCut = false;
            }
            refreshTree();
            await refreshGitStatus();
            notice = `Moved ${entryName(path)} to the Trash`;
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

    // contextMenuItems builds the row menu. A tree row that is a file creates in
    // the folder it lives in, which is why it offers the same items; a tab row
    // offers the tab commands. A separator is an item with an empty id, which
    // the menu draws as a line.
    function contextMenuItems(): ContextMenuItem[] {
        const target = contextMenu;
        if (target === null) {
            return [];
        }
        if (target.tabId !== undefined) {
            return tabContextMenuItems(target.tabId);
        }
        return treeMenuItems(clipboardPath);
    }

    // tabContextMenuItems is the menu for the tab under the pointer. Commands
    // are relative to that tab, not to whichever tab happens to be active.
    function tabContextMenuItems(tabId: string): ContextMenuItem[] {
        const tab = findTab(tabId);
        if (tab === undefined) {
            return [];
        }
        const reveal = tab.path || tab.sourcePath || '';
        const index = tabs.findIndex((entry) => entry.id === tabId);
        const items: ContextMenuItem[] = [
            { id: 'close-tab', label: 'Close Tab' },
            { id: 'move-tab-new-window', label: 'Move to New Window' },
            { id: '', label: '' },
            {
                id: 'close-other-tabs',
                label: 'Close Other Tabs',
                disabled: tabs.length === 1 && !searchOpen,
            },
            {
                id: 'close-right-tabs',
                label: 'Close Tabs to Right',
                disabled: index === -1 || (index === tabs.length - 1 && !searchOpen),
            },
            {
                id: 'close-deleted-tabs',
                label: 'Close Tabs with Deleted Files',
                disabled: !tabs.some((entry) => entry.deleted),
            },
            { id: '', label: '' },
            {
                id: 'close-all-tabs',
                label: 'Close All',
                disabled: tabs.length === 0 && !searchOpen,
            },
        ];
        if (reveal !== '') {
            items.push(
                { id: '', label: '' },
                { id: 'reveal', label: 'Reveal in Finder' },
            );
        }
        return items;
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
        const tab = findTab(tabId);
        if (tab === undefined) {
            return;
        }
        switch (id) {
            case 'reveal':
                await revealPath(tab.path || tab.sourcePath || '');
                break;
            case 'close-tab':
                await closeTab(tabId);
                break;
            case 'move-tab-new-window':
                await moveTabToNewWindow(tabId);
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
        if (!(await closeTabsInOrder(tabsToClose(tabs, id, 'others')))) {
            return;
        }
        if (searchOpen) {
            closeSearchTab();
        }
        activeId = id;
    }

    // closeTabsToRight closes every tab after the right-clicked one, including
    // the Find in Files tab, which is always rendered after the documents.
    async function closeTabsToRight(id: string): Promise<void> {
        if (findTab(id) === undefined) {
            return;
        }
        if (!(await closeTabsInOrder(tabsToClose(tabs, id, 'right')))) {
            return;
        }
        if (searchOpen) {
            closeSearchTab();
        }
    }

    // closeAllTabs closes every document tab and the Find in Files tab. Dirty
    // buffers are asked about one at a time, and a cancelled prompt stops the
    // run with the remaining tabs still open.
    async function closeAllTabs(): Promise<void> {
        if (!(await closeTabsInOrder(tabsToClose(tabs, '', 'all')))) {
            return;
        }
        if (searchOpen) {
            closeSearchTab();
        }
    }

    // closeDeletedTabs closes the buffers whose files are gone. A dirty one
    // still asks, because closing it discards text that could recreate the file.
    async function closeDeletedTabs(): Promise<void> {
        await closeTabsInOrder(tabsToClose(tabs, '', 'deleted'));
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
        await openFile(path);
    }

    // copyTreeText copies a path spelling of one tree entry. The variants match
    // the Copy Path items that act on the active buffer, so the same three
    // strings are reachable from the tree.
    async function copyTreeText(kind: CopyPathKind, path: string): Promise<void> {
        const text =
            kind === 'file'
                ? path
                : kind === 'relative'
                  ? relativePath(path, folders)
                  : entryName(path);
        try {
            await CopyText(text);
            notice = `Copied ${text}`;
        } catch (error) {
            notice = errorMessage(error);
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

    function toRecentItems(paths: readonly string[]): PaletteItem[] {
        return paths.map((path) => {
            const file = toQuickOpenFile(path, folders);
            return { id: path, name: file.name, detail: file.detail };
        });
    }

    function showRecentFiles(): void {
        if (recentFiles.length === 0) {
            notice = 'No recent files.';
            return;
        }
        quickOpen = false;
        commandPalette = false;
        languagePalette = false;
        indentationPalette = false;
        searchActive = false;
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
            notice = 'Recent files cleared.';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // revealActiveFileInTree selects and expands to the active file's row. A
    // diff tab names the file it was made from, which is the row to show.
    $effect(() => {
        if (!sidebarAutoReveal) {
            return;
        }
        revealedPath = active?.path || active?.sourcePath || '';
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
            folders = await closeAllSidebarFolders(windowName);
            activeWorkspace = null;
            expandedPaths = [];
            void refreshGitStatus();
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // applyReload replaces a clean buffer with what is on disk. The model is
    // told first, so the editor's dirty check agrees with the tab record.
    function applyReload(id: string, opened: OpenDocument): void {
        if (opened.kind === 'text') {
            editorPane?.reload(id, opened.content, opened.eol);
        }
        tabs = updateDocument(tabs, id, {
            kind: opened.kind,
            content: opened.content,
            savedContent: undefined,
            src: opened.src,
            modified: opened.modified,
            dirty: false,
            bom: opened.bom,
            eol: opened.eol,
            invalidUTF8: opened.invalidUTF8,
            realPath: opened.realPath || opened.path,
            deleted: false,
        });
    }

    // markDeletedIfGone flags a tab whose file no longer exists, keeping its
    // buffer so a save writes it back. A read can fail for other reasons (a
    // binary swapped in, lost permissions), so only a missing file marks the
    // tab; anything else stays silent like before.
    async function markDeletedIfGone(tab: OpenDocument): Promise<boolean> {
        if (tab.deleted) {
            return true;
        }
        let exists = true;
        try {
            exists = await FileExists(tab.path);
        } catch (error) {
            console.error(`Failed to check ${tab.path}.`, error);
            return false;
        }
        if (!exists) {
            tabs = updateDocument(tabs, tab.id, { deleted: true });
            return true;
        }
        return false;
    }

    // reloadIfClean applies a read that was started while the tab was clean.
    // The user may have typed while it was in flight, and those edits win.
    function reloadIfClean(id: string, opened: OpenDocument): void {
        const current = findTab(id);
        if (current === undefined || current.dirty) {
            return;
        }
        applyReload(id, opened);
    }

    async function refreshOpenDocuments(): Promise<void> {
        for (const snapshot of tabs) {
            // Earlier reads await, so the tab may have changed or closed since
            // the loop started.
            const tab = findTab(snapshot.id);
            if (tab === undefined || tab.kind !== 'text' || tab.path === '') {
                continue;
            }
            if (!tab.dirty) {
                // External changes never overwrite edits the user has not
                // saved; the next save reports the conflict instead.
                const opened = await readDocument(tab.path);
                if (opened !== undefined) {
                    reloadIfClean(tab.id, opened);
                    continue;
                }
            }
            // The file could not be read, or the tab is dirty so reading was
            // skipped: only a file that is really gone marks its tab.
            await markDeletedIfGone(tab);
        }
    }

    async function refreshWorkspace(): Promise<void> {
        treeRefreshToken++;
        await refreshOpenDocuments();
        await refreshDiffTabs();
        await refreshGitStatus();
        // HEAD may have moved since the buffers were opened, so the gutter
        // baselines are reread too.
        for (const tab of tabs) {
            if (tab.kind === 'text' && tab.path !== '') {
                void loadBaseline(tab.id, tab.path);
            }
        }
        notice = '';
    }

    // handleWatchedChanges reloads clean buffers and refreshes the tree for
    // anything else. A dirty buffer stays as it is and says why; a deleted
    // file marks its tab either way, keeping the buffer for a save.
    async function handleWatchedChanges(paths: string[]): Promise<void> {
        let refreshTree = false;
        for (const path of paths) {
            const tab = findDocumentByPath(tabs, path);
            if (tab === undefined || tab.kind !== 'text') {
                refreshTree = true;
                continue;
            }
            if (tab.dirty) {
                if (!(await markDeletedIfGone(tab))) {
                    notice = `${tab.name} changed on disk; your edits were kept.`;
                }
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                await markDeletedIfGone(tab);
                continue;
            }
            reloadIfClean(tab.id, opened);
        }
        if (refreshTree) {
            treeRefreshToken++;
        }
        // A file that changed on disk is exactly what the marks and an open
        // diff describe, so both follow the same reread.
        void refreshGitStatus();
        void refreshDiffTabs();
    }

    function showFindInFiles(): void {
        quickOpen = false;
        commandPalette = false;
        recentOpen = false;
        languagePalette = false;
        indentationPalette = false;
        searchOpen = true;
        searchActive = true;
        searchFocusToken++;
    }

    function closeSearchTab(): void {
        searchOpen = false;
        searchActive = false;
        searchMatches = [];
        searchResultIndex = -1;
        searchError = '';
    }

    async function runSearch(): Promise<void> {
        const query = searchQuery.trim();
        if (query === '') {
            searchMatches = [];
            searchResultIndex = -1;
            searchError = '';
            searchedParams = {
                query: '',
                regex: false,
                caseSensitive: false,
                include: '',
                exclude: '',
            };
            return;
        }
        searchBusy = true;
        searchError = '';
        try {
            const result = await SearchProject(windowName, {
                query,
                regex: searchRegex,
                case_sensitive: searchCaseSensitive,
                include: searchInclude,
                exclude: searchExcludeGlobs,
                replacement: '',
                expected_modified: null,
                open_paths: null,
            });
            searchMatches = result.matches ?? [];
            searchResultIndex = -1;
            searchTruncated = result.truncated;
            searchedParams = {
                query,
                regex: searchRegex,
                caseSensitive: searchCaseSensitive,
                include: searchInclude,
                exclude: searchExcludeGlobs,
            };
        } catch (error) {
            searchMatches = [];
            searchResultIndex = -1;
            searchError = errorMessage(error);
        } finally {
            searchBusy = false;
        }
    }

    // replaceInFiles is the safe Replace All. It confirms the scope first, then
    // lets Go write the files whose mtime still matches the search; files that
    // changed since are skipped and named in the result. Open buffers get the
    // same edit through Monaco so it stays unsaved and undoable, and the app
    // never reloads a buffer out from under the user's undo stack.
    async function replaceInFiles(): Promise<void> {
        const query = searchedParams.query;
        if (query === '' || searchMatches.length === 0) {
            return;
        }
        const files = searchGroups.length;
        const matches = searchMatches.length;
        const confirmed = await askConfirm(
            'Replace All in Files',
            replaceConfirmMessage(matches, files, searchTruncated),
            'Replace All',
        );
        if (!confirmed) {
            return;
        }

        searchBusy = true;
        searchError = '';
        try {
            const expected = expectedModified(searchMatches);
            const expression = searchExpression(
                query,
                searchedParams.regex,
                searchedParams.caseSensitive,
            );
            // Files open in this window are edited in their Monaco buffers,
            // so tell Go to leave them on disk. They stay unsaved and undoable.
            // openPaths uses the search's spelling so it matches the path Go
            // walks, even when the tab was opened through a symlink.
            const openPaths: string[] = [];
            const openTabs: OpenDocument[] = [];
            const seenTabs = new Set<string>();
            for (const group of searchGroups) {
                const tab = findDocumentByPath(tabs, group.path);
                if (tab !== undefined && tab.kind === 'text' && tab.path !== '') {
                    openPaths.push(group.path);
                    // Two roots can name the same file, and both would map to
                    // one tab; edit its buffer once.
                    if (!seenTabs.has(tab.id)) {
                        seenTabs.add(tab.id);
                        openTabs.push(tab);
                    }
                }
            }
            const result = await ReplaceInProject(windowName, {
                query,
                regex: searchedParams.regex,
                case_sensitive: searchedParams.caseSensitive,
                include: searchedParams.include,
                exclude: searchedParams.exclude,
                replacement: searchReplacement,
                expected_modified: expected,
                open_paths: openPaths,
            });

            // Apply the same edit to each open buffer. A dirty buffer's model
            // may hold unsaved text the search never saw, so replace in the
            // buffer's current text; Monaco keeps the undo history and marks
            // the tab dirty through onDirtyChange.
            for (const tab of openTabs) {
                const current = editorPane?.value(tab.id) ?? tab.content;
                const replaced = replaceText(
                    current,
                    expression,
                    searchReplacement,
                    searchedParams.regex,
                );
                if (replaced === current) {
                    continue;
                }
                const updated = editorPane?.replaceInBuffer(tab.id, replaced);
                if (updated === undefined) {
                    // A tab restored from the session may never have been
                    // displayed, so it has no Monaco model yet. Store the new
                    // text on the tab so the model is created already dirty.
                    tabs = updateDocument(tabs, tab.id, {
                        content: replaced,
                        savedContent: tab.savedContent ?? tab.content,
                        dirty: true,
                    });
                }
            }

            treeRefreshToken++;
            await refreshGitStatus();
            await refreshDiffTabs();
            notice = replaceNotice(result.matches, result.files, result.skipped ?? [], result.failed ?? []);
            await runSearch();
        } catch (error) {
            searchError = errorMessage(error);
        } finally {
            searchBusy = false;
        }
    }

    // openSearchMatch switches to the file, then puts the caret on the hit.
    // The model is created by the tab effect, so the reveal waits a tick.
    async function openSearchMatch(path: string, line: number, column: number): Promise<void> {
        const id = await openFile(path);
        if (id === '') {
            return;
        }
        searchActive = false;
        await tick();
        editorPane?.revealLine(id, line, column);
    }

    // stepFindResult opens the next or previous Find in Files hit. The index
    // advances before the file is opened, so holding F4 cannot open the same
    // hit twice while the first read is still in flight.
    async function stepFindResult(direction: 1 | -1): Promise<void> {
        if (searchMatches.length === 0) {
            notice = 'No search results to step through.';
            return;
        }
        const index = stepSearchResult(searchResultIndex, searchMatches.length, direction);
        searchResultIndex = index;
        const match = searchMatches[index];
        if (match === undefined) {
            return;
        }
        await openSearchMatch(match.path, match.line, match.column);
    }

    // revealPath shows one file in the platform's file manager.
    async function revealPath(path: string): Promise<void> {
        try {
            await RevealFile(path);
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function revealActiveFile(): Promise<void> {
        // A diff tab has no path of its own, but it names the file it was made
        // from, which is the file worth revealing.
        const path = active?.path || active?.sourcePath || '';
        if (path === '') {
            notice = 'Open a saved file to reveal it in Finder.';
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
        const tab = active;
        // A diff tab has no path of its own, but it names the file it was made
        // from, which is the path worth copying.
        const path = tab?.path || tab?.sourcePath || '';
        if (tab === undefined || path === '') {
            notice = 'Open a saved file to copy its path.';
            return;
        }
        const text =
            kind === 'file'
                ? path
                : kind === 'relative'
                  ? relativePath(path, folders)
                  : tab.path === ''
                    ? entryName(path)
                    : tab.name;
        try {
            await CopyText(text);
            notice = `Copied ${text}`;
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // restoreSession reopens the files the previous run left open. Missing
    // files are skipped, so one deleted file does not stop the rest.
    async function restoreSession(session: SavedWindowSession): Promise<void> {
        for (const path of session.openFiles) {
            if (findDocumentByPath(tabs, path) !== undefined) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            showDocument(opened, false);
        }
        const restoredActive = findDocumentByPath(tabs, session.active);
        if (restoredActive !== undefined) {
            activeId = restoredActive.id;
        }
    }

    // persistSession writes the current window state immediately. The live
    // effect below covers ordinary tab changes; workspace switching and closing
    // call this so the old set is on disk before the tab list is replaced.
    async function persistSession(): Promise<void> {
        if (windowName === '') {
            return;
        }
        try {
            await saveSession(windowName, {
                workspace: activeWorkspace?.path ?? '',
                folders: [...folders],
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
        if (preview !== null && preview.id !== activeId) {
            preview = null;
        }
    });

    // The watcher only needs paths that exist: open files and expanded folders.
    // It is registered per window, so a change reaches the window that asked
    // for it and closing the window stops its events.
    $effect(() => {
        if (windowName === '') {
            return;
        }
        const paths = watchedPaths;
        void watchPaths(windowName, paths).catch((error) =>
            console.error('Failed to watch the workspace.', error),
        );
    });

    // The session is written whenever the file-backed tabs, active file,
    // workspace binding or expanded set changes. Gating on sessionReady keeps
    // startup from overwriting the saved session with the empty list the app
    // begins with. A non-owner window parks its workspace set but never
    // replaces the launch window's tabs.
    $effect(() => {
        if (!sessionReady || sessionSuspended || windowName === '') {
            return;
        }
        const files = sessionFiles;
        const activePath = sessionActive;
        const workspacePath = activeWorkspace?.path ?? '';
        const currentFolders = folders;
        const expanded = expandedPaths;
        if (!sessionOwner && workspacePath === '') {
            return;
        }
        void saveSession(windowName, {
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
        if (windowName === '') {
            return;
        }
        void SetUnsavedChanges(windowName, unsaved).catch((error) =>
            console.error('Failed to report unsaved changes.', error),
        );
    });

    onMount(() => {
        let unsubscribe: (() => void) | undefined;
        let unsubscribeMenu: (() => void) | undefined;
        let unsubscribeClose: (() => void) | undefined;
        let unsubscribeQuickOpen: (() => void) | undefined;
        let unsubscribeCommandPalette: (() => void) | undefined;
        let unsubscribeMenuCommand: (() => void) | undefined;
        let unsubscribeFileCommands: (() => void) | undefined;
        let unsubscribeCloseRequest: (() => void) | undefined;
        let unsubscribeEditorCommands: (() => void) | undefined;
        let unsubscribeWatched: (() => void) | undefined;
        let unsubscribeWorkspace: (() => void) | undefined;
        let unsubscribeTabs: (() => void) | undefined;
        let unsubscribeConfig: (() => void) | undefined;
        let unsubscribeAppearance: (() => void) | undefined;
        let unsubscribeSettings: (() => void) | undefined;
        let systemTheme: MediaQueryList | undefined;
        let disposed = false;

        // System mode follows the OS appearance without a restart.
        const onSystemTheme = (): void => {
            if (themeSettings.mode === 'system') {
                void applyCurrentTheme();
            }
        };

        void (async () => {
            // The window name is the identity every per-window binding and
            // event uses, so it is resolved before anything else happens.
            const name = await resolveWindowName();
            if (disposed) {
                return;
            }
            windowName = name;
            setWindowName(name);
            appName = await GetProductName();

            // The request this window was created with decides two things: the
            // paths or buffers to open, and whether this is the window that
            // restores and writes the shared session. Only the session flag is
            // read now; the request itself is collected once its listener is
            // installed, so nothing queued meanwhile can be lost.
            sessionOwner = await ShouldRestoreSession(windowName);

            const workspace = await loadWorkspaceState();
            sidebarVisible = workspace.sidebarVisible;
            sidebarAutoReveal = workspace.autoReveal;
            recentFiles = workspace.recentFiles;
            folders = await loadSidebarFolders(windowName);
            // Git status may walk a large repository; the session restore does
            // not wait for it, and the branch appears when the read lands.
            void refreshGitStatus();
            if (sessionOwner) {
                const saved = await loadWindowSession(windowName);
                if (saved.folders.length > 0) {
                    try {
                        folders = await setSidebarFolders(windowName, saved.folders);
                    } catch (error) {
                        console.error('Failed to restore the sidebar folders.', error);
                    }
                }
                if (saved.workspace !== '') {
                    try {
                        const info = await loadWorkspace(saved.workspace);
                        activeWorkspace = info;
                        folders = await setSidebarFolders(windowName, info.folders ?? []);
                        void refreshGitStatus();
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
            editorSettings = loaded;

            const preferences = await loadPreferences();
            if (disposed) {
                return;
            }
            themeSettings = preferences.theme;
            uiSettings = preferences.ui;
            filesExclude = preferences.excludes.files;
            searchExclude = preferences.excludes.search;
            applyUISettings(uiSettings);
            await refreshThemes();
            await applyCurrentTheme();
            void SetAppearanceMode(themeSettings.mode).catch((error) =>
                console.error('Failed to update the Appearance menu.', error),
            );

            const off = await listenForLaunch(windowName, (request) => void applyLaunch(request));
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
            unsubscribeMenuCommand = listenForMenuCommand(selectCommand);
            unsubscribeFileCommands = listenForFileCommands({
                'new-file': () => void newFileFromMenu(),
                'new-folder': () => void newFolderFromMenu(),
                'new-buffer': newFile,
                'open-file': () => void openFromDialog(),
                'save': () => saveActive(),
                'save-as': () => saveActive(true),
                'save-all': () => void saveAll(),
                'print': () => void printActive(),
            });
            unsubscribeCloseRequest = listenForCloseRequest((reason) => void confirmClose(reason));
            unsubscribeEditorCommands = listenForEditorCommands({
                transformSelection,
                trimWhitespace,
                selectionCommand: runSelectionCommand,
                zoom,
                togglePreview,
                setViewOption,
                find: () => runEditorAction('actions.find'),
                replace: () => runEditorAction('editor.action.startFindReplaceAction'),
                goToLine: () => runEditorAction('editor.action.gotoLine'),
                nextFindResult: () => void stepFindResult(1),
                previousFindResult: () => void stepFindResult(-1),
            });
            unsubscribeWatched = listenForWatchedChanges(
                (paths) => void handleWatchedChanges(paths),
            );
            unsubscribeWorkspace = listenForWorkspaceCommands({
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
            });
            unsubscribeTabs = listenForTabCommands({
                next: () => stepTab(1),
                previous: () => stepTab(-1),
                reopen: () => void reopenClosedTab(),
                goToTab,
                moveToNewWindow: () => void moveTabToNewWindow(),
            });
            unsubscribeConfig = listenForConfigChanges((change) => {
                if (change.themes) {
                    void (async () => {
                        await refreshThemes();
                        await applyCurrentTheme();
                    })();
                }
                if (change.keybindings) {
                    void refreshKeybindings();
                }
            });
            unsubscribeAppearance = listenForAppearance((mode) =>
                void changeThemeSettings({ ...themeSettings, mode }),
            );
            unsubscribeSettings = listenForSettingsOverlay(showSettingsOverlay);
            installEditorKeybindings(runKeybindingCommand);
            await refreshKeybindings();
            void WatchConfig();

            systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
            systemTheme.addEventListener('change', onSystemTheme);
        })();

        return () => {
            disposed = true;
            unsubscribe?.();
            unsubscribeMenu?.();
            unsubscribeClose?.();
            unsubscribeQuickOpen?.();
            unsubscribeCommandPalette?.();
            unsubscribeMenuCommand?.();
            unsubscribeFileCommands?.();
            unsubscribeCloseRequest?.();
            unsubscribeEditorCommands?.();
            unsubscribeWatched?.();
            unsubscribeWorkspace?.();
            unsubscribeTabs?.();
            unsubscribeConfig?.();
            unsubscribeAppearance?.();
            unsubscribeSettings?.();
            systemTheme?.removeEventListener('change', onSystemTheme);
            disposeKeybindings?.();
        };
    });
</script>

<div class="app" data-file-drop-target>
    <TitleBar {appName} filePath={shownPath(active)} folderLabel={folderLabel} />
    <div class="app__body" class:app__body--sidebar-right={uiSettings.sidebarPosition === 'right'}>
        <div class="app__sidebar" class:app__sidebar--hidden={!sidebarVisible}>
            {#if sidebarView === 'changes'}
                <ChangesView
                    changes={gitChanges}
                    {busy}
                    onOpenDiff={(path) => void openDiff(path)}
                    onRefresh={() => void refreshWorkspace()}
                />
            {:else}
                <Sidebar
                    {folders}
                    selectedPath={revealedPath}
                    {busy}
                    refreshToken={treeRefreshToken}
                    {busyPath}
                    statuses={gitStatuses}
                    iconStyle={uiSettings.iconStyle}
                    onAddFolder={() => void addFolderFromDialog()}
                    onRemoveFolder={(path) => void removeFolder(path)}
                    onOpenFile={(path) => void openFile(path)}
                    onContextMenu={showContextMenu}
                    onEmptyAreaMenu={showEmptyAreaMenu}
                    onExpandedChange={setExpanded}
                    onMovePath={(source, folder) => void moveEntry(source, folder)}
                />
            {/if}
        </div>
        <div class="app__editor">
            <TabBar
                {tabs}
                activeId={activeId}
                showIcons={uiSettings.tabIcons}
                iconStyle={uiSettings.iconStyle}
                searchOpen={searchOpen}
                searchActive={searchActive}
                onSelect={(id) => {
                    activeId = id;
                    searchActive = false;
                }}
                onClose={(id) => void closeTab(id)}
                onContextMenu={(event, tab) => showTabContextMenu(event, tab)}
                onReorder={reorderTab}
                onSelectSearch={() => {
                    searchOpen = true;
                    searchActive = true;
                    searchFocusToken++;
                }}
                onCloseSearch={closeSearchTab}
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
                        settings={editorSettings}
                        readOnly={active?.kind === 'diff'}
                        onRevealFile={() => void revealActiveFile()}
                        onCopyPath={(kind) => void copyActivePath(kind)}
                        onDirtyChange={(id, dirty) => {
                            tabs = setDocumentDirty(tabs, id, dirty);
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
                {#if searchActive}
                    <SearchPanel
                        bind:query={searchQuery}
                        bind:include={searchInclude}
                        bind:exclude={searchExcludeGlobs}
                        bind:replacement={searchReplacement}
                        bind:regex={searchRegex}
                        bind:caseSensitive={searchCaseSensitive}
                        groups={searchGroups}
                        summary={searchLabel}
                        error={searchError}
                        busy={searchBusy}
                        focusToken={searchFocusToken}
                        roots={folders}
                        onSearch={() => void runSearch()}
                        onReplaceAll={() => void replaceInFiles()}
                        onOpenMatch={(path, line, column) =>
                            void openSearchMatch(path, line, column)}
                    />
                {:else if active?.kind === 'image'}
                    {#key active.src}
                        <ImageViewer
                            src={active.src}
                            name={active.name}
                            onSize={(width, height) => {
                                imageSize = { id: activeId, width, height };
                            }}
                        />
                    {/key}
                {:else if previewActive && preview !== null}
                    <Preview
                        kind={preview.kind}
                        content={preview.content}
                        name={active?.name ?? ''}
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
        eol={active?.kind === 'text' ? active.eol : ''}
        indentation={indentationText}
        invalidUTF8={active?.kind === 'text' && active.invalidUTF8}
        {notice}
        {folderLabel}
        imageSize={imageSizeLabel}
        branch={branch}
        sidebarView={sidebarView}
        changesCount={gitChanges.length}
        onSwitchView={switchSidebarView}
        onToggleEol={toggleLineEnding}
        onPickIndentation={active?.kind === 'text' ? showIndentationPalette : undefined}
        onChangeLanguage={active?.kind === 'text' ? showLanguagePalette : undefined}
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
                void openFile(path);
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
    {#if confirmPrompt !== null}
        <Prompt
            title={confirmPrompt.title}
            message={confirmPrompt.message}
            actions={confirmPrompt.actions}
            onSelect={answerConfirm}
            onCancel={() => answerConfirm('cancel')}
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
    {#if namePrompt !== null}
        <NamePrompt
            title={namePrompt.title}
            label={namePrompt.label}
            bind:value={namePrompt.value}
            selectStem={namePrompt.selectStem}
            selectFolder={namePrompt.selectFolder}
            error={namePrompt.error}
            onSubmit={submitName}
            onCancel={cancelName}
        />
    {/if}
    {#if settingsOverlay}
        <SettingsOverlay
            editor={editorSettings}
            theme={themeSettings}
            ui={uiSettings}
            {themes}
            filesExclude={filesExclude}
            searchExclude={searchExclude}
            {sidebarVisible}
            autoReveal={sidebarAutoReveal}
            warning={settingsWarning}
            onEditor={changeEditorSettings}
            onTheme={(next) => void changeThemeSettings(next)}
            onUI={(next) => void changeUISettings(next)}
            onFilesExclude={(next) => void changeFilesExclude(next)}
            onSearchExclude={(next) => void changeSearchExclude(next)}
            onSidebarVisible={setSidebarVisible}
            onAutoReveal={setSidebarAutoReveal}
            onOpenSettings={() => void openSettings()}
            onOpenDefaults={() => void openDefaultSettings()}
            onOpenThemes={() => void openThemesFolder()}
            onOpenKeybindings={() => void openKeybindings()}
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
