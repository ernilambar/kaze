import type { WorkspaceInfo } from '../../bindings/kaze/internal/workspacefile/models';
import { readDocument, toBufferDocument, type LaunchRequest } from '../document';
import {
    addSidebarFolder,
    closeAllSidebarFolders,
    errorMessage,
    openFolderDialog,
    removeSidebarFolder,
    setSidebarFolders,
} from '../sidebar';
import { noticeFor } from '../status';
import { findDocumentByPath } from '../tabs';
import {
    clearRecentFiles,
    getRecentWorkspaces,
    loadWorkspace,
    openWorkspaceDialog,
    rememberRecentFile,
    saveSession,
    saveWorkspaceAs,
    toWorkspaceSession,
    type SavedWindowSession,
    type SavedWorkspaceSession,
} from '../workspace';
import type { WindowContext } from './context.svelte';
import type { Documents } from './documents.svelte';
import type { GitController } from './git.svelte';
import type { Prompts } from './prompts.svelte';
import type { SearchController } from './search.svelte';
import type { TreeOps } from './treeops.svelte';

// WorkspaceController owns the sidebar roots, the workspace file the window is
// bound to, Open Recent and the session written to session.json.
export class WorkspaceController {
    private readonly ctx: WindowContext;
    private readonly prompts: Prompts;
    private readonly git: GitController;
    private readonly search: SearchController;
    private readonly documents: Documents;
    private readonly treeops: TreeOps;

    // activeWorkspace is the file the roots came from. It is session state: the
    // roots themselves are what settings.json restores.
    activeWorkspace = $state<WorkspaceInfo | null>(null);

    // Open Recent outlives the window and is shared by every window.
    recentFiles = $state<string[]>([]);

    // recentWorkspaces is the workspace files already opened, newest first. It
    // is read from the parked tab sets each time the palette opens, so a switch
    // in another window shows up here too.
    recentWorkspaces = $state<string[]>([]);

    sessionReady = $state(false);

    // Only the window that restored the saved tabs owns session.json's launch
    // entry. Later windows have their own tabs and must not replace it, though
    // they still park their workspace tab sets.
    sessionOwner = $state(false);

    // sessionSuspended stops the live session writer while a workspace switch
    // or close is moving tabs, so the state between the two tab lists is never
    // written to disk. It is reactive because the save effect returns before it
    // subscribes to the tab list when suspended; clearing it must run the
    // effect again.
    sessionSuspended = $state(false);

    // The session records file-backed tabs only: an untitled buffer has no
    // text on disk, and a diff is a view of a file that is already recorded.
    sessionFiles = $derived.by(() =>
        this.ctx.tabs.filter((tab) => tab.kind === 'text' && tab.path !== '').map((tab) => tab.path),
    );

    // A diff tab records the file it shows, so a relaunch restores the file
    // itself as the active tab.
    sessionActive = $derived.by(() => this.ctx.active?.path || this.ctx.active?.sourcePath || '');

    // addFolders saves each root from a launch request and adopts the result.
    // The backend deduplicates, so relaunching with the same folder is a no-op.
    // The first failure is returned so the caller can show it in the status bar.
    addFolders = async (paths: string[]): Promise<string> => {
        let failure = '';
        let latest = this.ctx.folders;
        for (const path of paths) {
            try {
                latest = await addSidebarFolder(this.ctx.windowName, path);
            } catch (error) {
                if (failure === '') {
                    failure = errorMessage(error);
                }
            }
        }
        this.ctx.folders = latest;
        void this.git.refreshGitStatus();
        return failure;
    };

    applyLaunch = async (request: LaunchRequest): Promise<void> => {
        let failure = '';
        if (request.replace_folders) {
            // Open Folder in New Window: the new window starts with exactly the
            // chosen folder, and the roots are saved for the next launch.
            try {
                this.ctx.folders = await setSidebarFolders(this.ctx.windowName, request.folders);
                void this.git.refreshGitStatus();
            } catch (error) {
                failure = errorMessage(error);
            }
        } else if (request.folders.length > 0) {
            failure = await this.addFolders(request.folders);
        }

        // A tab moved from another window arrives as a buffer. Text keeps the
        // text the other window held, so unsaved edits survive the move; an
        // image or diff is read again from its path.
        for (const buffer of request.buffers) {
            if (buffer.kind === 'image' && buffer.path !== '') {
                await this.documents.openFile(buffer.path);
                continue;
            }
            if (buffer.kind === 'diff' && buffer.source_path !== '') {
                await this.git.openDiff(buffer.source_path);
                continue;
            }
            this.documents.showDocument(toBufferDocument(buffer), false);
        }

        for (const path of request.files) {
            const existing = findDocumentByPath(this.ctx.tabs, path);
            if (existing !== undefined) {
                this.ctx.activeId = existing.id;
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                if (failure === '') {
                    failure = `Cannot open ${path}`;
                }
                continue;
            }
            this.documents.showDocument(opened);
        }

        this.ctx.notice = failure !== '' ? failure : noticeFor(request);
    };

