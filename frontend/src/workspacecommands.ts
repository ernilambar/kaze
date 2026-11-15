import { Events } from '@wailsio/runtime';

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
        Events.On('kova:open-recent', () => handlers.openRecent?.()),
        Events.On('kova:clear-recent', () => handlers.clearRecent?.()),
        Events.On('kova:open-workspace', () => handlers.openWorkspace?.()),
        Events.On('kova:save-workspace-as', () => handlers.saveWorkspaceAs?.()),
        Events.On('kova:close-workspace', () => handlers.closeWorkspace?.()),
        Events.On('kova:refresh', () => handlers.refresh?.()),
        Events.On('kova:find-in-files', () => handlers.findInFiles?.()),
        Events.On('kova:toggle-sidebar', () => handlers.toggleSidebar?.()),
        Events.On('kova:close-all-folders', () => handlers.closeAllFolders?.()),
        Events.On('kova:reveal-file', () => handlers.revealFile?.()),
        Events.On('kova:copy-path', (event) => {
            const kind = event.data as CopyPathKind | undefined;
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
