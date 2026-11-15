import { Events } from '@wailsio/runtime';
import { CASE_TRANSFORMS } from './case';
import { VIEW_OPTIONS } from './editorcommands';
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
    { id: 'next-tab', name: 'Next Tab', detail: 'File' },
    { id: 'previous-tab', name: 'Previous Tab', detail: 'File' },
    { id: 'reopen-closed-tab', name: 'Reopen Closed Tab', detail: 'File' },
    ...Array.from({ length: 9 }, (_, index) => ({
        id: `go-to-tab-${index + 1}`,
        name: `Go to Tab ${index + 1}`,
        detail: 'File',
    })),
];

// COMMANDS is the full palette, in the order shown before the user types. The
// detail names the menu the command comes from, so typing "file" or "kova"
// narrows the list by category as well as by name.
export const COMMANDS: readonly Command[] = [
    { id: 'new-file', name: 'New File', detail: 'File' },
    { id: 'open-file', name: 'Open…', detail: 'File' },
    { id: 'open-recent', name: 'Open Recent…', detail: 'File' },
    { id: 'clear-recent', name: 'Clear Recent', detail: 'File' },
    { id: 'save', name: 'Save', detail: 'File' },
    { id: 'save-as', name: 'Save As…', detail: 'File' },
    { id: 'save-all', name: 'Save All', detail: 'File' },
    { id: 'print', name: 'Print…', detail: 'File' },
    { id: 'quick-open', name: 'Quick Open…', detail: 'File' },
    { id: 'add-folder', name: 'Add Folder to Sidebar…', detail: 'File' },
    { id: 'close-all-folders', name: 'Close All Folders', detail: 'File' },
    { id: 'find-in-files', name: 'Find in Files…', detail: 'File' },
    { id: 'close-tab', name: 'Close Tab', detail: 'File' },
    ...TAB_COMMANDS,
    { id: 'reveal-file', name: 'Reveal in Finder', detail: 'File' },
    { id: 'copy-file-path', name: 'Copy File Path', detail: 'File' },
    { id: 'copy-relative-path', name: 'Copy Relative Path', detail: 'File' },
    { id: 'copy-file-name', name: 'Copy File Name', detail: 'File' },
    ...EDIT_COMMANDS,
    ...VIEW_COMMANDS,
    { id: 'settings', name: 'Settings…', detail: 'Kova' },
    { id: 'install-command-line-tool', name: 'Install Command Line Tool…', detail: 'Kova' },
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
    return Events.On('kova:command-palette', () => handler());
}
