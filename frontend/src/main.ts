import './style.css';
import { createEditor } from './editor';
import { formatCursorPosition } from './status';

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
const editor = createEditor(document.getElementById('editor')!);

editor.onDidChangeCursorPosition((event) => {
    positionElement.textContent = formatCursorPosition(
        event.position.lineNumber,
        event.position.column,
    );
});
