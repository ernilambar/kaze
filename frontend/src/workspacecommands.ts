import { onWindowEvent } from './windowstate';

// CopyPathKind is what "Copy Path" copies: the absolute path, the path relative
// to the innermost sidebar root, or the bare file name.
export type CopyPathKind = 'file' | 'relative' | 'name';

export interface WorkspaceCommandHandlers {
    openRecent: () => void;
    clearRecent: () => void;
    openWorkspace: () => void;
    saveWorkspaceAs: () => void;
    closeWorkspace: () => void;
    refresh: () => void;
    findInFiles: () => void;
    toggleSidebar: () => void;
    closeAllFolders: () => void;
    revealFile: () => void;
    copyPath: (kind: CopyPathKind) => void;
}

// listenForWorkspaceCommands runs the handler for each File or View menu
// command that acts on the workspace rather than on one buffer. Returns one
// function that unsubscribes from all of them.
export function listenForWorkspaceCommands(
    handlers: Partial<WorkspaceCommandHandlers>,
): () => void {
    const offs = [
        onWindowEvent('app:open-recent', () => handlers.openRecent?.()),
        onWindowEvent('app:clear-recent', () => handlers.clearRecent?.()),
        onWindowEvent('app:open-workspace', () => handlers.openWorkspace?.()),
        onWindowEvent('app:save-workspace-as', () => handlers.saveWorkspaceAs?.()),
        onWindowEvent('app:close-workspace', () => handlers.closeWorkspace?.()),
        onWindowEvent('app:refresh', () => handlers.refresh?.()),
        onWindowEvent('app:find-in-files', () => handlers.findInFiles?.()),
        onWindowEvent('app:toggle-sidebar', () => handlers.toggleSidebar?.()),
        onWindowEvent('app:close-all-folders', () => handlers.closeAllFolders?.()),
        onWindowEvent('app:reveal-file', () => handlers.revealFile?.()),
        onWindowEvent('app:copy-path', (data) => {
            const kind = data as CopyPathKind | undefined;
            if (kind !== undefined) {
                handlers.copyPath?.(kind);
            }
        }),
    ];
    return () => {
        for (const off of offs) {
            off();
        }
    };
}
