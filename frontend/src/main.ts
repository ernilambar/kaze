import './style.css';
import { createEditor } from './editor';
import { loadEditorSettings, openSettingsFile } from './settings';
import { formatCursorPosition } from './status';

async function main(): Promise<void> {
    document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
        <header class="titlebar">
            <span class="titlebar__app">Kova</span>
            <span class="titlebar__file">untitled.txt</span>
        </header>
        <main class="editor" id="editor"></main>
        <footer class="statusbar">
            <span id="status-position">Ln 1, Col 1</span>
            <span id="status-language">Plain Text</span>
        </footer>
    `;

    const positionElement = document.getElementById('status-position')!;
    const settings = await loadEditorSettings();
    const editor = createEditor(document.getElementById('editor')!, settings);

    // Cmd+, on macOS, Ctrl+, elsewhere. Window-level so the shortcut works
    // regardless of what inside the page has focus.
    window.addEventListener('keydown', (event) => {
        const toggleModifier = event.metaKey || event.ctrlKey;
        if (toggleModifier && !event.altKey && !event.shiftKey && event.key === ',') {
            event.preventDefault();
            void openSettingsFile();
        }
    });

    editor.onDidChangeCursorPosition((event) => {
        positionElement.textContent = formatCursorPosition(
            event.position.lineNumber,
            event.position.column,
        );
    });
}

void main();
