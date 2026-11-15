import { describe, expect, it, vi } from 'vitest';

const { getSettings, updateEditorSettings } = vi.hoisted(() => ({
    getSettings: vi.fn(),
    updateEditorSettings: vi.fn(),
}));

vi.mock('../bindings/kova/app', () => ({
    GetSettings: getSettings,
    UpdateEditorSettings: updateEditorSettings,
}));

import {
    defaultEditorSettings,
    loadEditorSettings,
    saveEditorSettings,
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
    scroll_beyond_last_line: true,
    ensure_final_newline: true,
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
            sidebar: { folders: [] },
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
            scrollBeyondLastLine: true,
            ensureFinalNewline: true,
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
