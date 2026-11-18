import { onWindowEvent } from './windowstate';
import type * as monaco from 'monaco-editor';
import { ListThemes, LoadTheme } from '../bindings/kaze/app';
import type * as models from '../bindings/kaze/internal/themes/models';
import {
    DEFAULT_UI_FONT,
    type ThemeMode,
    type ThemeSettings,
    type UISettings,
} from './settings';

// ThemeAppearance is the variant's own light/dark label, not the selected mode.
export type ThemeAppearance = 'light' | 'dark';

export interface ThemePreview {
    background: string;
    foreground: string;
    accent: string;
    keyword: string;
    string: string;
    comment: string;
}

export interface ThemeInfo {
    family: string;
    name: string;
    appearance: ThemeAppearance;
    // Resolved swatches for the settings picker. Empty strings mean the theme
    // file did not provide a usable color; CSS falls back to the live palette.
    preview: ThemePreview;
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
    settings: boolean;
}

function appearanceOf(value: string): ThemeAppearance {
    return value === 'light' ? 'light' : 'dark';
}

export function toThemeInfo(summary: models.ThemeSummary): ThemeInfo {
    return {
        family: summary.family,
        name: summary.name,
        appearance: appearanceOf(summary.appearance),
        preview: {
            background: summary.preview?.background ?? '',
            foreground: summary.preview?.foreground ?? '',
            accent: summary.preview?.accent ?? '',
            keyword: summary.preview?.keyword ?? '',
            string: summary.preview?.string ?? '',
            comment: summary.preview?.comment ?? '',
        },
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

// monacoThemeName gives a palette its Monaco theme id. It is stable per name, so
// a live reload replaces the definition instead of piling up new ids.
export function monacoThemeName(name: string): string {
    return `theme-${name.replace(/[^A-Za-z0-9_-]/g, '-')}`;
}

// StartupTheme is the little a first frame needs: the surface colours that were
// last on screen. Settings and theme files are read asynchronously after the
// window is already visible, so without this the app showed its default light
// surfaces and then switched to the saved theme.
export interface StartupTheme {
    appearance: ThemeAppearance;
    background: string;
    foreground: string;
}

// STARTUP_THEME_KEY is read by an inline script in index.html as well, which is
// why the stored shape stays this small.
export const STARTUP_THEME_KEY = 'kaze.theme.startup';

// STARTUP_THEME_ID is the Monaco theme the cache paints before anything is
// loaded. The real palette replaces it once its file has been read.
export const STARTUP_THEME_ID = 'kaze-startup';

export function cacheStartupTheme(palette: ThemePalette): void {
    const background = palette.ui['bg'];
    if (background === undefined) {
        return;
    }
    const theme: StartupTheme = {
        appearance: palette.appearance,
        background,
        foreground: palette.ui['fg'] ?? '',
    };
    try {
        window.localStorage.setItem(STARTUP_THEME_KEY, JSON.stringify(theme));
    } catch (error) {
        console.error('Failed to cache the theme for startup.', error);
    }
}

// readStartupTheme returns the cached palette, or null when there is none or it
// cannot be trusted. A stale value only affects the first frame.
export function readStartupTheme(): StartupTheme | null {
    let raw: string | null;
    try {
        raw = window.localStorage.getItem(STARTUP_THEME_KEY);
    } catch (error) {
        return null;
    }
    if (raw === null) {
        return null;
    }
    let parsed: unknown;
    try {
        parsed = JSON.parse(raw);
    } catch (error) {
        return null;
    }
    if (typeof parsed !== 'object' || parsed === null) {
        return null;
    }
    const candidate = parsed as Partial<StartupTheme>;
    if (candidate.appearance !== 'light' && candidate.appearance !== 'dark') {
        return null;
    }
    if (typeof candidate.background !== 'string' || !isHexColor(candidate.background)) {
        return null;
    }
    return {
        appearance: candidate.appearance,
        background: candidate.background,
        foreground:
            typeof candidate.foreground === 'string' && isHexColor(candidate.foreground)
                ? candidate.foreground
                : '',
    };
}

// applyStartupTheme paints the cached colours on the document, before the app
// mounts and before the saved theme has been read.
export function applyStartupTheme(theme: StartupTheme): void {
    const root = document.documentElement;
    root.style.setProperty('--bg', theme.background);
    if (theme.foreground !== '') {
        root.style.setProperty('--fg', theme.foreground);
    }
    root.style.colorScheme = theme.appearance;
}

// startupThemeData is the Monaco theme for the cached palette. Monaco starts on
// its light default otherwise, which is the white the editor flashed on launch.
export function startupThemeData(theme: StartupTheme): monaco.editor.IStandaloneThemeData {
    return {
        base: theme.appearance === 'light' ? 'vs' : 'vs-dark',
        inherit: true,
        rules: [],
        colors: { 'editor.background': theme.background },
    };
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

// listenForConfigChanges runs handler when the themes folder, keybindings or
// settings file changes. Returns an unsubscribe function.
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
