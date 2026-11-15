import { onWindowEvent } from './windowstate';
import type * as monaco from 'monaco-editor';
import { ListThemes, LoadTheme } from '../bindings/kova/app';
import type * as models from '../bindings/kova/models';
import {
    DEFAULT_UI_FONT,
    type ThemeMode,
    type ThemeSettings,
    type UISettings,
} from './settings';

// ThemeAppearance is the variant's own light/dark label, not the selected mode.
export type ThemeAppearance = 'light' | 'dark';

export interface ThemeInfo {
    family: string;
    name: string;
    appearance: ThemeAppearance;
}

export interface ThemeCatalog {
    themes: ThemeInfo[];
    errors: string[];
}

export interface ThemeHighlight {
    color: string;
    fontStyle: string;
    fontWeight: number;
}

export interface ThemePalette {
    family: string;
    name: string;
    appearance: ThemeAppearance;
    // CSS custom property names without the `--`, matching settings.go.
    ui: Record<string, string>;
    // Monaco color ids.
    editor: Record<string, string>;
    syntax: Record<string, ThemeHighlight>;
}

// ConfigChange mirrors Go's live-reload payload.
export interface ConfigChange {
    themes: boolean;
    keybindings: boolean;
}

function appearanceOf(value: string): ThemeAppearance {
    return value === 'light' ? 'light' : 'dark';
}

export function toThemeInfo(summary: models.ThemeSummary): ThemeInfo {
    return {
        family: summary.family,
        name: summary.name,
        appearance: appearanceOf(summary.appearance),
    };
}

export function resolveThemeName(theme: ThemeSettings, prefersDark: boolean): string {
    switch (theme.mode) {
        case 'light':
            return theme.light;
        case 'dark':
            return theme.dark;
        default:
            return prefersDark ? theme.dark : theme.light;
    }
}

