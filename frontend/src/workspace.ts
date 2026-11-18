import type * as models from '../bindings/kaze/models';
import type * as config from '../bindings/kaze/internal/config/models';
import type * as workspacefile from '../bindings/kaze/internal/workspacefile/models';
import {
    AddRecentFile,
    ClearRecentFiles,
    GetRecentFiles,
    GetSettings,
    GetWindowSession,
    LoadWorkspace,
    OpenWorkspaceDialog,
    SaveWorkspaceAs,
    SetSession,
    SetSidebarAutoReveal,
    SetSidebarVisible,
    WatchPaths,
} from '../bindings/kaze/app';
import { onWindowEvent } from './windowstate';

// WorkspaceState is the state shared by every window: the sidebar's visibility
// and reveal behavior from settings.json plus the Open Recent list from
// session.json. Tabs live in session.json too, restored per launch.
export interface WorkspaceState {
    sidebarVisible: boolean;
    autoReveal: boolean;
    recentFiles: string[];
}

// SavedWindowSession is one window's session.json entry, normalized so the rest
// of the frontend never sees nullable arrays.
export interface SavedWindowSession {
    workspace: string;
    folders: string[];
    openFiles: string[];
    active: string;
    expanded: string[];
}

// SavedWorkspaceSession is the tab set parked under one workspace file.
export interface SavedWorkspaceSession {
    folders: string[];
    openFiles: string[];
    active: string;
    expanded: string[];
}

export function toWindowSession(session: models.WindowSession): SavedWindowSession {
    return {
        workspace: session.workspace ?? '',
        folders: session.folders ?? [],
        openFiles: session.open_files ?? [],
        active: session.active ?? '',
        expanded: session.expanded ?? [],
    };
}

export function toWorkspaceSession(session: workspacefile.WorkspaceSession): SavedWorkspaceSession {
    return {
        folders: session.folders ?? [],
        openFiles: session.open_files ?? [],
        active: session.active ?? '',
        expanded: session.expanded ?? [],
    };
}

export async function loadWorkspaceState(): Promise<WorkspaceState> {
    let settings: config.Settings | null = null;
    let recentFiles: string[] = [];
    try {
        settings = await GetSettings();
    } catch (error) {
        console.error('Failed to load workspace state.', error);
    }
    try {
        recentFiles = (await GetRecentFiles()) ?? [];
    } catch (error) {
        console.error('Failed to load recent files.', error);
    }
    return {
        sidebarVisible: settings?.sidebar?.visible ?? true,
        autoReveal: settings?.sidebar?.auto_reveal ?? true,
        recentFiles,
    };
}

// loadWindowSession returns the tabs the first window of a launch should
// restore. Later windows get an empty session from the backend.
export async function loadWindowSession(window: string): Promise<SavedWindowSession> {
    try {
        return toWindowSession(await GetWindowSession(window));
    } catch (error) {
        console.error('Failed to load the window session.', error);
        return { workspace: '', folders: [], openFiles: [], active: '', expanded: [] };
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

// saveSession records this window's file-backed tabs, its workspace binding and
// the folders the tree is showing. The backend also parks the same tab set
// under the workspace, so switching back restores it. Untitled buffers have no
// text on disk and are left out.
export async function saveSession(
    window: string,
    session: SavedWindowSession,
): Promise<void> {
    const payload: models.WindowSession = {
        workspace: session.workspace,
        folders: session.folders.slice(),
        open_files: session.openFiles.slice(),
        active: session.active,
        expanded: session.expanded.slice(),
    };
    await SetSession(window, payload);
}

export async function saveSidebarVisible(visible: boolean): Promise<void> {
    await SetSidebarVisible(visible);
}

export async function saveSidebarAutoReveal(autoReveal: boolean): Promise<void> {
    await SetSidebarAutoReveal(autoReveal);
}

// openWorkspaceDialog asks for a workspace file and returns its resolved
// folders plus the tab set parked under it. An empty path means the picker was
// cancelled.
export async function openWorkspaceDialog(): Promise<workspacefile.WorkspaceInfo> {
    return await OpenWorkspaceDialog();
}

// loadWorkspace reopens the workspace a saved window was showing, without a
// picker. It validates the file and returns the parked session, which may be
// empty.
export async function loadWorkspace(path: string): Promise<workspacefile.WorkspaceInfo> {
    return await LoadWorkspace(path);
}

// saveWorkspaceAs writes the current roots to a workspace file. An empty path
// means the save panel was cancelled.
export async function saveWorkspaceAs(folders: readonly string[]): Promise<workspacefile.WorkspaceInfo> {
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
