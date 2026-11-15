import type { main } from '../wailsjs/go/models';
import { GetLaunchRequest, LoadDocument } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

// Mirrors LaunchRequest in cli.go. The backend resolves paths; the frontend only
// decides what to display for them.
export interface LaunchRequest {
    files: string[];
    folder: string;
    missing: string[];
}

export const emptyLaunchRequest: LaunchRequest = { files: [], folder: '', missing: [] };

export interface OpenDocument {
    path: string;
    name: string;
    content: string;
}

// shownName is what the title bar prints for a document.
export function shownName(document: OpenDocument | undefined): string {
    return document?.name ?? 'untitled.txt';
}

// folderLabel names the working folder in the title bar and status bar.
export function folderLabel(folder: string): string {
    if (folder === '') {
        return '';
    }
    const parts = folder.split('/').filter((part) => part !== '');
    return parts.length === 0 ? '/' : parts[parts.length - 1];
}

export function toOpenDocument(document: main.Document): OpenDocument {
    return { path: document.path, name: document.name, content: document.content };
}

// readDocument loads one file, reporting failures as a message instead of
// throwing, so one bad path cannot stop the rest of a launch.
export async function readDocument(path: string): Promise<OpenDocument | undefined> {
    try {
        return toOpenDocument(await LoadDocument(path));
    } catch (error) {
        console.error(`Failed to open ${path}.`, error);
        return undefined;
    }
}

// listenForLaunch runs handler with every launch request, starting with the one
// Kova was started with. New requests arrive as events while the app runs.
export async function listenForLaunch(handler: (request: LaunchRequest) => void): Promise<() => void> {
    const off = EventsOn('kova:launch', (...data: unknown[]) => {
        const request = data[0] as LaunchRequest | undefined;
        if (request !== undefined) {
            handler(request);
        }
    });

    try {
        handler(await GetLaunchRequest());
    } catch (error) {
        console.error('Failed to read the launch request.', error);
    }

    return off;
}
