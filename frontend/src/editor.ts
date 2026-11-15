import * as monaco from 'monaco-editor';
import type { EditorSettings } from './settings';

// WELCOME_TEXT is what the editor shows when no file was opened.
export const WELCOME_TEXT = `Hello, world!

Kova is a simple text editor built with Wails and Monaco.
`;

// WELCOME_KEY is the model key for the welcome buffer. It is empty because an
// open document is identified by its path, and no real file has an empty path.
export const WELCOME_KEY = '';

const WELCOME_URI = 'inmemory://model/welcome.txt';

// createModel makes a text buffer with its own undo stack. One model per open
// document is what lets switching tabs preserve edits instead of rewriting the
// active buffer.
export function createModel(key: string, content: string): monaco.editor.ITextModel {
    const uri = key === WELCOME_KEY ? monaco.Uri.parse(WELCOME_URI) : monaco.Uri.file(key);
    return monaco.editor.createModel(content, 'plaintext', uri);
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
