import { Events } from '@wailsio/runtime';

// FileCommand is every File menu item that acts on a buffer. Go only forwards
// the click: buffers, their paths and their modified flags are frontend state,
// so the work happens here.
export type FileCommand = 'new-file' | 'open-file' | 'save' | 'save-as' | 'save-all' | 'print';

const FILE_COMMANDS: readonly FileCommand[] = [
    'new-file',
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
        Events.On(`kova:${command}`, () => handlers[command]?.()),
    );
    return () => {
        for (const off of offs) {
            off();
        }
    };
}

// listenForCloseRequest runs handler when the window or the app is closing with
// unsaved buffers. The frontend answers with ResolveClose, because only it can
// save them.
export function listenForCloseRequest(handler: () => void): () => void {
    return Events.On('kova:confirm-close', () => handler());
}
