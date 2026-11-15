import type * as models from '../bindings/kova/models';
import { GetLaunchRequest, LoadDocument } from '../bindings/kova/app';
import { Events } from '@wailsio/runtime';

// Mirrors LaunchRequest in cli.go. The backend resolves paths; the frontend only
// decides what to display for them.
export interface LaunchRequest {
    files: string[];
    folders: string[];
    missing: string[];
}

export const emptyLaunchRequest: LaunchRequest = { files: [], folders: [], missing: [] };

// toLaunchRequest normalizes the binding model, whose slices are nullable
// because a nil Go slice encodes as null. The rest of the frontend works with
// arrays only.
export function toLaunchRequest(request: models.LaunchRequest): LaunchRequest {
    return {
        files: request.files ?? [],
        folders: request.folders ?? [],
        missing: request.missing ?? [],
    };
}

// DocumentKind selects the tab's surface: Monaco for text, the image viewer for
// pictures. Mirrors the Kind field in document.go.
export type DocumentKind = 'text' | 'image';

export interface OpenDocument {
    path: string;
    name: string;
    kind: DocumentKind;
    content: string;
    src: string;
}

// shownName is what the title bar prints for a document.
export function shownName(document: OpenDocument | undefined): string {
    return document?.name ?? 'untitled.txt';
}

export function toOpenDocument(document: models.Document): OpenDocument {
    return {
        path: document.path,
        name: document.name,
        kind: document.kind === 'image' ? 'image' : 'text',
        content: document.content,
        src: document.src,
    };
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
    const off = Events.On('kova:launch', (event) => {
        const request = event.data as LaunchRequest | undefined;
        if (request !== undefined) {
            handler(request);
        }
    });

    try {
        handler(toLaunchRequest(await GetLaunchRequest()));
    } catch (error) {
        console.error('Failed to read the launch request.', error);
    }

    return off;
}
