import { treeMenuItems, type ContextMenuItem } from '../contextmenu';
import type { OpenDocument } from '../document';

// MenuTarget is where a right-click landed: a tab, or a tree row anchored at
// the pointer. isDir decides which items the tree offers; tabId is absent for a
// tree row.
export interface MenuTarget {
    path: string;
    isDir: boolean;
    tabId?: string;
}

// contextMenuItems builds the row menu for one right-click. A tab row offers the
// tab commands; a tree row creates in the folder it lives in, which is why it
// offers the same items whether it is a file or a folder, apart from Find in
// Folder; a sidebar root (one of roots) cannot be moved or deleted from it. An unknown tab means nothing to
// show. A separator is an item with an empty id, which the menu draws
// as a line.
export function contextMenuItems(
    target: MenuTarget | null,
    tabs: readonly OpenDocument[],
    clipboardPath: string,
    searchOpen: boolean,
    roots: readonly string[],
    configFiles: readonly string[] = [],
): ContextMenuItem[] {
    if (target === null) {
        return [];
    }
    if (target.tabId !== undefined) {
        const tab = tabs.find((entry) => entry.id === target.tabId);
        return tab === undefined ? [] : documentTabMenuItems(tab, tabs, searchOpen, configFiles);
    }
    return treeMenuItems(
        clipboardPath,
        target.isDir && roots.includes(target.path),
        target.isDir,
    );
}

// ContextMenuRoute is the controller a picked menu item belongs to. A tab item
// runs against the tab it was opened on, a sidebar root is removed by the
// workspace, and every other tree item is a file command. null means the item
// must not run at all.
export type ContextMenuRoute =
    | { kind: 'tab'; id: string; tabId: string }
    | { kind: 'remove-folder'; path: string }
    | { kind: 'tree'; id: string; path: string; isDir: boolean }
    | null;

// contextMenuRoute decides where a picked row menu item goes. Remove Folder
// from Sidebar is only offered for a root and is the workspace's command, so it
// must never fall through to the file commands that rewrite a path.
export function contextMenuRoute(
    id: string,
    target: MenuTarget | null,
    roots: readonly string[],
): ContextMenuRoute {
    if (target === null) {
        return null;
    }
    if (target.tabId !== undefined) {
        return { kind: 'tab', id, tabId: target.tabId };
    }
    if (id === 'remove-folder') {
        return roots.includes(target.path) ? { kind: 'remove-folder', path: target.path } : null;
    }
    return { kind: 'tree', id, path: target.path, isDir: target.isDir };
}

// documentTabMenuItems is the menu for the tab under the pointer. The commands
// are relative to that tab, not to whichever tab happens to be active, so the
// target's own row decisions are made here rather than by the window. The Find
// in Files tab is rendered after the document tabs, so closing "others" or "to
// the right" also closes it, which is why its presence keeps those commands
// enabled.
export function documentTabMenuItems(
    tab: OpenDocument,
    tabs: readonly OpenDocument[],
    searchOpen: boolean,
    configFiles: readonly string[] = [],
): ContextMenuItem[] {
    const reveal = tab.path || tab.sourcePath || '';
    const index = tabs.findIndex((entry) => entry.id === tab.id);
    const items: ContextMenuItem[] = [
        { id: 'close-tab', label: 'Close Tab' },
        { id: '', label: '' },
        {
            id: 'close-other-tabs',
            label: 'Close Other Tabs',
            disabled: tabs.length === 1 && !searchOpen,
        },
        {
            id: 'close-right-tabs',
            label: 'Close Tabs to Right',
            disabled: index === -1 || (index === tabs.length - 1 && !searchOpen),
        },
        {
            id: 'close-deleted-tabs',
            label: 'Close Tabs with Deleted Files',
            disabled: !tabs.some((entry) => entry.deleted),
        },
        { id: '', label: '' },
        {
            id: 'close-all-tabs',
            label: 'Close All',
            disabled: tabs.length === 0 && !searchOpen,
        },
    ];
    if (reveal !== '') {
        items.push(
            { id: '', label: '' },
            { id: 'reveal', label: 'Reveal in Finder' },
        );
    }
    // The file commands need a file on disk: an untitled buffer and a diff
    // snapshot have none, and a deleted file has nothing left to act on. The
    // settings and keybindings files are read at fixed paths, so they stay put.
    if (tab.path !== '') {
        const disabled =
            tab.deleted ||
            configFiles.some((path) => path === tab.path || path === tab.realPath);
        items.push(
            { id: '', label: '' },
            { id: 'rename', label: 'Rename…', disabled },
            { id: 'move', label: 'Move…', disabled },
            { id: 'duplicate', label: 'Duplicate…', disabled },
            { id: 'trash', label: 'Delete', disabled },
        );
    }
    items.push(
        { id: '', label: '' },
        { id: 'move-tab-new-window', label: 'Move to New Window' },
    );
    return items;
}
