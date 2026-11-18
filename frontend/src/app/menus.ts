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
// offers the same items whether it is a file or a folder. An unknown tab means
// nothing to show. A separator is an item with an empty id, which the menu draws
// as a line.
export function contextMenuItems(
    target: MenuTarget | null,
    tabs: readonly OpenDocument[],
    clipboardPath: string,
    searchOpen: boolean,
): ContextMenuItem[] {
    if (target === null) {
        return [];
    }
    if (target.tabId !== undefined) {
        const tab = tabs.find((entry) => entry.id === target.tabId);
        return tab === undefined ? [] : documentTabMenuItems(tab, tabs, searchOpen);
    }
    return treeMenuItems(clipboardPath);
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
): ContextMenuItem[] {
    const reveal = tab.path || tab.sourcePath || '';
    const index = tabs.findIndex((entry) => entry.id === tab.id);
    const items: ContextMenuItem[] = [
        { id: 'close-tab', label: 'Close Tab' },
        { id: 'move-tab-new-window', label: 'Move to New Window' },
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
    return items;
}
