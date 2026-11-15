import type * as models from '../bindings/kova/models';
import { GetSettings, UpdateEditorSettings } from '../bindings/kova/app';

export interface EditorSettings {
    fontFamily: string;
    fontSize: number;
    fontWeight: number;
    tabSize: number;
    hardTabs: boolean;
    ligatures: boolean;
    cursorWidth: number;
    cursorBlink: boolean;
    cursorSurroundingLines: number;
    scrollBeyondLastLine: boolean;
    ensureFinalNewline: boolean;
    wordWrap: boolean;
    invisibleCharacters: boolean;
    lineNumbers: boolean;
    indentGuides: boolean;
    codeFolding: boolean;
    autoCloseBrackets: boolean;
    smartIndent: boolean;
}

// Mirrors defaultSettings in settings.go. Used only when the backend call
// fails, e.g. running `pnpm dev` outside a Wails window.
export const defaultEditorSettings: EditorSettings = {
    fontFamily: 'Menlo',
    fontSize: 13,
    fontWeight: 400,
    tabSize: 4,
    hardTabs: false,
    ligatures: false,
    cursorWidth: 1,
    cursorBlink: true,
    cursorSurroundingLines: 0,
    scrollBeyondLastLine: false,
    ensureFinalNewline: false,
    wordWrap: true,
    invisibleCharacters: false,
    lineNumbers: true,
    indentGuides: true,
    codeFolding: true,
    autoCloseBrackets: true,
    smartIndent: true,
};

function toEditorSettings(settings: models.EditorSettings): EditorSettings {
    return {
        fontFamily: settings.font_family,
        fontSize: settings.font_size,
        fontWeight: settings.font_weight,
        tabSize: settings.tab_size,
        hardTabs: settings.hard_tabs,
        ligatures: settings.ligatures,
        cursorWidth: settings.cursor_width,
        cursorBlink: settings.cursor_blink,
        cursorSurroundingLines: settings.cursor_surrounding_lines,
        scrollBeyondLastLine: settings.scroll_beyond_last_line,
        ensureFinalNewline: settings.ensure_final_newline,
        wordWrap: settings.word_wrap,
        invisibleCharacters: settings.invisible_characters,
        lineNumbers: settings.line_numbers,
        indentGuides: settings.indent_guides,
        codeFolding: settings.code_folding,
        autoCloseBrackets: settings.auto_close_brackets,
        smartIndent: settings.smart_indent,
    };
}

function toBindingSettings(settings: EditorSettings): models.EditorSettings {
    return {
        font_family: settings.fontFamily,
        font_size: settings.fontSize,
        font_weight: settings.fontWeight,
        tab_size: settings.tabSize,
        hard_tabs: settings.hardTabs,
        ligatures: settings.ligatures,
        cursor_width: settings.cursorWidth,
        cursor_blink: settings.cursorBlink,
        cursor_surrounding_lines: settings.cursorSurroundingLines,
        scroll_beyond_last_line: settings.scrollBeyondLastLine,
        ensure_final_newline: settings.ensureFinalNewline,
        word_wrap: settings.wordWrap,
        invisible_characters: settings.invisibleCharacters,
        line_numbers: settings.lineNumbers,
        indent_guides: settings.indentGuides,
        code_folding: settings.codeFolding,
        auto_close_brackets: settings.autoCloseBrackets,
        smart_indent: settings.smartIndent,
    };
}

export async function loadEditorSettings(): Promise<EditorSettings> {
    try {
        return toEditorSettings((await GetSettings()).editor);
    } catch (error) {
        console.error('Failed to load settings; using defaults.', error);
        return { ...defaultEditorSettings };
    }
}

// saveEditorSettings writes the editor section back to settings.json and
// returns the normalized values the backend stored.
export async function saveEditorSettings(settings: EditorSettings): Promise<EditorSettings> {
    return toEditorSettings(await UpdateEditorSettings(toBindingSettings(settings)));
}
