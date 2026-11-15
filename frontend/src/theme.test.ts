import { describe, expect, it, beforeEach } from 'vitest';
import type * as models from '../bindings/kova/models';
import { defaultThemeSettings } from './settings';
import {
    applyStartupTheme,
    applyThemeCSS,
    applyUISettings,
    cacheStartupTheme,
    isHexColor,
    monacoThemeName,
    readStartupTheme,
    resolveThemeName,
    STARTUP_THEME_KEY,
    startupThemeData,
    syntaxRules,
    themeData,
    toMonacoColor,
    toThemePalette,
} from './theme';

function palette(overrides: Partial<models.ThemePalette> = {}): models.ThemePalette {
    return {
        family: 'Family',
        name: 'Test',
        appearance: 'dark',
        ui: { bg: '#101010', fg: '#eeeeee', border: 'not-a-color' },
        editor: { 'editor.background': '#101010', 'editor.foreground': '#eeeeee' },
        syntax: {},
        ...overrides,
    };
}

describe('resolveThemeName', () => {
    it('follows the mode', () => {
        const theme = { ...defaultThemeSettings, light: 'Light X', dark: 'Dark X' };
        expect(resolveThemeName({ ...theme, mode: 'light' }, true)).toBe('Light X');
        expect(resolveThemeName({ ...theme, mode: 'dark' }, false)).toBe('Dark X');
    });

    it('uses the system preference in system mode', () => {
        const theme = { ...defaultThemeSettings, light: 'Light X', dark: 'Dark X' };
        expect(resolveThemeName({ ...theme, mode: 'system' }, true)).toBe('Dark X');
        expect(resolveThemeName({ ...theme, mode: 'system' }, false)).toBe('Light X');
    });
});

describe('toMonacoColor', () => {
    it('expands short hex and drops alpha', () => {
        expect(toMonacoColor('#abc')).toBe('AABBCC');
        expect(toMonacoColor('#abcd')).toBe('AABBCC');
        expect(toMonacoColor('#aabbcc')).toBe('AABBCC');
        expect(toMonacoColor('#aabbccdd')).toBe('AABBCC');
    });

    it('rejects values Monaco cannot parse', () => {
        expect(toMonacoColor('red')).toBe('');
        expect(toMonacoColor('#12')).toBe('');
        expect(toMonacoColor('#gggggg')).toBe('');
    });
});

describe('isHexColor', () => {
    it('accepts the lengths Zed writes', () => {
        expect(isHexColor('#fff')).toBe(true);
        expect(isHexColor('#ffff')).toBe(true);
        expect(isHexColor('#ffffff')).toBe(true);
        expect(isHexColor('#ffffffff')).toBe(true);
        expect(isHexColor('#fffff')).toBe(false);
    });
});

describe('syntaxRules', () => {
    it('maps semantic captures onto Monaco token names', () => {
        const rules = syntaxRules({
            comment: { color: '#008000', fontStyle: '', fontWeight: 0 },
            keyword: { color: '#0000ff', fontStyle: '', fontWeight: 0 },
            'string.regex': { color: '#811f3f', fontStyle: '', fontWeight: 0 },
            punctuation: { color: '#111111', fontStyle: '', fontWeight: 0 },
            'punctuation.bracket': { color: '#222222', fontStyle: '', fontWeight: 0 },
        });
        const byToken = new Map(rules.map((rule) => [rule.token, rule]));
        expect(byToken.get('comment')?.foreground).toBe('008000');
        expect(byToken.get('keyword')?.foreground).toBe('0000FF');
        expect(byToken.get('regexp')?.foreground).toBe('811F3F');
        expect(byToken.get('delimiter')?.foreground).toBe('111111');
        expect(byToken.get('delimiter.bracket')?.foreground).toBe('222222');
    });

    it('turns font style and weight into Monaco fontStyle', () => {
        const rules = syntaxRules({
            comment: { color: '#008000', fontStyle: 'italic', fontWeight: 0 },
            'emphasis.strong': { color: '#000000', fontStyle: '', fontWeight: 700 },
        });
        const byToken = new Map(rules.map((rule) => [rule.token, rule]));
        expect(byToken.get('comment')?.fontStyle).toBe('italic');
        expect(byToken.get('strong')?.fontStyle).toBe('bold');
    });

    it('ignores captures Monaco never emits', () => {
        const rules = syntaxRules({
            function: { color: '#ff0000', fontStyle: '', fontWeight: 0 },
            predictive: { color: '#00ff00', fontStyle: '', fontWeight: 0 },
        });
        expect(rules).toEqual([]);
    });
});

