import { Events, Window } from '@wailsio/runtime';

// The Wails window name is the identity every backend binding and event uses to
// tell one editor window from another. It is read once at startup, before any
// listener is installed.
let name = '';

export function setWindowName(value: string): void {
    name = value;
}

export function currentWindowName(): string {
    return name;
}

// resolveWindowName asks the Wails runtime for this window's name. The runtime
// resolves an unnamed call against the calling webview, so each window gets its
// own answer.
export async function resolveWindowName(): Promise<string> {
    try {
        const resolved = await Window.Name();
        name = resolved;
        return resolved;
    } catch (error) {
        console.error('Failed to read the window name.', error);
        return '';
    }
}

// onWindowEvent listens for a menu or backend event addressed to this window.
// Wails broadcasts every custom event to every window and stamps the sender, so
// a listener ignores events meant for another window. Application-wide events
// (themes, keybindings, shared settings) carry no sender and reach all windows.
export function onWindowEvent(
    eventName: string,
    handler: (data: unknown, sender: string) => void,
): () => void {
    return Events.On(eventName, (event) => {
        const sender = event.sender ?? '';
        if (sender !== '' && sender !== name) {
            return;
        }
        handler(event.data, sender);
    });
}