export function systemPrefersDark(): boolean {
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

// isHexColor keeps a malformed theme value out of CSS and Monaco.
export function isHexColor(value: string): boolean {
    return /^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/.test(value);
}

// toMonacoColor converts a theme color to the six-digit form Monaco token rules
// require: no leading #, no alpha. A #RGB or #RGBA value is expanded first.
export function toMonacoColor(value: string): string {
    if (!isHexColor(value)) {
        return '';
    }
    let hex = value.slice(1);
    if (hex.length === 3 || hex.length === 4) {
        hex = hex
            .slice(0, 3)
            .split('')
            .map((digit) => digit + digit)
            .join('');
    } else {
        hex = hex.slice(0, 6);
    }
    return hex.toUpperCase();
}

export function toThemeHighlight(highlight: models.themeHighlight): ThemeHighlight {
    return {
        color: highlight.color,
        fontStyle: highlight.font_style ?? '',
        fontWeight: highlight.font_weight ?? 0,
    };
}

export function toThemePalette(palette: models.ThemePalette): ThemePalette {
    const ui: Record<string, string> = {};
    for (const [key, value] of Object.entries(palette.ui ?? {})) {
        if (value !== undefined && isHexColor(value)) {
            ui[key] = value;
        }
    }
    const editor: Record<string, string> = {};
    for (const [key, value] of Object.entries(palette.editor ?? {})) {
        if (value !== undefined && isHexColor(value)) {
            editor[key] = value;
        }
    }
    const syntax: Record<string, ThemeHighlight> = {};
    for (const [key, value] of Object.entries(palette.syntax ?? {})) {
        if (value !== undefined) {
            syntax[key] = toThemeHighlight(value);
        }
    }
    return {
        family: palette.family,
        name: palette.name,
        appearance: appearanceOf(palette.appearance),
        ui,
        editor,
        syntax,
    };
}

// CAPTURE_TOKENS maps Zed's semantic captures onto the names Monaco's Monarch
// tokenizers emit. Captures without a corresponding token are intentionally
// absent: coloring every `identifier` to fake a function color would be worse
// than leaving those tokens on the base theme.
const CAPTURE_TOKENS: readonly (readonly [string, readonly string[]])[] = [
    ['comment', ['comment']],
    ['comment.doc', ['comment.doc']],
    ['keyword', ['keyword']],
    ['preproc', ['keyword']],
    ['string', ['string']],
    ['string.regex', ['regexp']],
    ['number', ['number']],
    ['boolean', ['constant']],
    ['constant', ['constant']],
    ['type', ['type']],
    ['variable', ['variable']],
    ['operator', ['operator']],
    ['punctuation', ['delimiter']],
    ['punctuation.bracket', ['delimiter.bracket']],
    ['punctuation.delimiter', ['delimiter']],
    ['tag', ['tag']],
    ['attribute', ['attribute.name']],
    ['property', ['key']],
    ['invalid', ['invalid']],
    ['emphasis', ['emphasis']],
    ['emphasis.strong', ['strong']],
];

function fontStyleOf(highlight: ThemeHighlight): string {
    const styles: string[] = [];
    if (highlight.fontStyle === 'italic' || highlight.fontStyle === 'oblique') {
        styles.push('italic');
    }
    if (highlight.fontWeight >= 600) {
        styles.push('bold');
    }
    return styles.join(' ');
}

// syntaxRules turns the merged syntax map into Monaco token rules. Later rules
// win, so specific captures are emitted after their general form.
export function syntaxRules(syntax: Record<string, ThemeHighlight>): monaco.editor.ITokenThemeRule[] {
    const byToken = new Map<string, monaco.editor.ITokenThemeRule>();
    for (const [capture, tokens] of CAPTURE_TOKENS) {
        const highlight = syntax[capture];
        if (highlight === undefined) {
            continue;
        }
        const foreground = toMonacoColor(highlight.color);
        const fontStyle = fontStyleOf(highlight);
        if (foreground === '' && fontStyle === '') {
            continue;
        }
        for (const token of tokens) {
            const rule: monaco.editor.ITokenThemeRule = { token };
            if (foreground !== '') {
                rule.foreground = foreground;
            }
            if (fontStyle !== '') {
                rule.fontStyle = fontStyle;
            }
            byToken.set(token, rule);
        }
    }
    return Array.from(byToken.values());
}

export function themeData(palette: ThemePalette): monaco.editor.IStandaloneThemeData {
    return {
        base: palette.appearance === 'light' ? 'vs' : 'vs-dark',
        inherit: true,
        rules: syntaxRules(palette.syntax),
        colors: { ...palette.editor },
    };
}

// applyThemeCSS writes the palette's UI colors as custom properties. The
// appearance also drives `color-scheme`, so native scrollbars and form controls
// follow the theme.
export function applyThemeCSS(palette: ThemePalette): void {
    const root = document.documentElement;
    for (const [name, value] of Object.entries(palette.ui)) {
        root.style.setProperty(`--${name}`, value);
    }
    root.style.colorScheme = palette.appearance;
}

export function applyUISettings(ui: UISettings): void {
    const root = document.documentElement;
    const family = ui.fontFamily.trim() === '' ? DEFAULT_UI_FONT : ui.fontFamily;
    root.style.setProperty('--ui-font-family', family);
    root.style.setProperty('--ui-font-size', `${ui.fontSize}px`);
}

export async function listThemes(): Promise<ThemeCatalog> {
    const catalog = await ListThemes();
    return {
        themes: (catalog.themes ?? []).map(toThemeInfo),
        errors: catalog.errors ?? [],
    };
}

export async function loadTheme(name: string): Promise<ThemePalette> {
    return toThemePalette(await LoadTheme(name));
}

// listenForConfigChanges runs handler when the themes folder or keybindings
// file changes. Returns an unsubscribe function.
export function listenForConfigChanges(handler: (change: ConfigChange) => void): () => void {
    return onWindowEvent('app:config-changed', (data) => {
        const change = data as ConfigChange | undefined;
        if (change !== undefined) {
            handler(change);
        }
    });
}

export function listenForAppearance(handler: (mode: ThemeMode) => void): () => void {
    return onWindowEvent('app:appearance', (data) => {
        const mode = data as string | undefined;
        if (mode === 'system' || mode === 'light' || mode === 'dark') {
            handler(mode);
        }
    });
}

export function listenForSettingsOverlay(handler: () => void): () => void {
    return onWindowEvent('app:settings-overlay', () => handler());
}
