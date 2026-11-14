import * as monaco from 'monaco-editor';

const WELCOME_TEXT = `Hello, world!

Kova is a simple text editor built with Wails and Monaco.
`;

export function createEditor(container: HTMLElement): monaco.editor.IStandaloneCodeEditor {
    return monaco.editor.create(container, {
        value: WELCOME_TEXT,
        language: 'plaintext',
        theme: 'vs-dark',
        automaticLayout: true,
        fontSize: 14,
        fontFamily: "'SF Mono', Menlo, Monaco, 'Courier New', monospace",
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        padding: { top: 12, bottom: 12 },
    });
}
