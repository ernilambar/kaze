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
        onWindowEvent('kova:open-recent', () => handlers.openRecent?.()),
        onWindowEvent('kova:clear-recent', () => handlers.clearRecent?.()),
        onWindowEvent('kova:open-workspace', () => handlers.openWorkspace?.()),
        onWindowEvent('kova:save-workspace-as', () => handlers.saveWorkspaceAs?.()),
        onWindowEvent('kova:close-workspace', () => handlers.closeWorkspace?.()),
        onWindowEvent('kova:refresh', () => handlers.refresh?.()),
        onWindowEvent('kova:find-in-files', () => handlers.findInFiles?.()),
        onWindowEvent('kova:toggle-sidebar', () => handlers.toggleSidebar?.()),
        onWindowEvent('kova:close-all-folders', () => handlers.closeAllFolders?.()),
        onWindowEvent('kova:reveal-file', () => handlers.revealFile?.()),
        onWindowEvent('kova:copy-path', (data) => {
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
