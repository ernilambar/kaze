import {
    FileExists,
    MoveTabToNewWindow,
    OpenFileDialog,
    ResolveClose,
    SaveDocument,
    SaveFileDialog,
} from '../../bindings/kaze/internal/app/app';
import type { SaveResult } from '../../bindings/kaze/internal/document/models';
import {
    createUntitledDocument,
    normalizeEol,
    readDocument,
    type LaunchBuffer,
    type OpenDocument,
} from '../document';
import { languageForDocument } from '../editor';
import type { CloseReason } from '../filecommands';
import { errorMessage } from '../sidebar';
import {
    closeDocumentTab,
    findDocumentByPath,
    moveDocumentTab,
    openDocumentTab,
    setDocumentPinned,
    stepTabId,
    tabIdAt,
    updateDocument,
} from '../tabs';
import { appendFinalNewline, trimTrailingWhitespace as trimTrailingWhitespaceText } from '../text';
import type { Appearance } from './appearance.svelte';
import type { WindowContext } from './context.svelte';
import type { GitController } from './git.svelte';
import type { Prompts } from './prompts.svelte';

// ClosedTab is one entry in the Reopen Closed Tab stack. The buffer text is
// kept because an untitled tab has no file to read again.
interface ClosedTab {
    document: OpenDocument;
    content: string;
}

// DocumentsHost is what the document lifecycle needs from the window that it
// does not own: the product name for the quit prompt, Open Recent, and a tree
// reread after an external change.
export interface DocumentsHost {
    appName: () => string;
    recordRecent: (path: string) => Promise<void>;
    refreshTree: () => void;
}

// Documents opens, saves, closes and reloads the window's tabs. The tab list
// itself lives in the context; this controller owns the rules for changing it.
export class Documents {
    private readonly ctx: WindowContext;
    private readonly prompts: Prompts;
    private readonly git: GitController;
    private readonly appearance: Appearance;
    private readonly host: DocumentsHost;

    // closedTabs is the Reopen Closed Tab stack. It is never rendered, so it
    // stays a plain array instead of reactive state.
    closedTabs: ClosedTab[] = [];

    // showDocument adds a tab for the document unless its file is already open,
    // then makes it active. Restoring a session passes record=false, so a
    // relaunch does not reshuffle the Open Recent list.
    showDocument = (document: OpenDocument, record = true): void => {
        const opened = openDocumentTab(this.ctx.tabs, document);
        this.ctx.tabs = opened.tabs;
        this.ctx.activeId = opened.activeId;
        this.ctx.searchActive = false;
        if (record && document.path !== '') {
            void this.host.recordRecent(document.path);
        }
        if (document.kind === 'text' && document.path !== '') {
            void this.git.loadBaseline(document.id, document.path);
        }
    };

    // openFile opens one file and returns the tab that now holds it, or an
    // empty id when it could not be read. The id is the path spelling the tab
    // was created with, which can differ from the one passed in when the file
    // is open through a symlink.
    openFile = async (path: string): Promise<string> => {
        // Clicking a file that is already open focuses its tab instead of
        // rereading the file, which would discard edits made in the buffer.
        const existing = findDocumentByPath(this.ctx.tabs, path);
        if (existing !== undefined) {
            this.ctx.activeId = existing.id;
            this.ctx.notice = '';
            return existing.id;
        }
        const opened = await readDocument(path);
        if (opened === undefined) {
            this.ctx.notice = `Cannot open ${path}`;
            return '';
        }
        this.showDocument(opened);
        this.ctx.notice = '';
        return this.ctx.activeId;
    };

    openFromDialog = async (): Promise<void> => {
        if (this.ctx.busy) {
            return;
        }
        this.ctx.busy = true;
        try {
            const path = await OpenFileDialog();
            // Cancelling the dialog is a decision, not a failure.
            if (path === '') {
                return;
            }
            await this.openFile(path);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        } finally {
            this.ctx.busy = false;
        }
    };

