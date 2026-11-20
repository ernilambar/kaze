import { describe, expect, it, vi } from 'vitest';

const { getSettings, updateEditorSettings, updateFilesSettings, updateSearchSettings } =
    vi.hoisted(() => ({
        getSettings: vi.fn(),
        updateEditorSettings: vi.fn(),
        updateFilesSettings: vi.fn(),
        updateSearchSettings: vi.fn(),
    }));

vi.mock('../bindings/kaze/internal/app/app', () => ({
    GetSettings: getSettings,
    UpdateEditorSettings: updateEditorSettings,
    UpdateFilesSettings: updateFilesSettings,
    UpdateSearchSettings: updateSearchSettings,
}));

import {
    defaultEditorSettings,
    loadEditorSettings,
    loadPreferences,
    readSettings,
    saveEditorSettings,
    saveFilesSettings,
    saveSearchExclude,
} from './settings';

const storedEditor = {
    font_family: 'JetBrains Mono',
    font_size: 16.5,
    font_weight: 300,
    tab_size: 2,
    hard_tabs: true,
    ligatures: true,
    cursor_width: 3,
    cursor_blink: false,
    cursor_surrounding_lines: 2,
    scroll_beyond_last_line: 'half_page',
    ensure_final_newline: true,
    minimap: true,
    bracket_pair_colors: true,
    rulers: [80, 120],
    word_wrap: false,
    invisible_characters: true,
    line_numbers: false,
    indent_guides: false,
    code_folding: false,
    auto_close_brackets: false,
    smart_indent: false,
};

describe('loadEditorSettings', () => {
    it('maps settings.json values onto editor options', async () => {
        getSettings.mockResolvedValue({
            editor: storedEditor,
        });

        await expect(loadEditorSettings()).resolves.toEqual({
            fontFamily: 'JetBrains Mono',
            fontSize: 16.5,
            fontWeight: 300,
            tabSize: 2,
            hardTabs: true,
            ligatures: true,
            cursorWidth: 3,
            cursorBlink: false,
            cursorSurroundingLines: 2,
            scrollBeyondLastLine: 'half_page',
            ensureFinalNewline: true,
            minimap: true,
            bracketPairColors: true,
            rulers: [80, 120],
            wordWrap: false,
            invisibleCharacters: true,
            lineNumbers: false,
            indentGuides: false,
            codeFolding: false,
            autoCloseBrackets: false,
            smartIndent: false,
        });
    });

    it('returns defaults when the backend call fails', async () => {
        const error = new Error('no wails bridge');
        getSettings.mockRejectedValue(error);
        const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

        await expect(loadEditorSettings()).resolves.toEqual(defaultEditorSettings);
        expect(consoleError).toHaveBeenCalledWith(
            'Failed to load settings; using defaults.',
            error,
        );

        consoleError.mockRestore();
    });
});

describe('saveEditorSettings', () => {
    it('sends snake_case fields and returns the normalized result', async () => {
        updateEditorSettings.mockResolvedValue(storedEditor);

        const saved = await saveEditorSettings({ ...defaultEditorSettings });

        expect(updateEditorSettings).toHaveBeenCalledWith(
            expect.objectContaining({
                font_family: 'Menlo',
                tab_size: 4,
                hard_tabs: false,
                word_wrap: true,
            }),
        );
        expect(saved.tabSize).toBe(2);
        expect(saved.wordWrap).toBe(false);
    });
});

describe('loadPreferences', () => {
    it('maps the theme, interface and exclude sections together', async () => {
        getSettings.mockResolvedValue({
            editor: storedEditor,
            theme: { mode: 'dark', light: 'Kaze Light', dark: 'Kaze Dark' },
            ui: {
                font_family: 'Fira Sans',
                font_size: 14,
                tab_icons: false,
                icon_style: 'monochrome',
                sidebar_position: 'right',
            },
            files: {
                exclude: ['/dist', '*.log'],
                trim_trailing_whitespace: true,
                confirm_delete: false,
                show_ignored: false,
            },
            search: { exclude: ['vendor'] },
        });

        await expect(loadPreferences()).resolves.toEqual({
            theme: { mode: 'dark', light: 'Kaze Light', dark: 'Kaze Dark' },
            ui: {
                fontFamily: 'Fira Sans',
                fontSize: 14,
                tabIcons: false,
                iconStyle: 'monochrome',
                sidebarPosition: 'right',
            },
            files: {
                exclude: ['/dist', '*.log'],
                trimTrailingWhitespace: true,
                confirmDelete: false,
                showIgnored: false,
            },
            searchExclude: ['vendor'],
        });
    });
});

describe('files and search saves', () => {
    it('sends both members of the files section and returns the normalized result', async () => {
        updateFilesSettings.mockResolvedValue({
            exclude: ['node_modules'],
            trim_trailing_whitespace: true,
            confirm_delete: false,
            show_ignored: false,
        });
        updateSearchSettings.mockResolvedValue({ exclude: ['vendor'] });

        await expect(
            saveFilesSettings({
                exclude: [' node_modules ', 'node_modules'],
                trimTrailingWhitespace: true,
                confirmDelete: false,
                showIgnored: false,
            }),
        ).resolves.toEqual({
            exclude: ['node_modules'],
            trimTrailingWhitespace: true,
            confirmDelete: false,
            showIgnored: false,
        });
        expect(updateFilesSettings).toHaveBeenCalledWith({
            exclude: [' node_modules ', 'node_modules'],
            trim_trailing_whitespace: true,
            confirm_delete: false,
            show_ignored: false,
        });

        await expect(saveSearchExclude(['vendor'])).resolves.toEqual(['vendor']);
        expect(updateSearchSettings).toHaveBeenCalledWith({ exclude: ['vendor'] });
    });
});

describe('readSettings', () => {
    it('rejects a broken file instead of falling back to defaults', async () => {
        getSettings.mockRejectedValue(new Error('parse settings.json: unexpected end of JSON input'));

        await expect(readSettings()).rejects.toThrow('parse settings.json');
    });

    it('maps the editor section with the preferences', async () => {
        getSettings.mockResolvedValue({
            editor: storedEditor,
            theme: { mode: 'dark', light: 'Kaze Light', dark: 'Kaze Dark' },
            ui: {},
            files: { exclude: [], trim_trailing_whitespace: false },
            search: { exclude: ['dist'] },
        });

        const settings = await readSettings();
        expect(settings.editor.fontFamily).toBe('JetBrains Mono');
        expect(settings.searchExclude).toEqual(['dist']);
    });
});
