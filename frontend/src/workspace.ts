import type * as models from '../bindings/kova/models';
import {
    AddRecentFile,
    ClearRecentFiles,
    GetSettings,
    SetSession,
    SetSidebarVisible,
    WatchPaths,
} from '../bindings/kova/app';
import { Events } from '@wailsio/runtime';

// WorkspaceState is the part of settings.json that outlives a window: the
// sidebar's visibility, the Open Recent list and the tabs to restore.
export interface WorkspaceState {
    sidebarVisible: boolean;
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

// watchPaths tells Go which files and folders to poll for external changes.
export async function watchPaths(paths: readonly string[]): Promise<void> {
    await WatchPaths(paths.slice());
}

// listenForWatchedChanges runs handler with the paths that changed on disk.
// Returns an unsubscribe function.
export function listenForWatchedChanges(handler: (paths: string[]) => void): () => void {
    return Events.On('kova:watched-changed', (event) => {
        const paths = event.data as string[] | undefined;
        if (paths !== undefined) {
            handler(paths);
        }
    });
}
