import { onWindowEvent } from './windowstate';
import type { OpenDocument } from './document';

// openDocumentTab returns tabs with document appended, unless a tab for the
// same file is already open. The id of the tab that now holds the document
// comes back too, so the caller activates the existing tab instead of opening a
// duplicate; untitled buffers have no path, so they are never deduplicated.
export function openDocumentTab(
    tabs: readonly OpenDocument[],
    document: OpenDocument,
): { tabs: OpenDocument[]; activeId: string } {
    if (document.path !== '') {
        const existing = tabs.find((tab) => tab.path === document.path);
        if (existing !== undefined) {
            return { tabs: tabs.slice(), activeId: existing.id };
        }
    }
    return { tabs: [...tabs, document], activeId: document.id };
}

// closeDocumentTab removes id and picks the next active tab. Closing the active
// tab activates its right neighbour, then the left one, and finally leaves no
// tab open.
export function closeDocumentTab(
    tabs: readonly OpenDocument[],
    id: string,
    activeId: string,
): { tabs: OpenDocument[]; activeId: string } {
    const index = tabs.findIndex((tab) => tab.id === id);
    if (index === -1) {
        return { tabs: tabs.slice(), activeId };
    }

    const remaining = tabs.filter((tab) => tab.id !== id);
    if (activeId !== id) {
        return { tabs: remaining, activeId };
    }

    const next = remaining[index] ?? remaining[index - 1];
    return { tabs: remaining, activeId: next?.id ?? '' };
}

// updateDocument applies patch to one tab. The returned array is new only when
// a tab exists to change, so callers can rely on identity to skip work.
export function updateDocument(
    tabs: OpenDocument[],
    id: string,
    patch: Partial<OpenDocument>,
): OpenDocument[] {
    const index = tabs.findIndex((tab) => tab.id === id);
    if (index === -1) {
        return tabs;
    }
    const next = tabs.slice();
    next[index] = { ...tabs[index], ...patch };
    return next;
}

// setDocumentDirty records the editor's unmodified/modified state. Nothing
// changes when the flag already matches, so an edit undone back to the saved
// text does not churn the tab list.
export function setDocumentDirty(tabs: OpenDocument[], id: string, dirty: boolean): OpenDocument[] {
    const index = tabs.findIndex((tab) => tab.id === id);
    if (index === -1 || tabs[index].dirty === dirty) {
        return tabs;
    }
    const next = tabs.slice();
    next[index] = { ...tabs[index], dirty };
    return next;
}

// listenForCloseTab runs handler when the File menu asks to close the active
// tab. Returns an unsubscribe function.
export function listenForCloseTab(handler: () => void): () => void {
    return onWindowEvent('app:close-tab', () => handler());
}

// stepTabId returns the tab a next/previous command activates, wrapping at
// either end. When nothing is active the first tab wins, so the command always
// does something visible.
export function stepTabId(
    tabs: readonly OpenDocument[],
    activeId: string,
    direction: 1 | -1,
): string {
    if (tabs.length === 0) {
        return activeId;
    }
    const index = tabs.findIndex((tab) => tab.id === activeId);
    if (index === -1) {
        return tabs[0].id;
    }
    const next = (index + direction + tabs.length) % tabs.length;
    return tabs[next].id;
}

// tabIdAt returns the tab at a 1-based position, or an empty id when the list
// is shorter. The Go menu numbers its items from one, matching what the user
// sees in the tab bar.
export function tabIdAt(tabs: readonly OpenDocument[], position: number): string {
    if (position < 1 || position > tabs.length) {
        return '';
    }
    return tabs[position - 1].id;
}

// listenForTabCommands runs the handler for the tab navigation menu items.
// Returns an unsubscribe function.
export function listenForTabCommands(handlers: {
    next: () => void;
    previous: () => void;
    reopen: () => void;
    goToTab: (position: number) => void;
    moveToNewWindow: () => void;
}): () => void {
    const offs = [
        onWindowEvent('app:next-tab', () => handlers.next()),
        onWindowEvent('app:previous-tab', () => handlers.previous()),
        onWindowEvent('app:reopen-closed-tab', () => handlers.reopen()),
        onWindowEvent('app:move-tab-to-new-window', () => handlers.moveToNewWindow()),
        onWindowEvent('app:go-to-tab', (data) => {
            if (typeof data === 'number') {
                handlers.goToTab(data);
            }
        }),
    ];
    return () => {
        for (const off of offs) {
            off();
        }
    };
}
