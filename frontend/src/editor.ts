import * as monaco from 'monaco-editor';
import { DIFF_PREFIX, UNTITLED_PREFIX, splitTabKey } from './document';
import type { PaletteItem } from './palette';
import type { EditorSettings } from './settings';
import { shebangLanguage } from './shebang';

// DIFF_LANGUAGE is Monaco's built-in diff syntax, used by the read-only diff
// tabs. It is not chosen from a file extension.
const DIFF_LANGUAGE = 'diff';

// WELCOME_KEY is the model key for the idle buffer shown when no file is open.
// It is empty because an open document is identified by its tab id, and no tab
// has an empty id.
export const WELCOME_KEY = '';

// PLAIN_TEXT_LANGUAGE is Monaco's id for an unstyled buffer.
export const PLAIN_TEXT_LANGUAGE = 'plaintext';

const IDLE_URI = 'inmemory://model/idle.txt';

// modelURI gives a buffer its Monaco identity. Files use their path, so two
// models can never claim the same document; untitled buffers have no path, so
// they live in memory under their tab name.
function modelURI(key: string): monaco.Uri {
    // A tab whose id another tab already held carries a counter; the query
    // keeps its model apart while the path still names the file.
    const { base, copy } = splitTabKey(key);
    const uri = baseModelURI(base);
    return copy === '' ? uri : uri.with({ query: `copy=${copy}` });
}

function baseModelURI(key: string): monaco.Uri {
    if (key === WELCOME_KEY) {
        return monaco.Uri.parse(IDLE_URI);
    }
    if (key.startsWith(DIFF_PREFIX)) {
        return monaco.Uri.parse(
            `inmemory://model/diff/${encodeURIComponent(key.slice(DIFF_PREFIX.length))}`,
        );
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

// hasRegisteredName reports whether the path's own name or extension is known
// to Monaco. A `.txt` file is explicitly Plain Text, so a shebang must not
// override it; an unknown or absent extension is what leaves the first line to
// decide.
function hasRegisteredName(path: string): boolean {
    const name = baseName(path).toLowerCase();
    if (name === '') {
        return false;
    }
    const { byExtension, byFilename } = languages();
    if (byFilename.has(name)) {
        return true;
    }
    const dot = name.lastIndexOf('.');
    return dot > 0 && byExtension.has(name.slice(dot));
}

// languageForDocument resolves a buffer's language. An explicit override wins;
// otherwise the path decides, and only when it names no language does the
// first line get a say, so a Makefile stays a Makefile while an extensionless
// script works out to shell or python from its `#!`.
export function languageForDocument(
    path: string,
    content: string = '',
    override: string = '',
): string {
    if (override !== '') {
        return override;
    }
    const byPath = languageForPath(path);
    if (byPath !== PLAIN_TEXT_LANGUAGE || hasRegisteredName(path)) {
        return byPath;
    }
    return shebangLanguage(content) ?? PLAIN_TEXT_LANGUAGE;
}

// languageLabel is what the status bar prints. Monaco highlights SVG as XML,
// but the file's own name is what the reader expects to see there. An explicit
// override is shown as its language name, and a shebang-derived language is
// named the same way.
export function languageLabel(
    path: string,
    content: string = '',
    override: string = '',
): string {
    if (override === '' && path.toLowerCase().endsWith('.svg')) {
        return 'SVG';
    }
    const id = languageForDocument(path, content, override);
    return languages().names.get(id) ?? id;
}

// AUTO_LANGUAGE_ID is the Change Language Mode row that clears an override.
export const AUTO_LANGUAGE_ID = 'language-auto';

// languageChoices is the Change Language Mode list: Auto Detect first, then
// every language the editor bundle registered, sorted by the name the user
// sees.
export function languageChoices(): PaletteItem[] {
    const choices: PaletteItem[] = [
        { id: AUTO_LANGUAGE_ID, name: 'Auto Detect', detail: 'Language' },
    ];
    const seen = new Set<string>();
    for (const language of monaco.languages.getLanguages()) {
        if (seen.has(language.id)) {
            continue;
        }
        seen.add(language.id);
        choices.push({
            id: language.id,
            name: language.aliases?.[0] ?? language.id,
            detail: 'Language',
        });
    }
    return [
        choices[0],
        ...choices.slice(1).sort((a, b) => a.name.localeCompare(b.name)),
    ];
}

// createModel makes a text buffer with its own undo stack. One model per open
// document is what lets switching tabs preserve edits instead of rewriting the
// active buffer. eol is the file's line ending: the model adopts it so its text
// already matches what SaveDocument writes back.
export function createModel(
    key: string,
    content: string,
    eol: string = 'lf',
): monaco.editor.ITextModel {
    const language = key.startsWith(DIFF_PREFIX)
        ? DIFF_LANGUAGE
        : key === WELCOME_KEY
          ? PLAIN_TEXT_LANGUAGE
          : languageForDocument(splitTabKey(key).base, content);
    const model = monaco.editor.createModel(content, language, modelURI(key));
    model.setEOL(
        eol === 'crlf' ? monaco.editor.EndOfLineSequence.CRLF : monaco.editor.EndOfLineSequence.LF,
    );
    return model;
}

// editorOptions translates settings.json into Monaco options. Zoom passes the
// effective font size, because the zoom offset is a session value and never
// reaches the settings file. tabSize, insertSpaces and detectIndentation are
// model options: Monaco's standalone configuration carries them to every
// buffer it creates, where the file's own indentation is detected and the
// settings stand in only for new and empty buffers.
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
        minimap: { enabled: settings.minimap },
        rulers: settings.rulers,
        bracketPairColorization: { enabled: settings.bracketPairColors },
        matchBrackets: settings.bracketPairColors ? 'always' : 'never',
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
        // Detection guesses each file's indentation from its content, so a
        // 2-space file indents with 2 spaces even under tab_size: 4; the
        // settings values below are the fallback for new and empty buffers.
        detectIndentation: true,
        tabSize: settings.tabSize,
        insertSpaces: !settings.hardTabs,
        padding: { top: 12, bottom: 12 },
    };
}

