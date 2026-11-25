import { onWindowEvent } from './windowstate';
import { DIFF_PREFIX, uniqueTabId, type OpenDocument } from './document';

// findDocumentByPath returns the tab that holds the file at path, matching
// either the spelling it was opened with or its symlink-resolved path. An empty
// path never matches, because that is an untitled buffer rather than a file.
export function findDocumentByPath(
    tabs: readonly OpenDocument[],
    path: string,
): OpenDocument | undefined {
    if (path === '') {
        return undefined;
    }
    return tabs.find((tab) => tab.path === path || tab.realPath === path);
}

// tabNameDetails returns the folder detail each tab shows beside its name, so
// two tabs holding files of the same name can be told apart. The detail is the
// shortest trailing directory path that is unique among the tabs sharing that
// name: README.md under docs and under guide shows one folder each, and only a
// deeper tree needs more of the path. A name that is already unique, and a
// buffer with no file behind it, gets no entry.
export function tabNameDetails(tabs: readonly OpenDocument[]): Map<string, string> {
    const details = new Map<string, string>();
    const groups = new Map<string, OpenDocument[]>();
    for (const tab of tabs) {
        const group = groups.get(tab.name);
        if (group === undefined) {
            groups.set(tab.name, [tab]);
        } else {
            group.push(tab);
        }
    }

    for (const group of groups.values()) {
        if (group.length < 2) {
            continue;
        }
        // A diff tab's path is empty; its source is the file it came from.
        const folders = group.map((tab) =>
            (tab.kind === 'diff' ? (tab.sourcePath ?? '') : tab.path)
                .split(/[\\/]/)
                .filter((part) => part !== '')
                .slice(0, -1),
        );
        const deepest = Math.max(...folders.map((segments) => segments.length));
        for (let depth = 1; depth <= deepest; depth++) {
            const labels = folders.map((segments) =>
                (segments.length <= depth ? segments : segments.slice(-depth)).join('/'),
            );
            if (new Set(labels).size === labels.length) {
                group.forEach((tab, index) => {
                    if (labels[index] !== '') {
                        details.set(tab.id, labels[index]);
                    }
                });
                break;
            }
        }
    }
    return details;
}

