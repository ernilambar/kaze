import { describe, expect, it, vi } from 'vitest';

const { getSettings, openSettingsFile } = vi.hoisted(() => ({
    getSettings: vi.fn(),
    openSettingsFile: vi.fn(),
}));

vi.mock('../wailsjs/go/main/App', () => ({
    GetSettings: getSettings,
    OpenSettingsFile: openSettingsFile,
}));

import { defaultEditorSettings, loadEditorSettings, openSettingsFile as openSettings } from './settings';

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

describe('openSettingsFile', () => {
    it('asks the backend to open settings.json', async () => {
        openSettingsFile.mockResolvedValue(undefined);

        await openSettings();

        expect(openSettingsFile).toHaveBeenCalledTimes(1);
    });

    it('logs when the backend cannot launch an editor', async () => {
        const error = new Error('no editor configured');
        openSettingsFile.mockRejectedValue(error);
        const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

        await expect(openSettings()).resolves.toBeUndefined();
        expect(consoleError).toHaveBeenCalledWith(
            'Failed to open settings.json.',
            error,
        );

        consoleError.mockRestore();
    });
});
