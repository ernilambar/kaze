import { Events } from '@wailsio/runtime';
import type { OpenDocument } from './document';

// openDocumentTab returns tabs with document appended, unless a tab for the
// same path is already open. A new array is returned so Svelte sees a change.
export function openDocumentTab(
    tabs: readonly OpenDocument[],
    document: OpenDocument,
): OpenDocument[] {
    if (tabs.some((tab) => tab.path === document.path)) {
        return tabs.slice();
    }
    return [...tabs, document];
}

// closeDocumentTab removes path and picks the next active tab. Closing the
// active tab activates its right neighbour, then the left one, and finally
// leaves no tab open.
export function closeDocumentTab(
    tabs: readonly OpenDocument[],
    path: string,
    activePath: string,
): { tabs: OpenDocument[]; activePath: string } {
    const index = tabs.findIndex((tab) => tab.path === path);
    if (index === -1) {
        return { tabs: tabs.slice(), activePath };
    }

    const remaining = tabs.filter((tab) => tab.path !== path);
    if (activePath !== path) {
        return { tabs: remaining, activePath };
    }

    const next = remaining[index] ?? remaining[index - 1];
    return { tabs: remaining, activePath: next?.path ?? '' };
}

// listenForCloseTab runs handler when the File menu asks to close the active
// tab. Returns an unsubscribe function.
export function listenForCloseTab(handler: () => void): () => void {
    return Events.On('kova:close-tab', () => handler());
}