// openDocumentTab returns tabs with document appended, unless a tab for the
// same file is already open. The file is identified by its resolved path, so a
// path reached through a symlink and the real one share one tab; the id of that
// tab comes back too, so the caller activates it instead of opening a
// duplicate. Untitled buffers have no path, so they are never deduplicated.
export function openDocumentTab(
    tabs: readonly OpenDocument[],
    document: OpenDocument,
): { tabs: OpenDocument[]; activeId: string } {
    const existing = findDocumentByPath(
        tabs,
        document.path === '' ? '' : document.realPath || document.path,
    );
    if (existing !== undefined) {
        return { tabs: tabs.slice(), activeId: existing.id };
    }
    // An id comes from a path or name, but a renamed or saved-as tab keeps its
    // old one; a new tab must never take it.
    const id = uniqueTabId(tabs, document.id);
    const added = id === document.id ? document : { ...document, id };
    return { tabs: [...tabs, added], activeId: id };
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

// TabCloseScope names the group a tab context menu closes. "others" and
// "right" are relative to the right-clicked tab; "all" and "deleted" ignore
// it, which is why the target id is unused for those scopes.
export type TabCloseScope = 'others' | 'right' | 'all' | 'deleted';

// tabsToClose returns the tabs a context menu command closes, in their current
// order. An unknown target means there is nothing to close for the scopes that
// depend on it, while "all" and "deleted" still describe the tab set.
export function tabsToClose(
    tabs: readonly OpenDocument[],
    id: string,
    scope: TabCloseScope,
): OpenDocument[] {
    const index = tabs.findIndex((tab) => tab.id === id);
    switch (scope) {
        case 'others':
            return index === -1 ? [] : tabs.filter((_, position) => position !== index);
        case 'right':
            return index === -1 ? [] : tabs.slice(index + 1);
        case 'all':
            return tabs.slice();
        case 'deleted':
            return tabs.filter((tab) => tab.deleted);
    }
}

// moveDocumentTab drags id next to targetId, before it or after it. The array
// comes back unchanged (same identity) when the drop would not move anything,
// so callers can skip the session write a no-op reorder would cause.
export function moveDocumentTab(
    tabs: readonly OpenDocument[],
    id: string,
    targetId: string,
    placeAfter: boolean,
): OpenDocument[] {
    if (id === targetId) {
        return tabs as OpenDocument[];
    }
    const from = tabs.findIndex((tab) => tab.id === id);
    if (from === -1) {
        return tabs as OpenDocument[];
    }
    const next = tabs.slice();
    const [moved] = next.splice(from, 1);
    const target = next.findIndex((tab) => tab.id === targetId);
    if (target === -1) {
        return tabs as OpenDocument[];
    }
    const to = target + (placeAfter ? 1 : 0);
    if (to === from) {
        return tabs as OpenDocument[];
    }
    next.splice(to, 0, moved);
    return next;
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
// sees in the tab bar. Position 9 is always the last tab, as in Safari.
export function tabIdAt(tabs: readonly OpenDocument[], position: number): string {
    const index = position === 9 ? tabs.length : position;
    if (index < 1 || index > tabs.length) {
        return '';
    }
    return tabs[index - 1].id;
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

// retargetDocumentPath is the tab after its file moved on disk. Renaming or
// moving a file keeps the same buffer open under its new path: the tab's text,
// edits and undo history stay, so only its identity is rewritten and its id
// travels with the buffer. A diff tab follows the file it is a view of instead
// of pointing at a path that no longer exists.
export function retargetDocumentPath(
    document: OpenDocument,
    from: string,
    to: string,
): OpenDocument | undefined {
    if (from === '' || to === '' || from === to) {
        return undefined;
    }
    if (document.kind === 'diff') {
        const source = document.sourcePath ?? '';
        const moved = remapPath(source, from, to);
        if (moved === '' || moved === source) {
            return undefined;
        }
        const name = baseName(moved);
        return {
            ...document,
            id: `${DIFF_PREFIX}${moved}`,
            name: `${name} (Diff)`,
            sourcePath: moved,
        };
    }
    if (document.path === '') {
        return undefined;
    }

    // Either spelling can match: a tab remembers the path it was opened with
    // and the same file with its symlinks resolved, while Go reports whichever
    // one the command ran on.
    const path = remapPath(document.path, from, to) || remapPath(document.realPath, from, to);
    if (path === '') {
        return undefined;
    }
    return { ...document, path, realPath: path, name: baseName(path) };
}

// retargetDocumentPaths applies every move in one pass, so a tab whose file
// moved twice during one command ends up where it belongs. The list comes back
// with the same identity when nothing moved, so callers can skip the work a
// rewrite would cause.
export function retargetDocumentPaths(
    tabs: readonly OpenDocument[],
    moves: ReadonlyArray<{ from: string; to: string }>,
): OpenDocument[] {
    if (moves.length === 0) {
        return tabs as OpenDocument[];
    }
    let changed = false;
    const next = tabs.map((tab) => {
        let current = tab;
        for (const move of moves) {
            const moved = retargetDocumentPath(current, move.from, move.to);
            if (moved !== undefined) {
                current = moved;
                changed = true;
            }
        }
        return current;
    });
    return changed ? next : (tabs as OpenDocument[]);
}

// retargetPaths rewrites the paths a move from `from` to `to` carried along,
// dropping any that land on an entry already in the list. The list comes back
// with the same identity when nothing moved.
export function retargetPaths(paths: readonly string[], from: string, to: string): string[] {
    if (from === '' || to === '' || from === to || !paths.some((path) => pathWithin(path, from))) {
        return paths as string[];
    }
    const moved = paths.map((path) => remapPath(path, from, to) || path);
    return [...new Set(moved)];
}

// pathWithin reports whether path is base itself or sits inside it, so a
// rename of a folder retargets the tabs of the files under it.
export function pathWithin(path: string, base: string): boolean {
    if (path === '' || base === '') {
        return false;
    }
    const candidate = normalizeSeparators(path);
    const root = normalizeSeparators(base).replace(/\/+$/, '');
    return candidate === root || candidate.startsWith(`${root}/`);
}

// remapPath is path rewritten from base to target, or the empty string when
// path is not under base. Both separators are accepted, because a path the
// backend reported and the one a tab was opened with may spell the same file
// differently.
function remapPath(path: string, base: string, target: string): string {
    if (!pathWithin(path, base)) {
        return '';
    }
    const root = normalizeSeparators(base).replace(/\/+$/, '');
    const suffix = normalizeSeparators(path).slice(root.length);
    return `${target}${suffix}`;
}

function normalizeSeparators(value: string): string {
    return value.replace(/\\/g, '/');
}

function baseName(path: string): string {
    const parts = path.split(/[\\/]/).filter((part) => part !== '');
    return parts.length === 0 ? path : parts[parts.length - 1];
}
