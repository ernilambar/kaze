import { EventsOn } from '../wailsjs/runtime/runtime';
import type { PaletteItem } from './palette';
import { rankCandidates } from './quickOpen';

// Command is one entry in the command palette. Commands are static metadata;
// App.svelte maps each id to the function that runs it.
export type Command = PaletteItem;

// COMMANDS is the full palette, in the order shown before the user types. The
// detail names the menu the command comes from, so typing "file" or "kova"
// narrows the list by category as well as by name.
export const COMMANDS: readonly Command[] = [
    { id: 'quick-open', name: 'Quick Open…', detail: 'File' },
    { id: 'add-folder', name: 'Add Folder to Sidebar…', detail: 'File' },
    { id: 'close-tab', name: 'Close Tab', detail: 'File' },
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
    return EventsOn('kova:command-palette', () => handler());
}
