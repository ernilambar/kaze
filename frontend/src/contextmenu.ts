// ContextMenuItem is one row of the right-click menu. An empty id is a
// separator: it is never clickable and only groups the items around it.
export interface ContextMenuItem {
    id: string;
    label: string;
    disabled?: boolean;
}

// treeMenuItems is the row menu for a file-tree entry, in display order. The
// groups follow the order the editors people come from use: the row's own
// opener and the Finder hand-off first, then creating beside it, then the
// clipboard, then the three plain-text spellings of its path, then the commands
// that rewrite the entry itself. Creating comes before the clipboard because
// New File is the tree's most used action.
export function treeMenuItems(clipboardPath: string): ContextMenuItem[] {
    return [
        { id: 'open', label: 'Open' },
        { id: 'reveal', label: 'Reveal in Finder' },
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
}
