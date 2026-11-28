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
    shortcut: ShortcutLookup = () => '',
): ContextMenuItem[] {
    if (target === null) {
        return [];
    }
    if (target.tabId !== undefined) {
        const tab = tabs.find((entry) => entry.id === target.tabId);
        return tab === undefined ? [] : documentTabMenuItems(tab, tabs, searchOpen, configFiles, shortcut);
    }
    return withShortcuts(
        treeMenuItems(clipboardPath, target.isDir && roots.includes(target.path), target.isDir),
        TREE_ITEM_COMMANDS,
        shortcut,
    );
}

// ShortcutLookup gives the formatted key that runs a command id, or ''.
export type ShortcutLookup = (command: string) => string;

// TAB_ITEM_COMMANDS and TREE_ITEM_COMMANDS name the command whose key does the
// same job as a row item, so the menu can show it. The key acts on the active
// tab, as in VS Code, while the item acts on the row under the pointer.
const TAB_ITEM_COMMANDS: Readonly<Record<string, string>> = {
    'close-tab': 'close-tab',
    'close-all-tabs': 'close-all-tabs',
    reveal: 'reveal-file',
    'pin-tab': 'toggle-pin-tab',
    'move-tab-new-window': 'move-tab-new-window',
};

const TREE_ITEM_COMMANDS: Readonly<Record<string, string>> = {
    reveal: 'reveal-file',
    'new-file': 'new-file',
    'new-folder': 'new-folder',
    'copy-name': 'copy-file-name',
    'copy-path': 'copy-file-path',
    'copy-relative-path': 'copy-relative-path',
};

// withShortcuts adds the key hint to every item whose command has one.
function withShortcuts(
    items: ContextMenuItem[],
    commands: Readonly<Record<string, string>>,
    shortcut: ShortcutLookup,
): ContextMenuItem[] {
    return items.map((item) => {
        const command = commands[item.id];
        const keys = command === undefined ? '' : shortcut(command);
        return keys === '' ? item : { ...item, shortcut: keys };
    });
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
    shortcut: ShortcutLookup = () => '',
): ContextMenuItem[] {
    const reveal = tab.path || tab.sourcePath || '';
    const index = tabs.findIndex((entry) => entry.id === tab.id);
    // Bulk closes skip pinned tabs, so they only count the tabs they can close.
    const others = tabs.filter((entry) => entry.id !== tab.id && !entry.pinned).length;
    const toRight = index === -1 ? 0 : tabs.slice(index + 1).filter((entry) => !entry.pinned).length;
    const unpinned = tabs.filter((entry) => !entry.pinned).length;
    const items: ContextMenuItem[] = [
        { id: 'close-tab', label: 'Close Tab' },
        { id: '', label: '' },
        {
            id: 'close-other-tabs',
            label: 'Close Other Tabs',
            disabled: others === 0 && !searchOpen,
        },
        {
            id: 'close-right-tabs',
            label: 'Close Tabs to Right',
            disabled: index === -1 || (toRight === 0 && !searchOpen),
        },
        {
            id: 'close-deleted-tabs',
            label: 'Close Tabs with Deleted Files',
            disabled: !tabs.some((entry) => entry.deleted && !entry.pinned),
        },
        { id: '', label: '' },
        {
            id: 'close-all-tabs',
            label: 'Close All',
            disabled: unpinned === 0 && !searchOpen,
        },
    ];
    items.push({ id: '', label: '' });
    if (reveal !== '') {
        items.push({ id: 'reveal', label: 'Reveal in Finder' });
    }
    items.push({ id: 'pin-tab', label: tab.pinned ? 'Unpin' : 'Pin' });
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
    // Cmd+W does not close a pinned tab, so its Close Tab shows no key.
    const lookup: ShortcutLookup = tab.pinned
        ? (command) => (command === 'close-tab' ? '' : shortcut(command))
        : shortcut;
    return withShortcuts(items, TAB_ITEM_COMMANDS, lookup);
}
