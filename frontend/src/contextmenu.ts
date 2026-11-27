// ContextMenuItem is one row of the right-click menu. An empty id is a
// separator: it is never clickable and only groups the items around it.
export interface ContextMenuItem {
    id: string;
    label: string;
    disabled?: boolean;
}

// ROOT_LOCKED_COMMANDS are the tree commands that would duplicate, move or
// delete a sidebar root itself. The sidebar keeps the old path, so a root has
// to be removed from the sidebar rather than duplicated, renamed, moved or
// trashed in place.
export const ROOT_LOCKED_COMMANDS: readonly string[] = [
    'cut',
    'rename',
    'duplicate',
    'move',
    'trash',
];

// treeMenuItems is the row menu for a file-tree entry, in display order. The
// groups follow the order the editors people come from use: the row's own
// opener and the Finder hand-off first, then creating beside it, then the
// clipboard, then the three plain-text spellings of its path, then the commands
// that rewrite the entry itself. Creating comes before the clipboard because
// New File is the tree's most used action. A folder also offers Find in Folder
// beside the Finder hand-off. A sidebar root keeps every item but the ones that
// would move or delete it, and gains the command that drops it from the sidebar.
export function treeMenuItems(
    clipboardPath: string,
    isRoot: boolean,
    isDir: boolean,
): ContextMenuItem[] {
    const items: ContextMenuItem[] = [
        { id: 'open', label: 'Open' },
        { id: 'reveal', label: 'Reveal in Finder' },
        ...(isDir ? [{ id: 'find-in-folder', label: 'Find in Folder…' }] : []),
        { id: '', label: '' },
        { id: 'new-file', label: 'New File…' },
        { id: 'new-folder', label: 'New Folder…' },
        { id: '', label: '' },
        { id: 'cut', label: 'Cut' },
        { id: 'copy', label: 'Copy' },
        { id: 'paste', label: 'Paste', disabled: clipboardPath === '' },
        { id: '', label: '' },
        { id: 'copy-name', label: 'Copy Name' },
        { id: 'copy-path', label: 'Copy Path' },
        { id: 'copy-relative-path', label: 'Copy Relative Path' },
        { id: '', label: '' },
        { id: 'rename', label: 'Rename…' },
        { id: 'duplicate', label: 'Duplicate…' },
        { id: 'move', label: 'Move…' },
        { id: 'trash', label: 'Delete' },
    ];
    if (!isRoot) {
        return items;
    }
    items.push(
        { id: '', label: '' },
        { id: 'remove-folder', label: 'Remove Folder from Sidebar' },
    );
    return items.map((item) =>
        ROOT_LOCKED_COMMANDS.includes(item.id) ? { ...item, disabled: true } : item,
    );
}
