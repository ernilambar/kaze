import { CopyText, RevealFile } from '../../bindings/kaze/internal/app/app';
import { ROOT_LOCKED_COMMANDS } from '../contextmenu';
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
} from '../filetree';
import { errorMessage, relativePath } from '../sidebar';
import { findDocumentByPath, pathWithin, retargetDocumentPaths } from '../tabs';
import type { CopyPathKind } from '../workspacecommands';
import type { WindowContext } from './context.svelte';
import type { Documents } from './documents.svelte';
import type { GitController } from './git.svelte';
import type { Prompts } from './prompts.svelte';

// TreeOps runs the explorer's file commands: New File, New Folder, Rename,
// Duplicate, Cut/Copy/Paste, Move, Delete (to the Trash) and drag to move.
// Every one of them is a Go binding, because the file system is the backend's
// business; the frontend only decides which entry, asks for a name and
// retargets the tabs that follow a path. It also owns the tree's reveal and
// expanded state.
export class TreeOps {
    private readonly ctx: WindowContext;
    private readonly prompts: Prompts;
    private readonly git: GitController;
    private readonly documents: Documents;

    // clipboardPath is the tree entry Cut or Copy put aside. The operating
    // system clipboard is not used: moving a file is not the same thing as
    // copying its path, and the tree only ever moves one entry.
    clipboardPath = $state('');

    clipboardCut = $state(false);

    // busyPath is the entry a tree command is running on, so its row can show
    // that something is happening instead of the command looking ignored.
    busyPath = $state('');

    // revealedPath is the row the tree expands to. It follows the active tab
    // only while auto-reveal is on.
    revealedPath = $state('');

    expandedPaths = $state<string[]>([]);

    treeRefreshToken = $state(0);

    // refreshTree rereads the tree after a command changed what is on disk.
    refreshTree = (): void => {
        this.treeRefreshToken++;
    };

    // applyTreeMove retargets every tab whose file the command moved. A buffer
    // keeps its text, edits and undo history; only the path it saves to changes.
    // The renamed file's git baseline and diff marks are reread, because HEAD
    // still knows the old name.
    applyTreeMove = (from: string, to: string): void => {
        if (from === '' || to === '' || from === to) {
            return;
        }
        const previous = this.ctx.tabs;
        const next = retargetDocumentPaths(this.ctx.tabs, [{ from, to }]);
        if (next === previous) {
            return;
        }
        this.ctx.tabs = next;
        for (const tab of next) {
            const before = previous.find((entry) => entry.id === tab.id);
            if (before === undefined || before.path === tab.path) {
                continue;
            }
            // HEAD still knows the old name, so the gutter bars are recomputed
            // against the path the file now has.
            if (tab.kind === 'text') {
                void this.git.loadBaseline(tab.id, tab.path);
            }
        }
        // A diff tab followed its file, so the snapshot it shows is stale.
        if (next.some((tab) => tab.kind === 'diff')) {
            void this.git.refreshDiffTabs();
        }
        void this.git.refreshGitStatus();
    };

    // treeError reports a failed tree command in the status bar, where every
    // other notice appears, and keeps the tree as it was.
    treeError = (error: unknown): void => {
        this.ctx.notice = errorMessage(error);
    };

    // newFileFromMenu is File → New File: the file is created inside a sidebar
    // folder, so it exists on disk from the start. The folder is the one the
    // selected entry lives in; with no folders there is nowhere to create one,
    // so the menu opens a buffer instead rather than doing nothing.
    newFileFromMenu = async (): Promise<void> => {
        const folder = this.selectedFolder();
        if (folder === '') {
            this.ctx.notice = 'Add a folder to the sidebar to create a file.';
            this.documents.newFile();
            return;
        }
        await this.newFileInFolder(folder);
    };

    // newFolderFromMenu is File → New Folder, in the selected entry's folder.
    newFolderFromMenu = async (): Promise<void> => {
        const folder = this.selectedFolder();
        if (folder === '') {
            this.ctx.notice = 'Add a folder to the sidebar to create a folder.';
            return;
        }
        await this.newFolderInFolder(folder);
    };

