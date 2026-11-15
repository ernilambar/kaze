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

// PLAIN_TEXT_LANGUAGE is Monaco's id for an unstyled buffer.
export const PLAIN_TEXT_LANGUAGE = 'plaintext';

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

// LanguageIndex maps what a path says about its content to the language Monaco
// registered for it. It is built from Monaco's own registry rather than a
// hand-written table, so every bundled language is recognised.
interface LanguageIndex {
    byExtension: Map<string, string>;
    byFilename: Map<string, string>;
    names: Map<string, string>;
}

let index: LanguageIndex | undefined;

function languages(): LanguageIndex {
    if (index === undefined) {
        const byExtension = new Map<string, string>();
        const byFilename = new Map<string, string>();
        const names = new Map<string, string>();
        for (const language of monaco.languages.getLanguages()) {
            names.set(language.id, language.aliases?.[0] ?? language.id);
            for (const extension of language.extensions ?? []) {
                const key = extension.toLowerCase();
                // First registration wins, matching Monaco's own resolution: a
                // specific language registers before a more generic one.
                if (!byExtension.has(key)) {
                    byExtension.set(key, language.id);
                }
            }
            for (const filename of language.filenames ?? []) {
                const key = filename.toLowerCase();
                if (!byFilename.has(key)) {
                    byFilename.set(key, language.id);
                }
            }
        }
        index = { byExtension, byFilename, names };
    }
    return index;
}

function baseName(path: string): string {
    const slash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'));
    return path.slice(slash + 1);
}

// languageForPath is the Monaco language id for a file path. An unknown or
// extensionless name stays plain text instead of guessing.
export function languageForPath(path: string): string {
    const name = baseName(path).toLowerCase();
    if (name === '') {
        return PLAIN_TEXT_LANGUAGE;
    }
    const { byExtension, byFilename } = languages();
    const byName = byFilename.get(name);
    if (byName !== undefined) {
        return byName;
    }
    const dot = name.lastIndexOf('.');
    if (dot <= 0) {
        return PLAIN_TEXT_LANGUAGE;
    }
    return byExtension.get(name.slice(dot)) ?? PLAIN_TEXT_LANGUAGE;
}

// languageLabel is what the status bar prints. Monaco highlights SVG as XML,
// but the file's own name is what the reader expects to see there.
export function languageLabel(path: string): string {
    if (path.toLowerCase().endsWith('.svg')) {
        return 'SVG';
    }
    return languages().names.get(languageForPath(path)) ?? 'Plain Text';
}

// createModel makes a text buffer with its own undo stack. One model per open
// document is what lets switching tabs preserve edits instead of rewriting the
// active buffer.
export function createModel(key: string, content: string): monaco.editor.ITextModel {
    const language =
        key === WELCOME_KEY || key.startsWith(UNTITLED_PREFIX)
            ? PLAIN_TEXT_LANGUAGE
            : languageForPath(key);
    return monaco.editor.createModel(content, language, modelURI(key));
}

// editorOptions translates settings.json into Monaco options. Zoom passes the
// effective font size, because the zoom offset is a session value and never
// reaches the settings file.
export function editorOptions(
    settings: EditorSettings,
    fontSize = settings.fontSize,
): monaco.editor.IStandaloneEditorConstructionOptions {
    return {
        fontSize,
        fontFamily: settings.fontFamily,
        fontWeight: String(settings.fontWeight),
        fontLigatures: settings.ligatures,
        cursorWidth: settings.cursorWidth,
        cursorBlinking: settings.cursorBlink ? 'blink' : 'solid',
        cursorSurroundingLines: settings.cursorSurroundingLines,
        minimap: { enabled: false },
        scrollBeyondLastLine: settings.scrollBeyondLastLine,
        wordWrap: settings.wordWrap ? 'on' : 'off',
        renderWhitespace: settings.invisibleCharacters ? 'all' : 'none',
        lineNumbers: settings.lineNumbers ? 'on' : 'off',
        guides: { indentation: settings.indentGuides },
        folding: settings.codeFolding,
        autoClosingBrackets: settings.autoCloseBrackets ? 'languageDefined' : 'never',
        autoClosingQuotes: settings.autoCloseBrackets ? 'languageDefined' : 'never',
        autoSurround: settings.autoCloseBrackets ? 'languageDefined' : 'never',
        autoIndent: settings.smartIndent ? 'full' : 'none',
        detectIndentation: false,
        tabSize: settings.tabSize,
        insertSpaces: !settings.hardTabs,
        padding: { top: 12, bottom: 12 },
    };
}

// modelOptions is the indentation half of the settings. Monaco keeps tab size
// and space insertion on the model, not the editor, so each buffer is told.
export function modelOptions(settings: EditorSettings): monaco.editor.ITextModelUpdateOptions {
    return {
        tabSize: settings.tabSize,
        indentSize: settings.tabSize,
        insertSpaces: !settings.hardTabs,
    };
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
        ...editorOptions(settings),
    });
}
