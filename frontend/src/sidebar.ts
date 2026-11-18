import type * as models from '../bindings/kaze/internal/fileops/models';
import {
    AddSidebarFolder,
    LoadSidebarFolders,
    OpenFolderDialog,
    ReadFolder,
    RemoveAllSidebarFolders,
    RemoveSidebarFolder,
    SetSidebarFolders,
} from '../bindings/kaze/app';
import { onWindowEvent } from './windowstate';

// FolderEntry mirrors FolderEntry in internal/fileops/folder.go: one child of a folder. The
// backend lists one level at a time, so the tree only reads what it expands.
export interface FolderEntry {
    path: string;
    name: string;
    isDir: boolean;
}

export function toFolderEntry(entry: models.FolderEntry): FolderEntry {
    return { path: entry.path, name: entry.name, isDir: entry.is_dir };
}

// folderName is the label for a folder: its last path segment. The root folder
// has no segment of its own, so it keeps its slash.
export function folderName(folder: string): string {
    if (folder === '') {
        return '';
    }
    const parts = folder.split(/[\\/]/).filter((part) => part !== '');
    return parts.length === 0 ? '/' : parts[parts.length - 1];
}

// sidebarLabel summarizes the roots for the title and status bars. One root is
// named; several are counted, because naming only the first would mislead.
export function sidebarLabel(folders: string[]): string {
    if (folders.length === 0) {
        return '';
    }
    if (folders.length === 1) {
        return folderName(folders[0]);
    }
    return `${folders.length} folders`;
}

// errorMessage turns whatever the backend rejected with into a line the sidebar
// can show. Wails surfaces Go errors as strings rather than Error objects.
export function errorMessage(error: unknown): string {
    if (typeof error === 'string' && error !== '') {
        return error;
    }
    if (error instanceof Error && error.message !== '') {
        return error.message;
    }
    return 'Something went wrong';
}

// isWithin reports whether path sits inside folder. The trailing separator
// keeps /projects from matching /projects-old.
export function isWithin(folder: string, path: string): boolean {
    if (folder === '' || path === '') {
        return false;
    }
    const base = folder.endsWith('/') ? folder : `${folder}/`;
    return path.startsWith(base);
}

// loadSidebarFolders reads one window's roots. The launch window restores its
// saved list; a later window starts empty. A failure leaves the sidebar empty
// rather than blocking startup.
export async function loadSidebarFolders(window: string): Promise<string[]> {
    try {
        return (await LoadSidebarFolders(window)) ?? [];
    } catch (error) {
        console.error('Failed to load sidebar folders.', error);
        return [];
    }
}

// readFolder lists one level of a folder. Failures are thrown so the row that
// triggered the read can show the message in place.
export async function readFolder(path: string, root: string): Promise<FolderEntry[]> {
    return ((await ReadFolder(path, root)) ?? []).map(toFolderEntry);
}

// addSidebarFolder saves a root in one window and returns the new list. The
// backend rejects paths that are not folders and ignores folders already in
// the list.
export async function addSidebarFolder(window: string, folder: string): Promise<string[]> {
    return (await AddSidebarFolder(window, folder)) ?? [];
}

// removeSidebarFolder drops a root from one window and returns the new list.
export async function removeSidebarFolder(window: string, folder: string): Promise<string[]> {
    return (await RemoveSidebarFolder(window, folder)) ?? [];
}

// openFolderDialog asks the user for a folder. An empty string means the dialog
// was cancelled, which is not an error.
export async function openFolderDialog(): Promise<string> {
    return await OpenFolderDialog();
}

// listenForAddFolder runs handler when the File menu asks for the folder picker,
// so the menu and the sidebar button share one code path. Returns an unsubscribe
// function.
export function listenForAddFolder(handler: () => void): () => void {
    return onWindowEvent('app:add-folder', () => handler());
}

// relativePath is path with its innermost sidebar root removed. A path outside
// every root is returned unchanged, so the command still copies something
// meaningful.
export function relativePath(path: string, roots: readonly string[]): string {
    let innermost = '';
    for (const candidate of roots) {
        const root = candidate.length > 1 ? candidate.replace(/\/+$/, '') : candidate;
        if (root.length > innermost.length && isWithin(root, path)) {
            innermost = root;
        }
    }
    if (innermost === '') {
        return path;
    }
    return path.slice(innermost.length).replace(/^\/+/, '');
}

// closeAllSidebarFolders drops every root in one window and returns the empty
// list.
export async function closeAllSidebarFolders(window: string): Promise<string[]> {
    return (await RemoveAllSidebarFolders(window)) ?? [];
}

// setSidebarFolders replaces every root in one window. Opening a workspace
// uses it so the sidebar and session.json change together.
export async function setSidebarFolders(
    window: string,
    folders: readonly string[],
): Promise<string[]> {
    return (await SetSidebarFolders(window, folders.slice())) ?? [];
}
