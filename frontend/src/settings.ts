import type { main } from '../wailsjs/go/models';
import { GetSettings, OpenSettingsFile } from '../wailsjs/go/main/App';

export interface EditorSettings {
    fontFamily: string;
    fontSize: number;
    fontWeight: number;
}

// Mirrors defaultSettings in settings.go. Used only when the backend call
// fails, e.g. running `pnpm dev` outside a Wails window.
export const defaultEditorSettings: EditorSettings = {
    fontFamily: 'Menlo',
    fontSize: 13,
    fontWeight: 400,
};

function toEditorSettings(settings: main.Settings): EditorSettings {
    return {
        fontFamily: settings.editor.font_family,
        fontSize: settings.editor.font_size,
        fontWeight: settings.editor.font_weight,
    };
}

export async function loadEditorSettings(): Promise<EditorSettings> {
    try {
        return toEditorSettings(await GetSettings());
    } catch (error) {
        console.error('Failed to load settings; using defaults.', error);
        return { ...defaultEditorSettings };
    }
}

// Opens settings.json in $VISUAL/$EDITOR (or the OS default handler); the
// backend owns editor selection and process launching.
export async function openSettingsFile(): Promise<void> {
    try {
        await OpenSettingsFile();
    } catch (error) {
        console.error('Failed to open settings.json.', error);
    }
}
