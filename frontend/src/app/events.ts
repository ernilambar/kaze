import { listenForCommandPalette, listenForMenuCommand } from '../commands';
import { listenForLaunch, type LaunchRequest } from '../document';
import { listenForEditorCommands, type EditorCommandHandlers } from '../editorcommands';
import {
    listenForCloseRequest,
    listenForFileCommands,
    type CloseReason,
    type FileCommand,
} from '../filecommands';
import { listenForQuickOpen } from '../quickOpen';
import { listenForAddFolder } from '../sidebar';
import { listenForCloseTab, listenForTabCommands } from '../tabs';
import {
    listenForAppearance,
    listenForConfigChanges,
    listenForSettingsOverlay,
    type ConfigChange,
} from '../theme';
import type { ThemeMode } from '../settings';
import { listenForGitChanges, listenForWatchedChanges } from '../workspace';
import { listenForWorkspaceCommands, type WorkspaceCommandHandlers } from '../workspacecommands';

// AppEventHandlers is every menu and backend event a window reacts to, keyed by
// the listener that delivers it.
export interface AppEventHandlers {
    launch: (request: LaunchRequest) => void;
    addFolder: () => void;
    closeTab: () => void;
    quickOpen: () => void;
    commandPalette: () => void;
    menuCommand: (id: string) => void;
    fileCommands: Partial<Record<FileCommand, () => void>>;
    closeRequest: (reason: CloseReason) => void;
    editorCommands: Partial<EditorCommandHandlers>;
    watchedChanges: (paths: string[]) => void;
    gitChanges: () => void;
    workspaceCommands: Partial<WorkspaceCommandHandlers>;
    tabCommands: Parameters<typeof listenForTabCommands>[0];
    configChange: (change: ConfigChange) => void;
    appearance: (mode: ThemeMode) => void;
    settingsOverlay: () => void;
}

// subscribeAll installs every window listener and returns one function that
// removes them all. The launch listener goes first and is awaited: it collects
// the request the window was created with, and nothing else is installed until
// that request has been handed over.
export async function subscribeAll(
    windowName: string,
    handlers: AppEventHandlers,
): Promise<() => void> {
    const offs = [
        await listenForLaunch(windowName, handlers.launch),
        listenForAddFolder(handlers.addFolder),
        listenForCloseTab(handlers.closeTab),
        listenForQuickOpen(handlers.quickOpen),
        listenForCommandPalette(handlers.commandPalette),
        listenForMenuCommand(handlers.menuCommand),
        listenForFileCommands(handlers.fileCommands),
        listenForCloseRequest(handlers.closeRequest),
        listenForEditorCommands(handlers.editorCommands),
        listenForWatchedChanges(handlers.watchedChanges),
        listenForGitChanges(handlers.gitChanges),
        listenForWorkspaceCommands(handlers.workspaceCommands),
        listenForTabCommands(handlers.tabCommands),
        listenForConfigChanges(handlers.configChange),
        listenForAppearance(handlers.appearance),
        listenForSettingsOverlay(handlers.settingsOverlay),
    ];
    return () => {
        for (const off of offs) {
            off();
        }
    };
}
