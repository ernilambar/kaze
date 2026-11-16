import type * as models from '../bindings/kova/models';
import {
    AddRecentFile,
    ClearRecentFiles,
    GetSettings,
    OpenWorkspaceDialog,
    SaveWorkspaceAs,
    SetSession,
    SetSidebarAutoReveal,
    SetSidebarVisible,
    WatchPaths,
} from '../bindings/kova/app';
import { onWindowEvent } from './windowstate';

// WorkspaceState is the part of settings.json that outlives a window: the
// sidebar's visibility and reveal behavior, the Open Recent list and the tabs
// to restore.
export interface WorkspaceState {
    sidebarVisible: boolean;
    autoReveal: boolean;
    recentFiles: string[];
    session: {
        openFiles: string[];
        active: string;
    };
}

export async function loadWorkspaceState(): Promise<WorkspaceState> {
    try {
        const settings = await GetSettings();
        return {
            sidebarVisible: settings?.sidebar?.visible ?? true,
            autoReveal: settings?.sidebar?.auto_reveal ?? true,
            recentFiles: settings?.recent?.files ?? [],
            session: {
                openFiles: settings?.session?.open_files ?? [],
                active: settings?.session?.active ?? '',
            },
        };
    } catch (error) {
        console.error('Failed to load workspace state.', error);
        return {
            sidebarVisible: true,
            autoReveal: true,
            recentFiles: [],
            session: { openFiles: [], active: '' },
        };
    }
}

// rememberRecentFile puts path at the top of the recent list and returns the
// new list, which the palette shows in order.
export async function rememberRecentFile(path: string): Promise<string[]> {
    return (await AddRecentFile(path)) ?? [];
}

export async function clearRecentFiles(): Promise<string[]> {
    return (await ClearRecentFiles()) ?? [];
}

// saveSession records the file-backed tabs and the active one. Untitled
// buffers have no content on disk, so they are left out of the session.
export async function saveSession(openFiles: readonly string[], active: string): Promise<void> {
    const session: models.SessionState = { open_files: openFiles.slice(), active };
    await SetSession(session);
}

export async function saveSidebarVisible(visible: boolean): Promise<void> {
    await SetSidebarVisible(visible);
}

export async function saveSidebarAutoReveal(autoReveal: boolean): Promise<void> {
    await SetSidebarAutoReveal(autoReveal);
}

// openWorkspaceDialog asks for a workspace file and returns its resolved
// folders. An empty path means the picker was cancelled.
export async function openWorkspaceDialog(): Promise<models.WorkspaceInfo> {
    return await OpenWorkspaceDialog();
}

// saveWorkspaceAs writes the current roots to a workspace file. An empty path
// means the save panel was cancelled.
export async function saveWorkspaceAs(folders: readonly string[]): Promise<models.WorkspaceInfo> {
    return await SaveWorkspaceAs(folders.slice());
}

// watchPaths tells Go which files and folders one window wants polled for
// external changes.
export async function watchPaths(windowName: string, paths: readonly string[]): Promise<void> {
    await WatchPaths(windowName, paths.slice());
}

// listenForWatchedChanges runs handler with the paths that changed on disk for
// this window. Returns an unsubscribe function.
export function listenForWatchedChanges(handler: (paths: string[]) => void): () => void {
    return onWindowEvent('app:watched-changed', (data) => {
        const paths = data as string[] | undefined;
        if (paths !== undefined) {
            handler(paths);
        }
    });
}