describe('themeData', () => {
    it('picks the base theme from the appearance and keeps only valid colors', () => {
        const data = themeData(toThemePalette(palette()));
        expect(data.base).toBe('vs-dark');
        expect(data.inherit).toBe(true);
        expect(data.colors['editor.background']).toBe('#101010');
        expect(data.colors['editor.foreground']).toBe('#eeeeee');
    });

    it('turns a light palette into the light base', () => {
        const data = themeData(toThemePalette(palette({ appearance: 'light' })));
        expect(data.base).toBe('vs');
    });
});

describe('toThemePalette', () => {
    it('drops malformed UI and editor values', () => {
        const converted = toThemePalette(palette());
        expect(converted.ui).toEqual({ bg: '#101010', fg: '#eeeeee' });
        expect(converted.editor).toEqual({
            'editor.background': '#101010',
            'editor.foreground': '#eeeeee',
        });
    });
});

describe('CSS application', () => {
    it('sets custom properties and the color scheme', () => {
        applyThemeCSS(toThemePalette(palette({ appearance: 'light' })));
        const root = document.documentElement;
        expect(root.style.getPropertyValue('--bg')).toBe('#101010');
        expect(root.style.getPropertyValue('--fg')).toBe('#eeeeee');
        expect(root.style.colorScheme).toBe('light');
    });

    it('uses the system stack for an empty UI font and applies the size', () => {
        applyUISettings({
            fontFamily: '',
            fontSize: 15,
            tabIcons: true,
            iconStyle: 'colorful',
            sidebarPosition: 'left',
        });
        const root = document.documentElement;
        expect(root.style.getPropertyValue('--ui-font-family')).toContain('BlinkMacSystemFont');
        expect(root.style.getPropertyValue('--ui-font-size')).toBe('15px');

        applyUISettings({
            fontFamily: 'Fira Sans',
            fontSize: 12,
            tabIcons: true,
            iconStyle: 'colorful',
            sidebarPosition: 'left',
        });
        expect(root.style.getPropertyValue('--ui-font-family')).toBe('Fira Sans');
        expect(root.style.getPropertyValue('--ui-font-size')).toBe('12px');
    });
});

describe('monacoThemeName', () => {
    it('makes a stable id from the variant name', () => {
        expect(monacoThemeName('Kova Dark')).toBe('theme-Kova-Dark');
        expect(monacoThemeName('a/b:c')).toBe('theme-a-b-c');
        expect(monacoThemeName('Kova Dark')).toBe(monacoThemeName('Kova Dark'));
    });
});

describe('startup theme cache', () => {
    beforeEach(() => {
        localStorage.clear();
        document.documentElement.style.cssText = '';
    });

    it('round-trips the surface colours the first frame needs', () => {
        cacheStartupTheme(toThemePalette(palette()));
        expect(JSON.parse(localStorage.getItem(STARTUP_THEME_KEY) ?? 'null')).toEqual({
            appearance: 'dark',
            background: '#101010',
            foreground: '#eeeeee',
        });
        expect(readStartupTheme()).toEqual({
            appearance: 'dark',
            background: '#101010',
            foreground: '#eeeeee',
        });
    });

    it('ignores a cached value it cannot trust', () => {
        expect(readStartupTheme()).toBeNull();
        localStorage.setItem(STARTUP_THEME_KEY, 'not json');
        expect(readStartupTheme()).toBeNull();
        localStorage.setItem(
            STARTUP_THEME_KEY,
            JSON.stringify({ appearance: 'dark', background: 'white' }),
        );
        expect(readStartupTheme()).toBeNull();
        localStorage.setItem(
            STARTUP_THEME_KEY,
            JSON.stringify({ appearance: 'dark', background: '#123456' }),
        );
        expect(readStartupTheme()).toEqual({
            appearance: 'dark',
            background: '#123456',
            foreground: '',
        });
    });

    it('skips a palette with no background instead of caching a half theme', () => {
        cacheStartupTheme(toThemePalette(palette({ ui: {} })));
        expect(localStorage.getItem(STARTUP_THEME_KEY)).toBeNull();
    });

    it('paints the cached colours and the appearance', () => {
        applyStartupTheme({ appearance: 'light', background: '#fafafa', foreground: '#111111' });
        const root = document.documentElement;
        expect(root.style.getPropertyValue('--bg')).toBe('#fafafa');
        expect(root.style.getPropertyValue('--fg')).toBe('#111111');
        expect(root.style.colorScheme).toBe('light');
    });

    it('gives Monaco a background for the first frame', () => {
        const dark = startupThemeData({
            appearance: 'dark',
            background: '#101010',
            foreground: '#eeeeee',
        });
        expect(dark.base).toBe('vs-dark');
        expect(dark.colors['editor.background']).toBe('#101010');
        expect(
            startupThemeData({ appearance: 'light', background: '#ffffff', foreground: '' }).base,
        ).toBe('vs');
    });
});
