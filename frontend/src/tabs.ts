import { Events } from '@wailsio/runtime';
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
    return Events.On('kova:close-tab', () => handler());
}
