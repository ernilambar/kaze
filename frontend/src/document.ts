import type * as models from '../bindings/kova/models';
import { GetLaunchRequest, LoadDocument } from '../bindings/kova/app';
import { onWindowEvent } from './windowstate';

// LaunchBuffer mirrors LaunchBuffer in cli.go: one tab handed to another
// window. A text buffer carries its text so unsaved edits move with it.
export interface LaunchBuffer {
    name: string;
    path: string;
    kind: string;
    content: string;
    saved_content: string;
    modified: number;
    dirty: boolean;
    source_path: string;
}

// Mirrors LaunchRequest in cli.go. The backend resolves paths; the frontend only
// decides what to display for them.
export interface LaunchRequest {
    files: string[];
    folders: string[];
    missing: string[];
    buffers: LaunchBuffer[];
    replace_folders: boolean;
    restore_session: boolean;
}

export const emptyLaunchRequest: LaunchRequest = {
    files: [],
    folders: [],
    missing: [],
    buffers: [],
    replace_folders: false,
    restore_session: false,
};

// toLaunchRequest normalizes the binding model, whose slices are nullable
// because a nil Go slice encodes as null. The rest of the frontend works with
// arrays only.
export function toLaunchRequest(request: models.LaunchRequest): LaunchRequest {
    return {
        files: request.files ?? [],
        folders: request.folders ?? [],
        missing: request.missing ?? [],
        buffers: (request.buffers ?? []).map((buffer) => ({
            name: buffer.name,
            path: buffer.path,
            kind: buffer.kind,
            content: buffer.content,
            saved_content: buffer.saved_content ?? '',
            modified: buffer.modified,
            dirty: buffer.dirty,
            source_path: buffer.source_path ?? '',
        })),
        replace_folders: request.replace_folders ?? false,
        restore_session: request.restore_session ?? false,
    };
}

// DocumentKind selects the tab's surface: Monaco for text and diffs, the image
// viewer for pictures. Mirrors the Kind field in document.go, plus the
// frontend-only diff tab.
export type DocumentKind = 'text' | 'image' | 'diff';

// OpenDocument is one tab's buffer. It lives in the frontend because the editor
// owns the text; Go only reads it from disk and writes it back.
export interface OpenDocument {
    // id identifies the buffer for the tab's whole life. A file uses its path.
    // An untitled buffer uses a name that is not a path, so Save As can give it
    // one without the editor switching to a different model.
    id: string;
    // path is empty until an untitled buffer is saved, and for a diff tab,
    // which is a snapshot rather than a file.
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
    // savedContent is the text on disk when it differs from content. It is set
    // for a tab moved from another window with unsaved edits, so the editor
    // still knows what a clean buffer would be.
    savedContent?: string;
    // sourcePath is the file a diff tab is showing. It is empty for every other
    // kind, and kept apart from path so a diff never looks saveable.
    sourcePath?: string;
}

// UNTITLED_PREFIX starts the id of a buffer that has no file yet. Its id is not
// a path, so the editor model knows to live in memory, and Save As can attach a
// real path without the buffer changing identity.
export const UNTITLED_PREFIX = 'untitled:';

// DIFF_PREFIX starts the id of a diff tab. The editor gives these models their
// own language and read-only mode instead of treating them as files.
export const DIFF_PREFIX = 'diff:';

export function diffTabId(path: string): string {
    return `${DIFF_PREFIX}${path}`;
}

function baseName(path: string): string {
    const parts = path.split(/[\\/]/).filter((part) => part !== '');
    return parts.length === 0 ? path : parts[parts.length - 1];
}

// createDiffDocument is the read-only view of a file against HEAD. The diff is
// a snapshot: it is refreshed by the app when the workspace refreshes or the
// file is saved.
export function createDiffDocument(path: string, diff: string): OpenDocument {
    const name = baseName(path);
    return {
        id: diffTabId(path),
        path: '',
        name: `${name} (Diff)`,
        kind: 'diff',
        content: diff === '' ? `No changes against HEAD for ${name}.\n` : diff,
        src: '',
        modified: 0,
        dirty: false,
        sourcePath: path,
    };
}

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

// toBufferDocument turns a tab moved from another window back into a document.
// A text buffer keeps the text the other window held, so unsaved edits survive
// the move; an untitled buffer keeps its name because it has no path.
export function toBufferDocument(buffer: LaunchBuffer): OpenDocument {
    if (buffer.kind === 'diff') {
        return {
            id: diffTabId(buffer.source_path),
            path: '',
            name: buffer.name,
            kind: 'diff',
            content: buffer.content,
            src: '',
            modified: 0,
            dirty: false,
            sourcePath: buffer.source_path,
        };
    }
    return {
        id: buffer.path !== '' ? buffer.path : `${UNTITLED_PREFIX}${buffer.name}`,
        path: buffer.path,
        name: buffer.name,
        kind: buffer.kind === 'image' ? 'image' : 'text',
        content: buffer.content,
        savedContent: buffer.saved_content,
        src: '',
        modified: buffer.modified,
        dirty: buffer.dirty,
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

// listenForLaunch runs handler with every launch request for one window,
// starting with the one the window was created with. The listener is installed
// before the request is collected, so a path that arrives at the same moment is
// queued rather than lost.
export async function listenForLaunch(
    windowName: string,
    handler: (request: LaunchRequest) => void,
): Promise<() => void> {
    const off = onWindowEvent('kova:launch', (data) => {
        const request = data as LaunchRequest | undefined;
        if (request !== undefined) {
            handler(request);
        }
    });

    try {
        handler(toLaunchRequest(await GetLaunchRequest(windowName)));
    } catch (error) {
        console.error('Failed to read the launch request.', error);
    }

    return off;
}
