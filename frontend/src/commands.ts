import { onWindowEvent } from './windowstate';
import { CASE_TRANSFORMS } from './case';
import {
    INDENT_COMMANDS,
    LINE_COMMANDS,
    SELECTION_COMMANDS,
    VIEW_OPTIONS,
} from './editorcommands';
import type { PaletteItem } from './palette';
import { rankCandidates } from './quickOpen';

// Command is one entry in the command palette. Commands are static metadata;
// App.svelte maps each id to the function that runs it.
export type Command = PaletteItem;

const EDIT_COMMANDS: readonly Command[] = [
    ...CASE_TRANSFORMS.map((transform) => ({
        id: `case-${transform.id}`,
        name: `Transform to ${transform.label}`,
        detail: 'Edit',
    })),
    { id: 'trim-whitespace', name: 'Trim Trailing Whitespace', detail: 'Edit' },
    ...LINE_COMMANDS.map((entry) => ({
        id: entry.id,
        name: entry.label,
        detail: 'Edit',
    })),
    { id: 'change-language-mode', name: 'Change Language Mode…', detail: 'View' },
    { id: 'find', name: 'Find…', detail: 'Edit' },
    { id: 'replace', name: 'Replace…', detail: 'Edit' },
    { id: 'find-next-result', name: 'Find Next Result', detail: 'Edit' },
    { id: 'find-previous-result', name: 'Find Previous Result', detail: 'Edit' },
    { id: 'go-to-line', name: 'Go to Line…', detail: 'Go' },
    { id: 'go-to-symbol', name: 'Go to Symbol in File…', detail: 'Go' },
];

const SELECTION_PALETTE_COMMANDS: readonly Command[] = SELECTION_COMMANDS.map((entry) => ({
    id: entry.id,
    name: entry.label,
    // Toggle Line Comment lives in the Edit menu; the rest are in Selection.
    detail: entry.id === 'toggle-comment' ? 'Edit' : 'Selection',
}));

// INDENT_PALETTE_COMMANDS are the indentation rows both pickers show: the
// command palette and the list the status bar opens. They run Monaco's own
// actions, which change the active buffer only.
export const INDENT_PALETTE_COMMANDS: readonly Command[] = INDENT_COMMANDS.map((entry) => ({
    id: entry.id,
    name: entry.label,
    detail: 'Indentation',
}));

const APPEARANCE_COMMANDS: readonly Command[] = [
    { id: 'appearance-system', name: 'Appearance: System', detail: 'View' },
    { id: 'appearance-light', name: 'Appearance: Light', detail: 'View' },
    { id: 'appearance-dark', name: 'Appearance: Dark', detail: 'View' },
];

const VIEW_COMMANDS: readonly Command[] = [
    { id: 'show-explorer', name: 'Show Explorer', detail: 'View' },
    { id: 'show-changes', name: 'Show Changes', detail: 'View' },
    { id: 'toggle-preview', name: 'Toggle Preview', detail: 'View' },
    { id: 'zoom-in', name: 'Zoom In', detail: 'View' },
    { id: 'zoom-out', name: 'Zoom Out', detail: 'View' },
    { id: 'zoom-reset', name: 'Actual Size', detail: 'View' },
    { id: 'refresh', name: 'Refresh', detail: 'View' },
    { id: 'toggle-sidebar', name: 'Toggle Sidebar', detail: 'View' },
    ...VIEW_OPTIONS.map((option) => ({
        id: `toggle-${option.id}`,
        name: `Toggle ${option.label}`,
        detail: 'View',
    })),
];

// TAB_COMMANDS are the tab navigation entries. The nine Go to Tab commands are
// generated so the menu, the palette and the backend agree on the positions.
const TAB_COMMANDS: readonly Command[] = [
    { id: 'next-tab', name: 'Next Tab', detail: 'Go' },
    { id: 'previous-tab', name: 'Previous Tab', detail: 'Go' },
    { id: 'reopen-closed-tab', name: 'Reopen Closed Tab', detail: 'File' },
    ...Array.from({ length: 9 }, (_, index) => ({
        id: `go-to-tab-${index + 1}`,
        name: index === 8 ? 'Go to Last Tab' : `Go to Tab ${index + 1}`,
        detail: 'Go',
    })),
];

