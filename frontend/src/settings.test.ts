import { describe, expect, it, vi } from 'vitest';

const { getSettings } = vi.hoisted(() => ({ getSettings: vi.fn() }));

vi.mock('../wailsjs/go/main/App', () => ({
    GetSettings: getSettings,
}));

import { defaultEditorSettings, loadEditorSettings } from './settings';

describe('loadEditorSettings', () => {
    it('maps settings.json values onto editor options', async () => {
        getSettings.mockResolvedValue({
            editor: {
                font_family: 'JetBrains Mono',
                font_size: 16.5,
                font_weight: 300,
            },
        });

        await expect(loadEditorSettings()).resolves.toEqual({
            fontFamily: 'JetBrains Mono',
            fontSize: 16.5,
            fontWeight: 300,
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