    // selectedFolder is the folder a menu command acts on. The tree highlights
    // the entry the command left behind, so a selected file means the folder it
    // lives in and a selected folder means itself; with nothing selected the
    // first root is the workspace's default.
    selectedFolder = (): string => {
        const selected = this.revealedPath;
        if (selected === '') {
            return this.ctx.folders[0] ?? '';
        }
        const tab = findDocumentByPath(this.ctx.tabs, selected);
        if (tab !== undefined && tab.path !== '') {
            return parentPath(selected);
        }
        return selected;
    };

    // newFileInFolder is New File from the tree: the file is created on disk
    // first, so an empty file exists even if the user never types into it, then
    // opened as a tab.
    newFileInFolder = async (folder: string): Promise<void> => {
        await this.prompts.askName({
            title: 'New File',
            label: `Name for the new file in ${entryName(folder)}`,
            submitLabel: 'Create File',
            run: async (name) => {
                try {
                    const entry = await createFile(folder, name);
                    this.refreshTree();
                    await this.documents.openFile(entry.path);
                    this.ctx.notice = '';
                    return '';
                } catch (error) {
                    this.ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    };

    // newFolderInFolder is New Folder. The new folder is expanded so the user
    // sees where it went, which matters when it lands between many siblings.
    newFolderInFolder = async (folder: string): Promise<void> => {
        await this.prompts.askName({
            title: 'New Folder',
            label: `Name for the new folder in ${entryName(folder)}`,
            submitLabel: 'Create Folder',
            run: async (name) => {
                try {
                    const entry = await createFolder(folder, name);
                    this.refreshTree();
                    this.revealInTree(entry.path);
                    this.ctx.notice = '';
                    return '';
                } catch (error) {
                    this.ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    };

    // renameEntry accepts a new name and, when the text carries a folder, a new
    // location too. The field holds the whole path with only the file stem
    // selected, so typing replaces the name and leaves the folder alone. The tab
    // that holds the file follows the new path, so the buffer stays open and
    // keeps its edits and undo history.
    renameEntry = async (path: string): Promise<void> => {
        await this.prompts.askName({
            title: 'Rename',
            value: path,
            selectStem: true,
            submitLabel: 'Rename',
            run: async (target) => {
                try {
                    const entry = await renamePath(path, target);
                    this.applyTreeMove(path, entry.path);
                    this.refreshTree();
                    this.revealInTree(entry.path);
                    // A Cut waiting on the old path would move nothing, so it is
                    // dropped rather than left pointing at a file that is gone.
                    if (this.clipboardPath === path) {
                        this.clipboardPath = entry.path;
                    }
                    this.ctx.notice = '';
                    return '';
                } catch (error) {
                    this.ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    };

    // duplicateEntry asks for the copy's path, prefilled with the free " copy"
    // name the backend suggests. Accepting it does what Finder does; editing it
    // names the copy before it is written, and a folder in the text puts the
    // copy there. The copy is not opened: the user asked for a second file, not
    // for a second tab.
    duplicateEntry = async (path: string): Promise<void> => {
        let suggestion: string;
        try {
            suggestion = await duplicatePathSuggestion(path);
        } catch (error) {
            this.treeError(error);
            return;
        }
        await this.prompts.askName({
            title: 'Duplicate',
            value: suggestion,
            selectStem: true,
            submitLabel: 'Duplicate',
            run: async (target) => {
                try {
                    const entry = await duplicatePath(path, target);
                    this.refreshTree();
                    this.revealInTree(entry.path);
                    await this.git.refreshGitStatus();
                    this.ctx.notice = `Created ${entry.name}`;
                    return '';
                } catch (error) {
                    this.ctx.notice = '';
                    return errorMessage(error);
                }
            },
        });
    };

    // cutEntry puts the entry aside for Paste. It changes nothing on disk: the
    // move happens when the user chooses a folder to paste into.
    cutEntry = (path: string): void => {
        this.clipboardPath = path;
        this.clipboardCut = true;
        this.ctx.notice = `Ready to move ${entryName(path)}`;
    };

    copyEntry = (path: string): void => {
        this.clipboardPath = path;
        this.clipboardCut = false;
        this.ctx.notice = `Ready to copy ${entryName(path)}`;
    };

    // pasteInto copies or moves the entry the clipboard holds into a folder. A
    // cut is cleared after the move, so a second Paste cannot fail on a source
    // that is no longer there.
    pasteInto = async (folder: string): Promise<void> => {
        const source = this.clipboardPath;
        if (source === '') {
            return;
        }
        this.busyPath = folder;
        try {
            if (this.clipboardCut) {
                const entry = await movePath(source, folder);
                this.applyTreeMove(source, entry.path);
                this.clipboardPath = '';
                this.clipboardCut = false;
                this.revealInTree(entry.path);
                this.ctx.notice = `Moved ${entry.name}`;
            } else {
                const entry = await copyPath(source, folder);
                this.revealInTree(entry.path);
                this.ctx.notice = `Copied ${entry.name}`;
            }
            this.refreshTree();
            await this.git.refreshGitStatus();
        } catch (error) {
            this.treeError(error);
        } finally {
            this.busyPath = '';
        }
    };

    // moveEntry is a drop on a folder row. It is the same move Paste does, with
    // the source coming from the drag rather than the clipboard.
    moveEntry = async (source: string, folder: string): Promise<void> => {
        // A root row is not draggable; a root reaching here is refused for the
        // same reason the menu refuses to move one.
        if (this.ctx.folders.includes(source)) {
            return;
        }
        this.busyPath = folder;
        try {
            const entry = await movePath(source, folder);
            this.applyTreeMove(source, entry.path);
            this.finishTreeMove(source, entry.path, entry.name);
        } catch (error) {
            this.treeError(error);
        } finally {
            this.busyPath = '';
        }
    };

    // finishTreeMove rereads what a move invalidated: the tree, because the
    // entry left one folder and landed in another, and git, because a repository
    // may have gained or lost a path. The moved entry is revealed so it never
    // lands out of sight.
    finishTreeMove = (source: string, target: string, name: string): void => {
        this.applyTreeMove(source, target);
        this.refreshTree();
        this.revealInTree(target);
        void this.git.refreshGitStatus();
        this.ctx.notice = `Moved ${name}`;
    };

    // moveTreeEntry is Move…: the field holds the entry's full path with the
    // folder selected, so typing replaces where it lives and the file name stays
    // visible and intact. Move and Rename share the path operation; only the
    // initial selection differs. A refusal reopens the prompt on the same text
    // so a typo is corrected in place.
    moveTreeEntry = async (path: string): Promise<void> => {
        await this.prompts.askName({
            title: 'Move',
            value: path,
            selectFolder: true,
            submitLabel: 'Move',
            run: async (target) => {
                this.busyPath = path;
                try {
                    const entry = await renamePath(path, target);
                    this.finishTreeMove(path, entry.path, entry.name);
                    return '';
                } catch (error) {
                    this.ctx.notice = '';
                    return errorMessage(error);
                } finally {
                    this.busyPath = '';
                }
            },
        });
    };

    // trashEntry asks first, then moves the entry to the platform's Trash. Its
    // tabs close with it: the files they show no longer exist, and a tab that
    // cannot be saved is worse than no tab. A modified buffer asks about its
    // edits before the file goes, so nothing is lost by accident.
    trashEntry = async (path: string): Promise<void> => {
        const affected = this.ctx.tabs.filter(
            (tab) =>
                (tab.path !== '' && pathWithin(tab.path, path)) ||
                (tab.kind === 'diff' && pathWithin(tab.sourcePath ?? '', path)),
        );
        const dirty = affected.filter((tab) => tab.dirty);
        const confirmed = await this.prompts.askConfirm(
            'Delete',
            dirty.length > 0
                ? `Delete ${entryName(path)}? It moves to the Trash, and ${dirty.length === 1 ? 'one open buffer has' : `${dirty.length} open buffers have`} unsaved changes that will be lost.`
                : `Delete ${entryName(path)}? It moves to the Trash, so you can put it back from there.`,
            'Delete',
        );
        if (!confirmed) {
            return;
        }
        this.busyPath = path;
        try {
            await trashPath(path);
            for (const tab of affected) {
                this.documents.removeTab(tab.id);
            }
            if (pathWithin(this.clipboardPath, path)) {
                this.clipboardPath = '';
                this.clipboardCut = false;
            }
            this.refreshTree();
            await this.git.refreshGitStatus();
            this.ctx.notice = `Moved ${entryName(path)} to the Trash`;
        } catch (error) {
            this.treeError(error);
        } finally {
            this.busyPath = '';
        }
    };

    // revealInTree expands the tree to path and scrolls it into view. New
    // entries use it so they never land out of sight.
    revealInTree = (path: string): void => {
        this.revealedPath = path;
        this.treeRefreshToken++;
    };

    runTreeCommand = async (id: string, path: string, isDir: boolean): Promise<void> => {
        // The menu disables these for a root; refusing here keeps any other
        // caller from moving or trashing a folder the sidebar still lists.
        if (ROOT_LOCKED_COMMANDS.includes(id) && this.ctx.folders.includes(path)) {
            return;
        }
        if (id === 'reveal') {
            await this.revealPath(path);
            return;
        }
        // Commands that create or paste need a folder: a file row means the
        // folder it lives in.
        const folder = isDir ? path : parentPath(path);
        switch (id) {
            case 'open':
                await this.openTreeEntry(path, isDir);
                break;
            case 'new-file':
                await this.newFileInFolder(folder);
                break;
            case 'new-folder':
                await this.newFolderInFolder(folder);
                break;
            case 'cut':
                this.cutEntry(path);
                break;
            case 'copy':
                this.copyEntry(path);
                break;
            case 'paste':
                await this.pasteInto(folder);
                break;
            case 'copy-name':
                await this.copyTreeText('name', path);
                break;
            case 'copy-path':
                await this.copyTreeText('file', path);
                break;
            case 'copy-relative-path':
                await this.copyTreeText('relative', path);
                break;
            case 'duplicate':
                await this.duplicateEntry(path);
                break;
            case 'rename':
                await this.renameEntry(path);
                break;
            case 'move':
                await this.moveTreeEntry(path);
                break;
            case 'trash':
                await this.trashEntry(path);
                break;
        }
    };

    // openTreeEntry is the row's primary action: a file opens in the editor, a
    // folder expands in the tree. A file that is already open is focused rather
    // than reread, so edits in its buffer survive.
    openTreeEntry = async (path: string, isDir: boolean): Promise<void> => {
        if (isDir) {
            this.revealInTree(path);
            return;
        }
        await this.documents.openFile(path);
    };

    // copyTreeText copies a path spelling of one tree entry. The variants match
    // the Copy Path items that act on the active buffer, so the same three
    // strings are reachable from the tree.
    copyTreeText = async (kind: CopyPathKind, path: string): Promise<void> => {
        const text =
            kind === 'file'
                ? path
                : kind === 'relative'
                  ? relativePath(path, this.ctx.folders)
                  : entryName(path);
        try {
            await CopyText(text);
            this.ctx.notice = `Copied ${text}`;
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // revealPath shows one file in the platform's file manager.
    revealPath = async (path: string): Promise<void> => {
        try {
            await RevealFile(path);
            this.ctx.notice = '';
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // setExpanded tracks which folders the tree has open, so their contents can
    // be watched for external changes.
    setExpanded = (path: string, open: boolean): void => {
        if (open) {
            if (!this.expandedPaths.includes(path)) {
                this.expandedPaths = [...this.expandedPaths, path];
            }
            return;
        }
        this.expandedPaths = this.expandedPaths.filter((entry) => entry !== path);
    };

    constructor(ctx: WindowContext, prompts: Prompts, git: GitController, documents: Documents) {
        this.ctx = ctx;
        this.prompts = prompts;
        this.git = git;
        this.documents = documents;
    }
}