// COMMANDS is the full palette, in the order shown before the user types. The
// detail names the menu the command comes from, so typing "file" or "application"
// narrows the list by category as well as by name.
export const COMMANDS: readonly Command[] = [
    { id: 'new-window', name: 'New Window', detail: 'File' },
    { id: 'open-folder-new-window', name: 'Open Folder in New Window…', detail: 'File' },
    { id: 'new-file', name: 'New File…', detail: 'File' },
    { id: 'new-folder', name: 'New Folder…', detail: 'File' },
    { id: 'new-buffer', name: 'New Text File', detail: 'File' },
    { id: 'open-file', name: 'Open…', detail: 'File' },
    { id: 'open-recent', name: 'Open Recent…', detail: 'File' },
    { id: 'clear-recent', name: 'Clear Recent', detail: 'File' },
    { id: 'open-workspace', name: 'Open Workspace…', detail: 'File' },
    { id: 'switch-workspace', name: 'Open Recent Workspace…', detail: 'File' },
    { id: 'save-workspace-as', name: 'Save Workspace As…', detail: 'File' },
    { id: 'close-workspace', name: 'Close Workspace', detail: 'File' },
    { id: 'save', name: 'Save', detail: 'File' },
    { id: 'save-as', name: 'Save As…', detail: 'File' },
    { id: 'save-all', name: 'Save All', detail: 'File' },
    { id: 'print', name: 'Print…', detail: 'File' },
    { id: 'command-palette', name: 'Command Palette…', detail: 'View' },
    { id: 'quick-open', name: 'Quick Open…', detail: 'Go' },
    { id: 'add-folder', name: 'Add Folder to Sidebar…', detail: 'File' },
    { id: 'close-all-folders', name: 'Remove All Folders from Sidebar', detail: 'File' },
    { id: 'find-in-files', name: 'Find in Files…', detail: 'Edit' },
    { id: 'close-tab', name: 'Close Tab', detail: 'File' },
    { id: 'close-all-tabs', name: 'Close All Tabs', detail: 'File' },
    { id: 'close-window', name: 'Close Window', detail: 'File' },
    ...TAB_COMMANDS,
    { id: 'reveal-file', name: 'Reveal in Finder', detail: 'File' },
    { id: 'copy-file-path', name: 'Copy File Path', detail: 'File' },
    { id: 'copy-relative-path', name: 'Copy Relative Path', detail: 'File' },
    { id: 'copy-file-name', name: 'Copy File Name', detail: 'File' },
    ...EDIT_COMMANDS,
    ...INDENT_PALETTE_COMMANDS,
    ...SELECTION_PALETTE_COMMANDS,
    ...VIEW_COMMANDS,
    ...APPEARANCE_COMMANDS,
    { id: 'minimize-window', name: 'Minimize', detail: 'Window' },
    { id: 'zoom-window', name: 'Zoom', detail: 'Window' },
    { id: 'toggle-full-screen', name: 'Toggle Full Screen', detail: 'Window' },
    { id: 'move-tab-new-window', name: 'Move Tab to New Window', detail: 'Window' },
    { id: 'bring-all-to-front', name: 'Bring All to Front', detail: 'Window' },
    { id: 'settings', name: 'Settings…', detail: 'Settings' },
    { id: 'about', name: 'About Kaze', detail: 'Application' },
    { id: 'open-settings-json', name: 'Open Settings JSON', detail: 'Settings' },
    { id: 'open-default-settings', name: 'Open Default Settings', detail: 'Settings' },
    { id: 'open-themes-folder', name: 'Open Themes Folder', detail: 'Settings' },
    { id: 'open-keybindings', name: 'Open Keybindings…', detail: 'Settings' },
    { id: 'install-command-line-tool', name: 'Install CLI…', detail: 'Application' },
];

// rankCommands orders commands exactly like Quick Open orders files. The
// command list is a fixed menu, so nothing is dropped for length: the palette
// scrolls instead.
export function rankCommands(
    query: string,
    commands: readonly Command[] = COMMANDS,
): Command[] {
    return rankCandidates(query, commands, commands.length);
}

// listenForCommandPalette runs handler when the File menu asks for the palette.
// Returns an unsubscribe function.
export function listenForCommandPalette(handler: () => void): () => void {
    return onWindowEvent('app:command-palette', () => handler());
}

// listenForMenuCommand runs handler with the palette id a menu item sends when
// its command has no event of its own. Returns an unsubscribe function.
export function listenForMenuCommand(handler: (id: string) => void): () => void {
    return onWindowEvent('app:run-command', (data) => {
        if (typeof data === 'string') {
            handler(data);
        }
    });
}
