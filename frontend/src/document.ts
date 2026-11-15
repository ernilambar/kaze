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

// OpenDocument is one tab's buffer. It lives in the frontend because the editor
// owns the text; Go only reads it from disk and writes it back.
export interface OpenDocument {
    // id identifies the buffer for the tab's whole life. A file uses its path.
    // An untitled buffer uses a name that is not a path, so Save As can give it
    // one without the editor switching to a different model.
    id: string;
    // path is empty until an untitled buffer is saved.
    path: string;
    name: string;
    kind: DocumentKind;
    // content is the text that is on disk: what was loaded, or what was last
    // saved. A model is created from it once and keeps its own edits.
    content: string;
    src: string;
    // modified is the file's mtime when content was read, in Unix milliseconds.
    // It is handed back on save so a write never overwrites a change another
    // program made.
    modified: number;
    // dirty marks edits that are not on disk yet.
    dirty: boolean;
}

// UNTITLED_PREFIX starts the id of a buffer that has no file yet. Its id is not
// a path, so the editor model knows to live in memory, and Save As can attach a
// real path without the buffer changing identity.
export const UNTITLED_PREFIX = 'untitled:';

const UNTITLED_NAME = /^Untitled-(\d+)$/;

// nextUntitledName numbers a new buffer after the highest Untitled-N already
// open, so closing Untitled-1 and making another one never makes two tabs look
// alike while both exist.
export function nextUntitledName(tabs: readonly OpenDocument[]): string {
    let highest = 0;
    for (const tab of tabs) {
        const match = UNTITLED_NAME.exec(tab.name);
        if (match !== null) {
            highest = Math.max(highest, Number(match[1]));
        }
    }
    return `Untitled-${highest + 1}`;
}

// createUntitledDocument is New File: an empty buffer that Save turns into a
// Save As.
export function createUntitledDocument(tabs: readonly OpenDocument[]): OpenDocument {
    const name = nextUntitledName(tabs);
    return {
        id: `${UNTITLED_PREFIX}${name}`,
        path: '',
        name,
        kind: 'text',
        content: '',
        src: '',
        modified: 0,
        dirty: false,
    };
}

// shownName is what the title bar prints for a document.
export function shownName(document: OpenDocument | undefined): string {
    return document?.name ?? 'Untitled';
}

export function toOpenDocument(document: models.Document): OpenDocument {
    return {
        id: document.path,
        path: document.path,
        name: document.name,
        kind: document.kind === 'image' ? 'image' : 'text',
        content: document.content,
        src: document.src,
        modified: document.modified,
        dirty: false,
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
