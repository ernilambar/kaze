import * as monaco from 'monaco-editor';
import type { EditorSettings } from './settings';

const WELCOME_TEXT = `Hello, world!

Kova is a simple text editor built with Wails and Monaco.
`;

export function createEditor(
    container: HTMLElement,
    settings: EditorSettings,
): monaco.editor.IStandaloneCodeEditor {
    return monaco.editor.create(container, {
        value: WELCOME_TEXT,
        language: 'plaintext',
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
