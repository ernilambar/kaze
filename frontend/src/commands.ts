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
    { id: 'toggle-preview', name: 'Toggle Preview', detail: 'View' },
    { id: 'zoom-in', name: 'Zoom In', detail: 'View' },
    { id: 'zoom-out', name: 'Zoom Out', detail: 'View' },
    { id: 'zoom-reset', name: 'Actual Size', detail: 'View' },
    ...VIEW_OPTIONS.map((option) => ({
        id: `toggle-${option.id}`,
        name: `Toggle ${option.label}`,
        detail: 'View',
    })),
];

// COMMANDS is the full palette, in the order shown before the user types. The
// detail names the menu the command comes from, so typing "file" or "kova"
// narrows the list by category as well as by name.
export const COMMANDS: readonly Command[] = [
    { id: 'new-file', name: 'New File', detail: 'File' },
    { id: 'open-file', name: 'Open…', detail: 'File' },
    { id: 'save', name: 'Save', detail: 'File' },
    { id: 'save-as', name: 'Save As…', detail: 'File' },
    { id: 'save-all', name: 'Save All', detail: 'File' },
    { id: 'print', name: 'Print…', detail: 'File' },
    { id: 'quick-open', name: 'Quick Open…', detail: 'File' },
    { id: 'add-folder', name: 'Add Folder to Sidebar…', detail: 'File' },
    { id: 'close-tab', name: 'Close Tab', detail: 'File' },
    ...EDIT_COMMANDS,
    ...VIEW_COMMANDS,
    { id: 'settings', name: 'Settings…', detail: 'Kova' },
    { id: 'install-command-line-tool', name: 'Install Command Line Tool…', detail: 'Kova' },
];

// rankCommands orders commands exactly like Quick Open orders files.
export function rankCommands(
    query: string,
    commands: readonly Command[] = COMMANDS,
): Command[] {
    return rankCandidates(query, commands);
}

// listenForCommandPalette runs handler when the File menu asks for the palette.
// Returns an unsubscribe function.
export function listenForCommandPalette(handler: () => void): () => void {
    return Events.On('kova:command-palette', () => handler());
}
