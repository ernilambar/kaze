<script lang="ts">
    import * as monaco from 'monaco-editor';
    import { onMount, tick } from 'svelte';
    import {
        CopyText,
        GetKeybindings,
        InstallCommandLineTool,
        OpenDefaultSettings,
        OpenFileDialog,
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
        WatchConfig,
    } from '../bindings/kova/app';
    import type { SaveResult, SearchMatch } from '../bindings/kova/models';
    import { CASE_TRANSFORMS, type CaseTransform } from './case';
    import ChangesView from './ChangesView.svelte';
    import EditorPane from './EditorPane.svelte';
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
    import { COMMANDS, listenForCommandPalette, rankCommands } from './commands';
    import {
        createDiffDocument,
        createUntitledDocument,
        diffTabId,
        listenForLaunch,
        readDocument,
        shownName,
        type LaunchRequest,
        type OpenDocument,
    } from './document';
    import { languageLabel, WELCOME_KEY } from './editor';
    import {
        listenForEditorCommands,
        VIEW_OPTIONS,
        type ViewOption,
        type ZoomCommand,
    } from './editorcommands';
    import { listenForCloseRequest, listenForFileCommands } from './filecommands';
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
    import { groupSearchMatches, searchSummary } from './search';
    import {
        defaultEditorSettings,
        defaultThemeSettings,
        defaultUISettings,
        loadEditorSettings,
        loadPreferences,
        saveEditorSettings,
        saveThemeSettings,
        saveUISettings,
        type EditorSettings,
        type ThemeSettings,
        type UISettings,
    } from './settings';
    import {
        installKeybindings,
        isUsableBinding,
        type KeyContext,
    } from './keybindings';
    import {
        applyThemeCSS,
        applyUISettings,
        listThemes,
        listenForAppearance,
        listenForConfigChanges,
        listenForSettingsOverlay,
        loadTheme,
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
        sidebarLabel,
    } from './sidebar';
    import { noticeFor } from './status';
    import {
        closeDocumentTab,
        listenForCloseTab,
        listenForTabCommands,
        openDocumentTab,
        setDocumentDirty,
        stepTabId,
        tabIdAt,
        updateDocument,
    } from './tabs';
    import { appendFinalNewline } from './text';
    import {
        clearRecentFiles,
        listenForWatchedChanges,
        loadWorkspaceState,
        rememberRecentFile,
        saveSession,
        saveSidebarVisible,
        watchPaths,
        type WorkspaceState,
    } from './workspace';
    import { listenForWorkspaceCommands, type CopyPathKind } from './workspacecommands';

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

    const APP_NAME = 'Kova';

    let tabs = $state<OpenDocument[]>([]);
    let activeId = $state('');
    let folders = $state<string[]>([]);
    let notice = $state('');
    let busy = $state(false);

    // Git is read-only. The sidebar shows the status marks from gitChanges,
    // while the status bar shows the active file's branch.
    let sidebarView = $state<SidebarView>('explorer');
    let gitRepos = $state<GitRepo[]>([]);
    let gitChanges = $state<GitChange[]>([]);

    let editorSettings = $state<EditorSettings>({ ...defaultEditorSettings });
    let themeSettings = $state<ThemeSettings>({ ...defaultThemeSettings });
    let uiSettings = $state<UISettings>({ ...defaultUISettings });
    let themes = $state<ThemeInfo[]>([]);
    let themeErrors = $state<string[]>([]);
    let settingsOverlay = $state(false);
    let keybindingError = $state('');
    let disposeKeybindings: (() => void) | undefined;
    let preview = $state<PreviewState | null>(null);

    let quickOpen = $state(false);
    let quickOpenFiles = $state<QuickOpenFile[]>([]);
    let quickOpenLoading = $state(false);
    let commandPalette = $state(false);
    let recentOpen = $state(false);

    // Workspace state that outlives a window.
    let recentFiles = $state<string[]>([]);
    let sidebarVisible = $state(true);
    let expandedPaths = $state<string[]>([]);
    let treeRefreshToken = $state(0);
    let sessionReady = $state(false);
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
    let searchReplacement = $state('');
    let searchRegex = $state(false);
    let searchCaseSensitive = $state(false);
    let searchMatches = $state<SearchMatch[]>([]);
    let searchTruncated = $state(false);
    let searchBusy = $state(false);
    let searchError = $state('');
    let searchFocusToken = $state(0);

    let unsavedPrompt = $state<UnsavedPrompt | null>(null);
    let conflictPrompt = $state<ConflictPrompt | null>(null);

    let line = $state(1);
    let column = $state(1);

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
              : languageLabel(active?.path ?? ''),
    );
    // Monaco keeps its buffers while an image tab is showing, so switching back
    // to a text tab preserves edits. The idle buffer keeps the hidden editor
    // from creating a model for an image path.
    let editorKey = $derived(imageActive ? WELCOME_KEY : (active?.id ?? WELCOME_KEY));
    let editorContent = $derived(imageActive ? '' : (active?.content ?? ''));
    let openIds = $derived(tabs.map((tab) => tab.id));
    let unsaved = $derived(tabs.some((tab) => tab.dirty));
    let folderLabel = $derived(sidebarLabel(folders));
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
                latest = await addSidebarFolder(path);
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

    // refreshGitStatus rereads the repositories behind the sidebar roots. It
    // runs when the workspace changes and after Kova writes a file; a folder
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
            editorPane?.reload(id, document.content);
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
        const previousPath = findTab(id)?.path ?? '';
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
        // Save As can give an untitled buffer a real path, and with it a
        // language the model should start highlighting.
        editorPane?.setLanguage(id, result.path);
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
        // Pictures are read-only in Kova, and a diff is a generated view, so
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
        rememberClosedTab(tab);
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

    // Theme application: CSS variables cover the app chrome; Monaco gets its
    // own theme definition. A stable name per palette lets a live reload
    // replace the data and re-select the same theme.
    function applyThemePalette(palette: ThemePalette): void {
        applyThemeCSS(palette);
        const name = `kova-${palette.name.replace(/[^A-Za-z0-9_-]/g, '-')}`;
        monaco.editor.defineTheme(name, themeData(palette));
        monaco.editor.setTheme(name);
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
        const fallback = prefersDark ? 'Kova Dark' : 'Kova Light';
        try {
            applyThemePalette(await loadTheme(fallback));
            notice = `Theme "${name}" is unavailable.`;
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    // refreshThemes lists installed themes and repairs a name that no longer
    // exists. The first theme for an appearance is the fallback, so deleting a
    // theme file does not leave Kova stuck on the old palette.
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

    function showSettingsOverlay(): void {
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
        // A broken file did not decode; keep the previous listener and native
        // menu rather than silently dropping every shortcut. A file that
        // decoded has an authoritative list even when entries were rejected.
        if (!state.parsed) {
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
        disposeKeybindings = installKeybindings(usable, {
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

    function zoom(command: ZoomCommand): void {
        editorPane?.zoom(command);
    }

    // setViewOption applies a view toggle, saves it, and moves the View menu's
    // checkmark when the toggle came from the command palette instead.
    function setViewOption(option: ViewOption, checked: boolean): void {
        if (editorSettings[option] === checked) {
            return;
        }
        editorSettings = { ...editorSettings, [option]: checked };
        void saveEditorSettings(editorSettings)
            .then((saved) => {
                editorSettings = saved;
            })
            .catch((error) => {
                notice = errorMessage(error);
            });
        void SetViewOption(option, checked).catch((error) =>
            console.error('Failed to update the View menu.', error),
        );
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
        recentOpen = false;
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

        if (id.startsWith('go-to-tab-')) {
            goToTab(Number(id.slice('go-to-tab-'.length)));
            return;
        }

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
            case 'trim-whitespace':
                trimWhitespace();
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
            case 'close-all-folders':
                void closeAllFolders();
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
            folders = await addSidebarFolder(path);
            void refreshGitStatus();
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

    function toggleSidebar(): void {
        sidebarVisible = !sidebarVisible;
        void saveSidebarVisible(sidebarVisible).catch((error) =>
            console.error('Failed to save the sidebar state.', error),
        );
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
            folders = await closeAllSidebarFolders();
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
            editorPane?.reload(id, opened.content);
        }
        tabs = updateDocument(tabs, id, {
            kind: opened.kind,
            content: opened.content,
            src: opened.src,
            modified: opened.modified,
            dirty: false,
        });
    }

    async function refreshOpenDocuments(): Promise<void> {
        for (const tab of tabs) {
            if (tab.kind !== 'text' || tab.path === '') {
                continue;
            }
            // External changes never overwrite edits the user has not saved;
            // the next save reports the conflict instead.
            if (tab.dirty) {
                continue;
            }
            const opened = await readDocument(tab.path);
            if (opened === undefined) {
                continue;
            }
            applyReload(tab.id, opened);
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
    // anything else. A dirty buffer stays as it is and says why.
    async function handleWatchedChanges(paths: string[]): Promise<void> {
        let refreshTree = false;
        for (const path of paths) {
            const tab = tabs.find((entry) => entry.kind === 'text' && entry.path === path);
            if (tab === undefined) {
                refreshTree = true;
                continue;
            }
            if (tab.dirty) {
                notice = `${tab.name} changed on disk; your edits were kept.`;
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            applyReload(tab.id, opened);
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
        searchOpen = true;
        searchActive = true;
        searchFocusToken++;
    }

    function closeSearchTab(): void {
        searchOpen = false;
        searchActive = false;
        searchMatches = [];
        searchError = '';
    }

    async function runSearch(): Promise<void> {
        const query = searchQuery.trim();
        if (query === '') {
            searchMatches = [];
            searchError = '';
            return;
        }
        searchBusy = true;
        searchError = '';
        try {
            const result = await SearchProject({
                query,
                regex: searchRegex,
                case_sensitive: searchCaseSensitive,
                include: searchInclude,
                exclude: '',
                replacement: '',
            });
            searchMatches = result.matches ?? [];
            searchTruncated = result.truncated;
        } catch (error) {
            searchMatches = [];
            searchError = errorMessage(error);
        } finally {
            searchBusy = false;
        }
    }

    async function replaceInFiles(): Promise<void> {
        const query = searchQuery.trim();
        if (query === '') {
            return;
        }
        searchBusy = true;
        searchError = '';
        try {
            const result = await ReplaceInProject({
                query,
                regex: searchRegex,
                case_sensitive: searchCaseSensitive,
                include: searchInclude,
                exclude: '',
                replacement: searchReplacement,
            });
            await refreshOpenDocuments();
            treeRefreshToken++;
            await refreshGitStatus();
            await refreshDiffTabs();
            notice = `Replaced ${result.matches} in ${result.files} files.`;
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
        await openFile(path);
        const tab = tabs.find((entry) => entry.path === path);
        if (tab === undefined) {
            return;
        }
        searchActive = false;
        await tick();
        editorPane?.revealLine(tab.id, line, column);
    }

    async function revealActiveFile(): Promise<void> {
        const path = active?.path ?? '';
        if (path === '') {
            notice = 'Open a saved file to reveal it in Finder.';
            return;
        }
        try {
            await RevealFile(path);
            notice = '';
        } catch (error) {
            notice = errorMessage(error);
        }
    }

    async function copyActivePath(kind: CopyPathKind): Promise<void> {
        const tab = active;
        if (tab === undefined || tab.path === '') {
            notice = 'Open a saved file to copy its path.';
            return;
        }
        const text =
            kind === 'file'
                ? tab.path
                : kind === 'relative'
                  ? relativePath(tab.path, folders)
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
    async function restoreSession(session: WorkspaceState['session']): Promise<void> {
        for (const path of session.openFiles) {
            if (tabs.some((tab) => tab.path === path)) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            showDocument(opened, false);
        }
        const restoredActive = tabs.find((tab) => tab.path === session.active);
        if (restoredActive !== undefined) {
            activeId = restoredActive.id;
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
    $effect(() => {
        const paths = watchedPaths;
        void watchPaths(paths).catch((error) =>
            console.error('Failed to watch the workspace.', error),
        );
    });

    // The session is written whenever the set of file-backed tabs or the active
    // file changes. Gating on sessionReady keeps startup from overwriting the
    // saved session with the empty list the app begins with.
    $effect(() => {
        if (!sessionReady) {
            return;
        }
        const files = sessionFiles;
        const activePath = sessionActive;
        void saveSession(files, activePath).catch((error) =>
            console.error('Failed to save the session.', error),
        );
    });

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
            const workspace = await loadWorkspaceState();
            sidebarVisible = workspace.sidebarVisible;
            recentFiles = workspace.recentFiles;
            folders = await loadSidebarFolders();
            // Git status may walk a large repository; the session restore does
            // not wait for it, and the branch appears when the read lands.
            void refreshGitStatus();
            await restoreSession(workspace.session);
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
            applyUISettings(uiSettings);
            await refreshThemes();
            await applyCurrentTheme();
            void SetAppearanceMode(themeSettings.mode).catch((error) =>
                console.error('Failed to update the Appearance menu.', error),
            );

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
            unsubscribeEditorCommands = listenForEditorCommands({
                transformSelection,
                trimWhitespace,
                zoom,
                togglePreview,
                setViewOption,
            });
            unsubscribeWatched = listenForWatchedChanges(
                (paths) => void handleWatchedChanges(paths),
            );
            unsubscribeWorkspace = listenForWorkspaceCommands({
                openRecent: showRecentFiles,
                clearRecent: () => void clearRecent(),
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

<div class="app">
    <TitleBar appName={APP_NAME} fileName={shownName(active)} folderLabel={folderLabel} />
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
                    selectedPath={active?.path ?? ''}
                    {busy}
                    refreshToken={treeRefreshToken}
                    statuses={gitStatuses}
                    iconStyle={uiSettings.iconStyle}
                    onAddFolder={() => void addFolderFromDialog()}
                    onRemoveFolder={(path) => void removeFolder(path)}
                    onOpenFile={(path) => void openFile(path)}
                    onExpandedChange={setExpanded}
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
                        {openIds}
                        settings={editorSettings}
                        readOnly={active?.kind === 'diff'}
                        onDirtyChange={(id, dirty) => {
                            tabs = setDocumentDirty(tabs, id, dirty);
                        }}
                        onCursorMove={(nextLine, nextColumn) => {
                            line = nextLine;
                            column = nextColumn;
                        }}
                    />
                </div>
                {#if searchActive}
                    <SearchPanel
                        bind:query={searchQuery}
                        bind:include={searchInclude}
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
                    <Welcome
                        recent={recentItems.slice(0, 6)}
                        onCommand={selectCommand}
                        onOpenRecent={(path) => void openFile(path)}
                    />
                {/if}
            </div>
        </div>
    </div>
    <StatusBar
        {line}
        {column}
        {language}
        {notice}
        {folderLabel}
        imageSize={imageSizeLabel}
        branch={branch}
        sidebarView={sidebarView}
        changesCount={gitChanges.length}
        onSwitchView={switchSidebarView}
    />
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
    {#if settingsOverlay}
        <SettingsOverlay
            theme={themeSettings}
            ui={uiSettings}
            {themes}
            warning={settingsWarning}
            onTheme={(next) => void changeThemeSettings(next)}
            onUI={(next) => void changeUISettings(next)}
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
