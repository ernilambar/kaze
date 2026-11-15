import * as monaco from 'monaco-editor';
import { UNTITLED_PREFIX } from './document';
import type { EditorSettings } from './settings';

// WELCOME_TEXT is what the editor shows when no file was opened.
export const WELCOME_TEXT = `Hello, world!

Kova is a simple text editor built with Wails and Monaco.
`;

// WELCOME_KEY is the model key for the welcome buffer. It is empty because an
// open document is identified by its tab id, and no tab has an empty id.
export const WELCOME_KEY = '';

const WELCOME_URI = 'inmemory://model/welcome.txt';

// modelURI gives a buffer its Monaco identity. Files use their path, so two
// models can never claim the same document; untitled buffers have no path, so
// they live in memory under their tab name.
function modelURI(key: string): monaco.Uri {
    if (key === WELCOME_KEY) {
        return monaco.Uri.parse(WELCOME_URI);
    }
    if (key.startsWith(UNTITLED_PREFIX)) {
        return monaco.Uri.parse(`inmemory://model/${key.slice(UNTITLED_PREFIX.length)}`);
    }
    return monaco.Uri.file(key);
}

// createModel makes a text buffer with its own undo stack. One model per open
// document is what lets switching tabs preserve edits instead of rewriting the
// active buffer.
export function createModel(key: string, content: string): monaco.editor.ITextModel {
    return monaco.editor.createModel(content, 'plaintext', modelURI(key));
}

export function createEditor(
    container: HTMLElement,
    settings: EditorSettings,
    model: monaco.editor.ITextModel,
): monaco.editor.IStandaloneCodeEditor {
    return monaco.editor.create(container, {
        model,
        theme: 'vs-dark',
        automaticLayout: true,
        fontSize: settings.fontSize,
        fontFamily: settings.fontFamily,
        fontWeight: String(settings.fontWeight),
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        padding: { top: 12, bottom: 12 },
    });
}
