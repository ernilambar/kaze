import {
    branchLabel,
    changedFolders,
    loadGitStatus,
    readGitBaseline,
    readGitDiff,
    repoForPath,
    type GitChange,
    type GitRepo,
    type GitStatusKind,
} from '../git';
import { createDiffDocument, diffTabId, type OpenDocument } from '../document';
import { errorMessage } from '../sidebar';
import { updateDocument } from '../tabs';
import type { WindowContext } from './context.svelte';

export interface GitHost {
    showDocument: (document: OpenDocument, record?: boolean) => void;
}

// GitController is read-only git state. The sidebar shows the status marks
// from changes, while the status bar shows the active file's branch.
export class GitController {
    private readonly ctx: WindowContext;
    private readonly host: GitHost;

    repos = $state<GitRepo[]>([]);
    changes = $state<GitChange[]>([]);

    // The tree looks statuses up by path; the changes list keeps the ordered
    // records. Both come from the same snapshot.
    statuses = $derived.by(() => {
        const statuses: Record<string, GitStatusKind> = {};
        for (const change of this.changes) {
            statuses[change.path] = change.status;
        }
        return statuses;
    });

    // folderChanges maps folders with a visible change beneath them to its most
    // significant status, so a folder row marks itself even when it is
    // collapsed and takes the matching colour. Deletions stay in the changes
    // list, because they have no tree row to reveal.
    folderChanges = $derived.by(() => changedFolders(this.changes, this.ctx.folders));

    // The arrow defers the read of ctx until the constructor has set it.
    activeRepo = $derived.by(() =>
        repoForPath(this.repos, this.ctx.active?.path || this.ctx.active?.sourcePath || ''),
    );

    branch = $derived(branchLabel(this.activeRepo));

    // refreshGitStatus rereads the repositories behind the sidebar roots. It
    // runs when the workspace changes and after the app writes a file; a folder
    // outside git simply contributes nothing. A newer request wins, so adding
    // a root while an older read is still running cannot roll the result back.
    // request is plain: it is never rendered.
    request = 0;

    refreshGitStatus = async (): Promise<void> => {
        const request = ++this.request;
        try {
            const state = await loadGitStatus(this.ctx.folders);
            if (request !== this.request) {
                return;
            }
            this.repos = state.repos;
            this.changes = state.changes;
        } catch (error) {
            console.error('Failed to read git status.', error);
        }
    };

    // loadBaseline gives one buffer its HEAD text so the gutter can mark the
    // lines that differ while the user types. An untracked file's baseline is
    // empty, which makes every line read as added.
    loadBaseline = async (id: string, path: string): Promise<void> => {
        if (path === '') {
            return;
        }
        try {
            const baseline = await readGitBaseline(path);
            if (!baseline.inRepo || baseline.ignored) {
                this.ctx.editorPane?.clearBaseline(id);
                return;
            }
            this.ctx.editorPane?.setBaseline(id, baseline.content);
        } catch (error) {
            console.error('Failed to read the file baseline.', error);
            this.ctx.editorPane?.clearBaseline(id);
        }
    };

    // openDiff shows one file's changes against HEAD in a read-only tab. A
    // diff that is already open is focused instead of rebuilt.
    openDiff = async (path: string): Promise<void> => {
        const id = diffTabId(path);
        if (this.ctx.tabs.some((tab) => tab.id === id)) {
            this.ctx.activeId = id;
            this.ctx.searchActive = false;
            return;
        }
        try {
            const diff = await readGitDiff(path);
            this.host.showDocument(createDiffDocument(path, diff), false);
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // reloadDiffTab replaces a diff's snapshot with a fresh one. The model is
    // read-only, so this is the only way its text changes.
    reloadDiffTab = async (id: string, path: string): Promise<void> => {
        try {
            const document = createDiffDocument(path, await readGitDiff(path));
            this.ctx.tabs = updateDocument(this.ctx.tabs, id, { content: document.content });
            this.ctx.editorPane?.reload(id, document.content, document.eol);
        } catch (error) {
            console.error('Failed to refresh the diff.', error);
        }
    };

    refreshDiffTabs = async (): Promise<void> => {
        for (const tab of this.ctx.tabs) {
            if (tab.kind === 'diff' && tab.sourcePath !== undefined) {
                await this.reloadDiffTab(tab.id, tab.sourcePath);
            }
        }
    };

    // refreshGitAfterSave updates the marks and any diff of the file that was
    // just written, so the sidebar does not describe the previous content.
    refreshGitAfterSave = async (path: string): Promise<void> => {
        await this.refreshGitStatus();
        for (const tab of this.ctx.tabs) {
            if (tab.kind === 'diff' && tab.sourcePath === path) {
                await this.reloadDiffTab(tab.id, path);
            }
        }
    };

    constructor(ctx: WindowContext, host: GitHost) {
        this.ctx = ctx;
        this.host = host;
    }
}
