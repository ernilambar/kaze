import type * as models from '../bindings/kaze/models';
import {
    CopyPath,
    CreateFile,
    CreateFolder,
    DuplicatePath,
    MovePath,
    RenamePath,
    TrashPath,
} from '../bindings/kaze/app';

// FileTreeResult mirrors FileTreeResult in filetree.go: the path a tree command
// produced and the same path with symlinks resolved. A tab matches on the
// resolved one and displays the spelling that was written.
export interface FileTreeEntry {
    path: string;
    realPath: string;
    name: string;
    isDir: boolean;
}

export function toFileTreeEntry(result: models.FileTreeResult): FileTreeEntry {
    return {
        path: result.path,
        realPath: result.real_path || result.path,
        name: result.name,
        isDir: result.is_dir,
    };
}

// entryName is the last segment of path. The tree names the folder a command
// runs in, so a new entry only needs a name of its own.
export function entryName(path: string): string {
    const parts = path.split(/[\\/]/).filter((part) => part !== '');
    return parts.length === 0 ? path : parts[parts.length - 1];
}

// parentPath is the folder path lives in. The root has no parent here, which the
// caller treats as "the folder itself".
export function parentPath(path: string): string {
    const trimmed = path.replace(/[\\/]+$/, '');
    const cut = Math.max(trimmed.lastIndexOf('/'), trimmed.lastIndexOf('\\'));
    if (cut <= 0) {
        return cut === 0 ? trimmed.slice(0, 1) : '';
    }
    return trimmed.slice(0, cut);
}

// createFile makes an empty file in parent. The name is one segment: the
// backend rejects anything that would walk up the tree.
export async function createFile(parent: string, name: string): Promise<FileTreeEntry> {
    return toFileTreeEntry(await CreateFile(parent, name));
}

export async function createFolder(parent: string, name: string): Promise<FileTreeEntry> {
    return toFileTreeEntry(await CreateFolder(parent, name));
}

// renamePath renames in place, or moves and renames when target carries a
// folder. The backend creates missing parent folders.
export async function renamePath(path: string, target: string): Promise<FileTreeEntry> {
    return toFileTreeEntry(await RenamePath(path, target));
}

// duplicatePath copies next to the source under a " copy" name.
export async function duplicatePath(path: string): Promise<FileTreeEntry> {
    return toFileTreeEntry(await DuplicatePath(path));
}

// copyPath copies into a folder, keeping the name when it is free.
export async function copyPath(path: string, targetDir: string): Promise<FileTreeEntry> {
    return toFileTreeEntry(await CopyPath(path, targetDir));
}

// movePath moves into a folder. It is also what a drop on a folder row does.
export async function movePath(path: string, targetDir: string): Promise<FileTreeEntry> {
    return toFileTreeEntry(await MovePath(path, targetDir));
}

// trashPath moves the entry to the platform's Trash. Nothing in the tree deletes
// for good, so a mistake can still be undone in Finder.
export async function trashPath(path: string): Promise<void> {
    await TrashPath(path);
}

// DRAG_TYPE marks a drag that started on a tree row. A drop can then tell a move
// inside the tree from files dragged in from another application.
export const DRAG_TYPE = 'application/x-kaze-tree-path';

// A drag spans several rows and every row is its own component, so what is
// being dragged lives here rather than in one row's state. It is a plain value,
// not reactive state: only the drop needs it, and the highlight is per row.
let dragging = '';

export function setDraggingPath(path: string): void {
    dragging = path;
}

export function draggingPath(): string {
    return dragging;
}

// dropSource is the path a drop would move: a row dragged from the tree, or a
// file dragged in from another application when the webview can name its path.
export function dropSource(transfer: DataTransfer | null): string {
    if (transfer === null) {
        return '';
    }
    const internal = transfer.getData(DRAG_TYPE);
    if (internal !== '') {
        return internal;
    }
    return fileDragPaths(transfer.files).at(0) ?? '';
}

// fileDragPaths are the local paths of the files a drag carries. A webview that
// does not expose them yields none, which the caller reads as "nothing to move"
// rather than as an error.
export function fileDragPaths(files: FileList | null): string[] {
    if (files === null) {
        return [];
    }
    const paths: string[] = [];
    for (const file of Array.from(files)) {
        const path = (file as File & { path?: string }).path;
        if (typeof path === 'string' && path !== '') {
            paths.push(path);
        }
    }
    return paths;
}
