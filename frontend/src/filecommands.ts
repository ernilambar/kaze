import { onWindowEvent } from './windowstate';

// FileCommand is every File menu item that acts on a buffer. Go only forwards
// the click: buffers, their paths and their modified flags are frontend state,
// so the work happens here.
export type FileCommand =
    | 'new-file'
    | 'new-buffer'
    | 'open-file'
    | 'save'
    | 'save-as'
    | 'save-all'
    | 'print';

const FILE_COMMANDS: readonly FileCommand[] = [
    'new-file',
    'new-buffer',
    'open-file',
    'save',
    'save-as',
    'save-all',
    'print',
];

// listenForFileCommands runs the handler for each command the File menu sends
// and returns one function that unsubscribes from all of them.
export function listenForFileCommands(handlers: Partial<Record<FileCommand, () => void>>): () => void {
    const offs = FILE_COMMANDS.map((command) =>
        onWindowEvent(`app:${command}`, () => handlers[command]?.()),
    );
    return () => {
        for (const off of offs) {
            off();
        }
    };
}

// CloseReason is why a close prompt was raised: the window's own close button
// ("close") or an app quit ("quit"). The prompt's wording differs.
export type CloseReason = 'quit' | 'close';

// listenForCloseRequest runs handler when a window or the app is closing with
// unsaved buffers. The frontend answers with ResolveClose, because only it can
// save them.
export function listenForCloseRequest(handler: (reason: CloseReason) => void): () => void {
    return onWindowEvent('app:confirm-close', (data) => {
        const prompt = data as { reason?: string } | undefined;
        handler(prompt?.reason === 'close' ? 'close' : 'quit');
    });
}