    // newFile opens an empty buffer with no path yet. Save turns it into a
    // Save As, which is how most editors treat an untitled document.
    newFile = (): void => {
        const document = createUntitledDocument(this.ctx.tabs);
        this.ctx.tabs = [...this.ctx.tabs, document];
        this.ctx.activeId = document.id;
        this.ctx.notice = '';
    };

    removeTab = (id: string): void => {
        const result = closeDocumentTab(this.ctx.tabs, id, this.ctx.activeId);
        this.ctx.tabs = result.tabs;
        this.ctx.activeId = result.activeId;
    };

    // setPinned records a tab's pin. The tab moves to its group's boundary, so
    // the pinned prefix the rest of the strip relies on stays intact.
    setPinned = (id: string, pinned: boolean): void => {
        this.ctx.tabs = setDocumentPinned(this.ctx.tabs, id, pinned);
    };

    togglePinTab = (id: string = this.ctx.activeId): void => {
        const tab = this.ctx.findTab(id);
        if (tab === undefined) {
            return;
        }
        this.setPinned(id, !tab.pinned);
    };

    // closeActiveTabFromKeyboard is Cmd+W and File ▸ Close Tab. A pinned tab is
    // not closed by it; the context menu's Close Tab still is.
    closeActiveTabFromKeyboard = (): void => {
        const tab = this.ctx.active;
        if (tab?.pinned) {
            this.ctx.notice = 'Unpin the tab to close it.';
            return;
        }
        if (this.ctx.activeId !== '') {
            void this.closeTab(this.ctx.activeId);
        }
    };

    // bufferText is what the editor holds right now. A tab that has never been
    // displayed has no model yet, so its loaded content is the answer.
    bufferText = (tab: OpenDocument): string => {
        return this.ctx.editorPane?.value(tab.id) ?? tab.content;
    };

    chooseSavePath = async (tab: OpenDocument): Promise<string> => {
        try {
            return await SaveFileDialog(tab.path !== '' ? tab.path : tab.name);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
            return '';
        }
    };

