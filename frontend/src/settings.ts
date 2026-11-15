import type * as models from '../bindings/kova/models';
import { GetSettings, UpdateEditorSettings, UpdateThemeSettings, UpdateUISettings } from '../bindings/kova/app';

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

// ThemeMode is how the palette is chosen. System follows the OS appearance.
export type ThemeMode = 'system' | 'light' | 'dark';

// ThemeSettings names one variant for each appearance. The names are the
// `name` fields inside Zed theme files.
export interface ThemeSettings {
    mode: ThemeMode;
    light: string;
    dark: string;
}

export type IconStyle = 'colorful' | 'monochrome';
export type SidebarPosition = 'left' | 'right';

// UISettings controls the chrome around the editor.
export interface UISettings {
    fontFamily: string;
    fontSize: number;
    tabIcons: boolean;
    iconStyle: IconStyle;
    sidebarPosition: SidebarPosition;
}

// DEFAULT_UI_FONT matches the body font stack in style.css. An empty setting
// means "use this", so the CSS and the fallback agree.
export const DEFAULT_UI_FONT =
    '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif';

// Defaults mirror defaultSettings in settings.go: system appearance, the built-in
// pair, the system UI font, 13px chrome, file icons on and the sidebar left.
export const defaultThemeSettings: ThemeSettings = {
    mode: 'system',
    light: 'Kova Light',
    dark: 'Kova Dark',
};

export const defaultUISettings: UISettings = {
    fontFamily: '',
    fontSize: 13,
    tabIcons: true,
    iconStyle: 'colorful',
    sidebarPosition: 'left',
};

function toThemeSettings(theme: models.ThemeSettings): ThemeSettings {
    const mode = theme.mode === 'light' || theme.mode === 'dark' ? theme.mode : 'system';
    return {
        mode,
        light: theme.light,
        dark: theme.dark,
    };
}

function toBindingTheme(theme: ThemeSettings): models.ThemeSettings {
    return {
        mode: theme.mode,
        light: theme.light,
        dark: theme.dark,
    };
}

function toUISettings(ui: models.UISettings): UISettings {
    return {
        fontFamily: ui.font_family,
        fontSize: ui.font_size,
        tabIcons: ui.tab_icons,
        iconStyle: ui.icon_style === 'monochrome' ? 'monochrome' : 'colorful',
        sidebarPosition: ui.sidebar_position === 'right' ? 'right' : 'left',
    };
}

function toBindingUI(ui: UISettings): models.UISettings {
    return {
        font_family: ui.fontFamily,
        font_size: ui.fontSize,
        tab_icons: ui.tabIcons,
        icon_style: ui.iconStyle,
        sidebar_position: ui.sidebarPosition,
    };
}

// loadPreferences reads the theme and interface sections. It is called once at
// startup; the editor section has its own loader because toggles rewrite it
// independently.
export async function loadPreferences(): Promise<{
    theme: ThemeSettings;
    ui: UISettings;
}> {
    try {
        const settings = await GetSettings();
        return {
            theme: toThemeSettings(settings.theme),
            ui: toUISettings(settings.ui),
        };
    } catch (error) {
        console.error('Failed to load appearance settings; using defaults.', error);
        return {
            theme: { ...defaultThemeSettings },
            ui: { ...defaultUISettings },
        };
    }
}

export async function saveThemeSettings(theme: ThemeSettings): Promise<ThemeSettings> {
    return toThemeSettings(await UpdateThemeSettings(toBindingTheme(theme)));
}

export async function saveUISettings(ui: UISettings): Promise<UISettings> {
    return toUISettings(await UpdateUISettings(toBindingUI(ui)));
}