// EDITOR_CHORDS are app commands bound to Cmd+K sequences. The page listener
// would swallow Cmd+K and break Monaco's own Cmd+K chords, so inside the editor
// Monaco dispatches them; DEFAULT_KEYBINDINGS covers the rest of the window.
const EDITOR_CHORDS: readonly { command: string; second: number }[] = [
    { command: 'close-all-tabs', second: monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyW },
    { command: 'close-all-folders', second: monaco.KeyCode.KeyF },
];

// installEditorKeybindings moves Monaco's defaults onto Kaze's keymap. It runs
// once: the rules are global to every editor. run executes an app command.
export function installEditorKeybindings(run: (command: string) => void): void {
    const { KeyMod, KeyCode } = monaco;
    for (const chord of EDITOR_CHORDS) {
        const id = `kaze.${chord.command}`;
        monaco.editor.addCommand({ id, run: () => run(chord.command) });
        monaco.editor.addKeybindingRule({
            keybinding: monaco.KeyMod.chord(KeyMod.CtrlCmd | KeyCode.KeyK, chord.second),
            command: id,
            when: 'editorFocus',
        });
    }
    const cmdAlt = KeyMod.CtrlCmd | KeyMod.Alt;
    const findInput = 'findInputFocussed || replaceInputFocussed';
    monaco.editor.addKeybindingRules([
        {
            keybinding: KeyMod.CtrlCmd | KeyCode.KeyJ,
            command: 'editor.action.joinLines',
            when: 'editorTextFocus && !editorReadonly',
        },
        // Cmd+Shift+L splits a selection into lines; Select All Occurrences
        // moves to Ctrl+Cmd+G.
        {
            keybinding: KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.KeyL,
            command: '-editor.action.selectHighlights',
        },
        {
            keybinding: KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.KeyL,
            command: 'editor.action.insertCursorAtEndOfEachLineSelected',
            when: 'editorTextFocus',
        },
        {
            keybinding: KeyMod.CtrlCmd | KeyMod.WinCtrl | KeyCode.KeyG,
            command: 'editor.action.selectHighlights',
            when: 'editorFocus',
        },
        {
            keybinding: KeyMod.WinCtrl | KeyMod.Alt | KeyCode.UpArrow,
            command: 'editor.action.insertCursorAbove',
            when: 'editorTextFocus',
        },
        {
            keybinding: KeyMod.WinCtrl | KeyMod.Alt | KeyCode.DownArrow,
            command: 'editor.action.insertCursorBelow',
            when: 'editorTextFocus',
        },
        // Monaco claims Cmd+Alt+C/R for its find toggles whenever the editor
        // has focus, which would swallow Copy Path and Reveal in Finder. Keep
        // the toggles inside the find widget only.
        { keybinding: cmdAlt | KeyCode.KeyC, command: '-toggleFindCaseSensitive' },
        { keybinding: cmdAlt | KeyCode.KeyC, command: 'toggleFindCaseSensitive', when: findInput },
        { keybinding: cmdAlt | KeyCode.KeyR, command: '-toggleFindRegex' },
        { keybinding: cmdAlt | KeyCode.KeyR, command: 'toggleFindRegex', when: findInput },
    ]);
}

export function createEditor(
    container: HTMLElement,
    settings: EditorSettings,
    model: monaco.editor.ITextModel,
): monaco.editor.IStandaloneCodeEditor {
    return monaco.editor.create(container, {
        model,
        automaticLayout: true,
        ...editorOptions(settings),
    });
}