    applySaved = (id: string, result: SaveResult, content: string): void => {
        const previousPath = this.ctx.findTab(id)?.path ?? '';
        // The editor decides whether the buffer is clean: typing during the
        // write leaves newer text in the model than the file now holds.
        const clean = this.ctx.editorPane?.markSaved(id, content) ?? true;
        this.ctx.tabs = updateDocument(this.ctx.tabs, id, {
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
        const saved = this.ctx.findTab(id);
        this.ctx.editorPane?.setLanguage(
            id,
            languageForDocument(
                result.path,
                this.ctx.editorPane?.value(id) ?? saved?.content ?? '',
                saved?.language ?? '',
            ),
        );
        if (previousPath !== result.path) {
            void this.git.loadBaseline(id, result.path);
        }
        void this.git.refreshGitAfterSave(result.path).catch((error) =>
            console.error('Failed to refresh the git status.', error),
        );
        this.ctx.notice = '';
    };

    // saveDocument writes one buffer, asking for a path when it has none. A file
    // another program changed is reported as a conflict first, so its content is
    // never overwritten without the user saying so. False means nothing was
    // written: a cancelled dialog, a conflict the user declined, or an error.
    saveDocument = async (id: string, saveAs = false): Promise<boolean> => {
        const tab = this.ctx.findTab(id);
        // Pictures are read-only here, and a diff is a generated view, so
        // there is nothing to write back for either.
        if (tab === undefined || tab.kind !== 'text') {
            return true;
        }

        // These save-time rewrites edit the buffer before its text is read, so
        // the write and the model still agree and the tab does not stay dirty.
        if (this.appearance.filesTrimTrailingWhitespace) {
            this.ctx.editorPane?.trimTrailingWhitespaceInBuffer(tab.id);
        }
        if (this.appearance.editorSettings.ensureFinalNewline) {
            this.ctx.editorPane?.ensureFinalNewline(tab.id);
        }
        let content = this.bufferText(tab);
        if (this.appearance.editorSettings.ensureFinalNewline) {
            content = appendFinalNewline(content);
        }
        // The model's text is already normalized to the document's EOL, but a
        // buffer that was never shown comes straight from the tab. Normalizing
        // here keeps the baseline markSaved compares with the model.
        content = normalizeEol(content, tab.eol);
        if (this.appearance.filesTrimTrailingWhitespace) {
            content = trimTrailingWhitespaceText(content);
        }
        let path = tab.path;
        let expected = tab.modified;
        if (saveAs || path === '') {
            path = await this.chooseSavePath(tab);
            if (path === '') {
                return false;
            }
            expected = 0;
            // Two tabs writing one file would disagree about its content, so
            // Save As refuses a path that is already open. The dialog returns a
            // resolved path, so an alias of an open file counts too.
            if (
                this.ctx.tabs.some(
                    (other) =>
                        other.id !== id && (other.path === path || other.realPath === path),
                )
            ) {
                this.ctx.notice = `${path} is already open.`;
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
                    const proceed = await this.prompts.askConfirm(
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
                    const choice = await this.prompts.askConflict(
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
                            this.ctx.notice = `Could not reload ${tab.name} from disk.`;
                            return false;
                        }
                        this.applyReload(id, opened);
                        void this.git.loadBaseline(id, tab.path);
                        void this.git.refreshDiffTabs();
                        void this.git.refreshGitStatus();
                        return true;
                    }
                    overwrite = true;
                    continue;
                }
                this.applySaved(id, result, content);
                return true;
            }
        } catch (error) {
            this.ctx.notice = errorMessage(error);
            return false;
        }
    };

    // saveAll writes every modified buffer in tab order. A cancelled Save As or
    // a failed write stops the run, so the rest are left exactly as they were.
    saveAll = async (): Promise<boolean> => {
        for (const tab of this.ctx.tabs.filter((entry) => entry.dirty)) {
            if (!(await this.saveDocument(tab.id))) {
                return false;
            }
        }
        return true;
    };

    // closeTab drops a buffer. A modified buffer asks first, because closing it
    // is also what discards its undo stack. The result is false when the user
    // cancelled or the save failed, so a group command can stop there.
    closeTab = async (id: string): Promise<boolean> => {
        const tab = this.ctx.findTab(id);
        if (tab === undefined) {
            return false;
        }
        if (tab.dirty) {
            const answer = await this.prompts.askUnsaved({
                title: 'Close Tab',
                message: `Save changes to ${tab.name} before closing?`,
                details: [],
                confirmLabel: 'Save',
            });
            if (answer === 'cancel') {
                return false;
            }
            if (answer === 'save' && !(await this.saveDocument(id))) {
                return false;
            }
        }
        this.rememberClosedTab(tab);
        this.removeTab(id);
        return true;
    };

    // closeTabsInOrder closes a group one tab at a time, stopping at the first
    // cancelled prompt or failed save so the remaining tabs stay exactly as
    // they were.
    closeTabsInOrder = async (candidates: readonly OpenDocument[]): Promise<boolean> => {
        for (const tab of candidates) {
            if (!(await this.closeTab(tab.id))) {
                return false;
            }
        }
        return true;
    };

    resolveClose = async (quit: boolean): Promise<void> => {
        try {
            await ResolveClose(this.ctx.windowName, quit);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // confirmClose answers Go's close request for this window. It is the one
    // place that decides whether the window may go away, so the close button
    // and Cmd+Q behave the same and share one prompt. The reason changes the
    // wording: closing one window is not quitting the app.
    confirmClose = async (reason: CloseReason = 'quit'): Promise<void> => {
        // Another prompt is already on screen; the user is deciding, so a
        // second Cmd+Q must not stack a dialog on top of it. Answering the
        // request with "cancel" clears Go's pending flag, so the next Cmd+Q
        // asks again instead of being swallowed.
        if (this.prompts.unsaved !== null || this.prompts.confirm !== null) {
            await this.resolveClose(false);
            return;
        }

        const dirty = this.ctx.tabs.filter((tab) => tab.dirty);
        if (dirty.length === 0) {
            await this.resolveClose(true);
            return;
        }

        const answer = await this.prompts.askUnsaved({
            title: reason === 'close' ? 'Close Window' : `Quit ${this.host.appName()}`,
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
            await this.resolveClose(false);
            return;
        }
        if (answer === 'save' && !(await this.saveAll())) {
            // Saving failed or was cancelled: stay open rather than drop the
            // buffers the user just tried to keep.
            await this.resolveClose(false);
            return;
        }
        await this.resolveClose(true);
    };

    // rememberClosedTab pushes a tab onto the Reopen Closed Tab stack, newest
    // last. The stack is capped so a long session cannot grow without bound.
    rememberClosedTab = (tab: OpenDocument): void => {
        this.closedTabs.push({ document: { ...tab }, content: this.bufferText(tab) });
        if (this.closedTabs.length > 20) {
            this.closedTabs.shift();
        }
    };

    reopenClosedTab = async (): Promise<void> => {
        const closed = this.closedTabs.pop();
        if (closed === undefined) {
            this.ctx.notice = 'No closed tab to reopen.';
            return;
        }

        const saved = closed.document;
        // A clean file tab is reopened from disk. Closing a dirty tab already
        // discarded its edits, so there is nothing newer to restore.
        if (saved.kind === 'text' && saved.path !== '' && !saved.dirty) {
            const opened = await readDocument(saved.path);
            if (opened !== undefined) {
                this.showDocument(opened);
                if (saved.pinned) {
                    this.setPinned(this.ctx.activeId, true);
                }
                this.ctx.notice = '';
                return;
            }
        }

        // An untitled buffer lives only in memory, so its text comes from the
        // stack. A file that disappeared falls back to the same snapshot.
        let restored = saved;
        if (saved.kind === 'text' && saved.path === '' && this.ctx.tabs.some((tab) => tab.id === saved.id)) {
            // A New File made since the close may already own this name; give
            // the restored buffer a fresh id so the two do not share a model.
            const replacement = createUntitledDocument(this.ctx.tabs);
            restored = { ...saved, id: replacement.id, name: replacement.name };
        }
        this.showDocument({
            ...restored,
            pinned: false,
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
        if (saved.pinned) {
            this.setPinned(this.ctx.activeId, true);
        }
        this.ctx.notice = '';
    };

    stepTab = (direction: 1 | -1): void => {
        if (this.ctx.tabs.length === 0) {
            return;
        }
        this.ctx.searchActive = false;
        this.ctx.activeId = stepTabId(this.ctx.tabs, this.ctx.activeId, direction);
    };

    goToTab = (position: number): void => {
        const id = tabIdAt(this.ctx.tabs, position);
        if (id === '') {
            return;
        }
        this.ctx.searchActive = false;
        this.ctx.activeId = id;
    };

    // reorderTab keeps the dragged tab on the same side of the target the
    // insertion line showed. The tab list order is the saved session order.
    reorderTab = (id: string, targetId: string, placeAfter: boolean): void => {
        this.ctx.tabs = moveDocumentTab(this.ctx.tabs, id, targetId, placeAfter);
    };

    // applyReload replaces a clean buffer with what is on disk. The model is
    // told first, so the editor's dirty check agrees with the tab record.
    applyReload = (id: string, opened: OpenDocument): void => {
        if (opened.kind === 'text') {
            this.ctx.editorPane?.reload(id, opened.content, opened.eol);
        }
        this.ctx.tabs = updateDocument(this.ctx.tabs, id, {
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
    };

    // markDeletedIfGone flags a tab whose file no longer exists, keeping its
    // buffer so a save writes it back. A read can fail for other reasons (a
    // binary swapped in, lost permissions), so only a missing file marks the
    // tab; anything else stays silent like before.
    markDeletedIfGone = async (tab: OpenDocument): Promise<boolean> => {
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
            this.ctx.tabs = updateDocument(this.ctx.tabs, tab.id, { deleted: true });
            return true;
        }
        return false;
    };

    // reloadIfClean applies a read that was started while the tab was clean.
    // The user may have typed while it was in flight, and those edits win.
    reloadIfClean = (id: string, opened: OpenDocument): void => {
        const current = this.ctx.findTab(id);
        if (current === undefined || current.dirty) {
            return;
        }
        this.applyReload(id, opened);
    };

    refreshOpenDocuments = async (): Promise<void> => {
        for (const snapshot of this.ctx.tabs) {
            // Earlier reads await, so the tab may have changed or closed since
            // the loop started.
            const tab = this.ctx.findTab(snapshot.id);
            if (tab === undefined || tab.kind !== 'text' || tab.path === '') {
                continue;
            }
            if (!tab.dirty) {
                // External changes never overwrite edits the user has not
                // saved; the next save reports the conflict instead.
                const opened = await readDocument(tab.path);
                if (opened !== undefined) {
                    this.reloadIfClean(tab.id, opened);
                    continue;
                }
            }
            // The file could not be read, or the tab is dirty so reading was
            // skipped: only a file that is really gone marks its tab.
            await this.markDeletedIfGone(tab);
        }
    };

    // handleWatchedChanges reloads clean buffers and refreshes the tree for
    // anything else. A dirty buffer stays as it is and says why; a deleted
    // file marks its tab either way, keeping the buffer for a save.
    handleWatchedChanges = async (paths: string[]): Promise<void> => {
        let refreshTree = false;
        for (const path of paths) {
            const tab = findDocumentByPath(this.ctx.tabs, path);
            if (tab === undefined || tab.kind !== 'text') {
                refreshTree = true;
                continue;
            }
            if (tab.dirty) {
                if (!(await this.markDeletedIfGone(tab))) {
                    this.ctx.notice = `${tab.name} changed on disk; your edits were kept.`;
                }
                continue;
            }
            const opened = await readDocument(path);
            if (opened === undefined) {
                await this.markDeletedIfGone(tab);
                continue;
            }
            this.reloadIfClean(tab.id, opened);
        }
        if (refreshTree) {
            this.host.refreshTree();
        }
        // A file that changed on disk is exactly what the marks and an open
        // diff describe, so both follow the same reread.
        void this.git.refreshGitStatus();
        void this.git.refreshDiffTabs();
    };

    // moveTabToNewWindow hands one tab to a new window and drops it from this
    // one. The buffer text travels with the request, so unsaved edits are not
    // lost; the tab is removed only after the call succeeds. Without an id the
    // active tab is moved, which is what the menu command does.
    moveTabToNewWindow = async (id: string = this.ctx.activeId): Promise<void> => {
        const tab = this.ctx.findTab(id);
        if (tab === undefined) {
            this.ctx.notice = 'Open a tab to move to a new window.';
            return;
        }
        const buffer: LaunchBuffer = {
            name: tab.name,
            path: tab.kind === 'diff' ? '' : tab.path,
            real_path: tab.kind === 'diff' ? '' : tab.realPath,
            kind: tab.kind,
            content: tab.kind === 'text' ? this.bufferText(tab) : tab.content,
            saved_content: tab.content,
            modified: tab.modified,
            dirty: tab.kind === 'text' ? tab.dirty : false,
            source_path: tab.sourcePath ?? '',
            bom: tab.bom,
            eol: tab.eol,
            invalid_utf8: tab.invalidUTF8,
            pinned: tab.pinned ?? false,
        };
        try {
            await MoveTabToNewWindow(buffer);
            this.removeTab(tab.id);
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    constructor(
        ctx: WindowContext,
        prompts: Prompts,
        git: GitController,
        appearance: Appearance,
        host: DocumentsHost,
    ) {
        this.ctx = ctx;
        this.prompts = prompts;
        this.git = git;
        this.appearance = appearance;
        this.host = host;
    }
}