    addFolderFromDialog = async (): Promise<void> => {
        if (this.ctx.busy) {
            return;
        }
        this.ctx.busy = true;
        try {
            const path = await openFolderDialog();
            // Cancelling the dialog is a decision, not a failure.
            if (path === '') {
                return;
            }
            this.ctx.folders = await addSidebarFolder(this.ctx.windowName, path);
            void this.git.refreshGitStatus();
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.ctx.busy = false;
        }
    };

    // Removing a root only drops it from the sidebar: tabs stay open, because
    // they hold their own buffers and are closed from the tab bar.
    removeFolder = async (path: string): Promise<void> => {
        try {
            this.ctx.folders = await removeSidebarFolder(this.ctx.windowName, path);
            // A root added back later is new, so it starts open.
            this.treeops.restoredExpansionRoots = this.treeops.restoredExpansionRoots.filter(
                (root) => root !== path,
            );
            if (this.ctx.folders.length === 0) {
                this.activeWorkspace = null;
            }
            void this.git.refreshGitStatus();
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // restoreWorkspaceTabs opens the file-backed tabs parked under a workspace.
    // Missing files are skipped, so one deleted file does not stop the rest.
    restoreWorkspaceTabs = async (session: SavedWorkspaceSession): Promise<void> => {
        for (const path of session.openFiles) {
            if (findDocumentByPath(this.ctx.tabs, path) !== undefined) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            this.documents.showDocument(opened, false);
        }
        const restoredActive = findDocumentByPath(this.ctx.tabs, session.active);
        if (restoredActive !== undefined) {
            this.ctx.activeId = restoredActive.id;
        }
    };

    // clearWorkspaceTabs drops every document tab and the Find Results tab.
    // The old set has already been parked in session.json, and the Reopen
    // Closed Tab stack is cleared so it cannot pull tabs from another
    // workspace back into this one.
    clearWorkspaceTabs = (): void => {
        this.ctx.tabs = [];
        this.ctx.activeId = '';
        this.documents.closedTabs = [];
        if (this.search.open) {
            this.search.closeSearchTab();
        }
    };

    // prepareWorkspaceSwitch is the one prompt before a switch replaces the
    // tab set. Saving all is the default because the parked tab set is a
    // convenience, not a place to discard edits.
    prepareWorkspaceSwitch = async (title: string, message: string): Promise<boolean> => {
        const dirty = this.ctx.tabs.filter((tab) => tab.dirty);
        if (dirty.length === 0) {
            return true;
        }
        const answer = await this.prompts.askUnsaved({
            title,
            message,
            details: dirty.map((tab) => tab.path || tab.name),
            confirmLabel: 'Save All and Continue',
        });
        if (answer === 'cancel') {
            return false;
        }
        if (answer === 'save') {
            return await this.documents.saveAll();
        }
        return true;
    };

    // switchWorkspace parks the current tabs, replaces the roots and restores
    // the target workspace's parked tabs. The two tab lists never coexist in
    // the live session writer.
    switchWorkspace = async (target: WorkspaceInfo): Promise<boolean> => {
        await this.persistSession();
        if (!(await this.prepareWorkspaceSwitch(
            'Switch Workspace',
            'Switching workspaces closes the open tabs. Save them first?',
        ))) {
            return false;
        }

        this.sessionSuspended = true;
        let switched = false;
        try {
            this.ctx.folders = await setSidebarFolders(this.ctx.windowName, target.folders ?? []);
            this.activeWorkspace = target;
            const parked = toWorkspaceSession(target.session);
            this.treeops.restoreExpanded(parked.expanded, parked.folders);
            this.treeops.sidebarWidth = parked.sidebarWidth;
            this.clearWorkspaceTabs();
            await this.restoreWorkspaceTabs(parked);
            switched = true;
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.sessionSuspended = false;
        }
        if (switched) {
            void this.git.refreshGitStatus();
            await this.persistSession();
        }
        return switched;
    };

    // openWorkspaceFromDialog replaces the window's workspace with the picked
    // file. The previous tab set stays parked under its own workspace.
    openWorkspaceFromDialog = async (): Promise<void> => {
        if (this.ctx.busy || this.prompts.unsaved !== null || this.prompts.confirm !== null) {
            return;
        }
        this.ctx.busy = true;
        try {
            const workspace = await openWorkspaceDialog();
            if (workspace.path === '') {
                return;
            }
            await this.switchWorkspace(workspace);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.ctx.busy = false;
        }
    };

    // loadRecentWorkspaces refreshes the list the Open Recent Workspace palette
    // shows. The read is cheap and every call sees other windows' switches.
    loadRecentWorkspaces = async (): Promise<void> => {
        try {
            this.recentWorkspaces = await getRecentWorkspaces();
        } catch (error) {
            this.recentWorkspaces = [];
            this.ctx.notice = errorMessage(error);
        }
    };

    // switchWorkspaceTo reopens a workspace by path, without a file picker.
    switchWorkspaceTo = async (path: string): Promise<void> => {
        if (this.ctx.busy || this.prompts.unsaved !== null || this.prompts.confirm !== null) {
            return;
        }
        this.ctx.busy = true;
        try {
            const workspace = await loadWorkspace(path);
            if (workspace.path === '') {
                return;
            }
            await this.switchWorkspace(workspace);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.ctx.busy = false;
        }
    };

    // saveWorkspaceAsDialog writes the current roots to a workspace file, binds
    // the window to it and parks the current tabs under the new name.
    saveWorkspaceAsDialog = async (): Promise<void> => {
        if (this.ctx.busy || this.prompts.unsaved !== null || this.prompts.confirm !== null) {
            return;
        }
        this.ctx.busy = true;
        try {
            const workspace = await saveWorkspaceAs(this.ctx.folders);
            if (workspace.path === '') {
                return;
            }
            this.activeWorkspace = workspace;
            await this.persistSession();
            this.ctx.notice = `Saved workspace ${workspace.name}`;
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.ctx.busy = false;
        }
    };

    // closeWorkspace clears the roots and closes the tabs after the save
    // prompt. The parked tab set stays in session.json, so reopening the
    // workspace later restores it.
    closeWorkspace = async (): Promise<void> => {
        if (this.ctx.busy || this.prompts.unsaved !== null || this.prompts.confirm !== null) {
            return;
        }
        if (this.activeWorkspace === null && this.ctx.folders.length === 0) {
            return;
        }
        this.ctx.busy = true;
        try {
            await this.persistSession();
            if (!(await this.prepareWorkspaceSwitch(
                'Close Workspace',
                'Closing the workspace closes its tabs. Save them first?',
            ))) {
                return;
            }

            this.sessionSuspended = true;
            try {
                this.ctx.folders = await closeAllSidebarFolders(this.ctx.windowName);
                this.activeWorkspace = null;
                this.treeops.restoreExpanded([], []);
                this.treeops.sidebarWidth = 0;
                this.clearWorkspaceTabs();
                this.ctx.notice = '';
            } finally {
                this.sessionSuspended = false;
            }
            void this.git.refreshGitStatus();
            await this.persistSession();
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.ctx.busy = false;
        }
    };

    closeAllFolders = async (): Promise<void> => {
        try {
            this.ctx.folders = await closeAllSidebarFolders(this.ctx.windowName);
            this.activeWorkspace = null;
            this.treeops.restoreExpanded([], []);
            void this.git.refreshGitStatus();
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // recordRecent puts an opened file at the top of Open Recent.
    recordRecent = async (path: string): Promise<void> => {
        try {
            this.recentFiles = await rememberRecentFile(path);
        } catch (error) {
            console.error('Failed to remember the recent file.', error);
        }
    };

    clearRecent = async (): Promise<void> => {
        try {
            this.recentFiles = await clearRecentFiles();
            this.ctx.notice = 'Recent files cleared.';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    refreshWorkspace = async (): Promise<void> => {
        this.treeops.treeRefreshToken++;
        await this.documents.refreshOpenDocuments();
        await this.git.refreshDiffTabs();
        await this.git.refreshGitStatus();
        // HEAD may have moved since the buffers were opened, so the gutter
        // baselines are reread too.
        this.git.reloadBaselines();
        this.ctx.notice = '';
    };

    // restoreSession reopens the files the previous run left open. Missing
    // files are skipped, so one deleted file does not stop the rest.
    restoreSession = async (session: SavedWindowSession): Promise<void> => {
        for (const path of session.openFiles) {
            if (findDocumentByPath(this.ctx.tabs, path) !== undefined) {
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                continue;
            }
            this.documents.showDocument(opened, false);
        }
        const restoredActive = findDocumentByPath(this.ctx.tabs, session.active);
        if (restoredActive !== undefined) {
            this.ctx.activeId = restoredActive.id;
        }
    };

    // persistSession writes the current window state immediately. The live
    // effect below covers ordinary tab changes; workspace switching and closing
    // call this so the old set is on disk before the tab list is replaced.
    persistSession = async (): Promise<void> => {
        if (this.ctx.windowName === '') {
            return;
        }
        try {
            await saveSession(this.ctx.windowName, {
                workspace: this.activeWorkspace?.path ?? '',
                folders: [...this.ctx.folders],
                openFiles: [...this.sessionFiles],
                active: this.sessionActive,
                expanded: [...this.treeops.expandedPaths],
                sidebarWidth: this.treeops.sidebarWidth,
            });
        } catch (error) {
            console.error('Failed to save the session.', error);
        }
    };

    constructor(ctx: WindowContext, prompts: Prompts, git: GitController, search: SearchController, documents: Documents, treeops: TreeOps) {
        this.ctx = ctx;
        this.prompts = prompts;
        this.git = git;
        this.search = search;
        this.documents = documents;
        this.treeops = treeops;
    }
}
