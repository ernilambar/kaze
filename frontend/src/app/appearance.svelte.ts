import * as monaco from 'monaco-editor';
import { GetKeybindings, SetAppearanceMode, SetViewOption } from '../../bindings/kaze/internal/app/app';
import { COMMANDS } from '../commands';
import { VIEW_OPTIONS, type ViewOption } from '../editorcommands';
import {
    installKeybindings,
    isUsableBinding,
    withDefaultKeybindings,
    type KeyContext,
} from '../keybindings';
import {
    defaultEditorSettings,
    defaultThemeSettings,
    defaultUISettings,
    readSettings,
    saveEditorSettings,
    saveFilesSettings,
    saveSearchExclude,
    saveThemeSettings,
    saveUISettings,
    type EditorSettings,
    type ThemeSettings,
    type UISettings,
} from '../settings';
import { errorMessage } from '../sidebar';
import {
    applyThemeCSS,
    applyUISettings,
    cacheStartupTheme,
    listThemes,
    loadTheme,
    monacoThemeName,
    resolveThemeName,
    systemPrefersDark,
    themeData,
    type ThemeInfo,
    type ThemePalette,
} from '../theme';
import type { WindowContext } from './context.svelte';

export interface AppearanceHost {
    refreshTree: () => void;
    sidebarVisible: () => boolean;
    selectCommand: (id: string) => void;
}

// Appearance holds the settings sections, the themes and the keybindings, and
// applies and saves each change.
export class Appearance {
    private readonly ctx: WindowContext;
    private readonly host: AppearanceHost;

    editorSettings = $state<EditorSettings>({ ...defaultEditorSettings });

    // Editor saves are chained so the last change wins; the version identifies
    // the newest request, whose normalized result is the one applied.
    editorSaveQueue: Promise<void> = Promise.resolve();
    editorSaveVersion = 0;
    themeSettings = $state<ThemeSettings>({ ...defaultThemeSettings });
    uiSettings = $state<UISettings>({ ...defaultUISettings });

    // Excludes are the two glob lists settings.json keeps. The tree and Find in
    // Files both read them in Go; the settings overlay edits them.
    filesExclude = $state<string[]>([]);
    filesTrimTrailingWhitespace = $state(false);
    searchExclude = $state<string[]>([]);
    themes = $state<ThemeInfo[]>([]);
    themeErrors = $state<string[]>([]);
    keybindingError = $state('');
    disposeKeybindings: (() => void) | undefined;

    settingsWarning = $derived(
        this.keybindingError !== '' ? this.keybindingError : (this.themeErrors[0] ?? ''),
    );

    // Theme application: CSS variables cover the app chrome; Monaco gets its
    // own theme definition. A stable name per palette lets a live reload
    // replace the data and re-select the same theme.
    applyThemePalette = (palette: ThemePalette): void => {
        applyThemeCSS(palette);
        cacheStartupTheme(palette);
        monaco.editor.defineTheme(monacoThemeName(palette.name), themeData(palette));
        monaco.editor.setTheme(monacoThemeName(palette.name));
    };

    applyCurrentTheme = async (): Promise<void> => {
        const prefersDark = systemPrefersDark();
        const name = resolveThemeName(this.themeSettings, prefersDark);
        try {
            this.applyThemePalette(await loadTheme(name));
            return;
        } catch (error) {
            console.error('Failed to apply the selected theme.', error);
        }
        const fallback = prefersDark ? defaultThemeSettings.dark : defaultThemeSettings.light;
        try {
            this.applyThemePalette(await loadTheme(fallback));
            this.ctx.notice = `Theme "${name}" is unavailable.`;
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // refreshThemes lists installed themes and repairs a name that no longer
    // exists. The first theme for an appearance is the fallback, so deleting a
    // theme file does not leave the editor stuck on the old palette.
    refreshThemes = async (): Promise<void> => {
        try {
            const catalog = await listThemes();
            this.themes = catalog.themes;
            this.themeErrors = catalog.errors;
            const lightFallback = catalog.themes.find(
                (entry) => entry.appearance === 'light',
            )?.name;
            const darkFallback = catalog.themes.find(
                (entry) => entry.appearance === 'dark',
            )?.name;
            const names = new Set(catalog.themes.map((entry) => entry.name));
            let next = this.themeSettings;
            if (!names.has(next.light) && lightFallback !== undefined) {
                next = { ...next, light: lightFallback };
            }
            if (!names.has(next.dark) && darkFallback !== undefined) {
                next = { ...next, dark: darkFallback };
            }
            if (next.light !== this.themeSettings.light || next.dark !== this.themeSettings.dark) {
                this.themeSettings = next;
                this.themeSettings = await saveThemeSettings(next);
            }
            if (this.themeErrors.length > 0) {
                this.ctx.notice = this.themeErrors[0];
            }
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    changeThemeSettings = async (next: ThemeSettings): Promise<void> => {
        this.themeSettings = next;
        void SetAppearanceMode(next.mode).catch((error) =>
            console.error('Failed to update the Appearance menu.', error),
        );
        await this.applyCurrentTheme();
        try {
            this.themeSettings = await saveThemeSettings(next);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    changeUISettings = async (next: UISettings): Promise<void> => {
        this.uiSettings = next;
        applyUISettings(next);
        try {
            this.uiSettings = await saveUISettings(next);
            applyUISettings(this.uiSettings);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // changeFilesExclude saves the explorer's glob list and rereads the tree:
    // the level already on screen was opened with the old matcher. The update
    // endpoint replaces the whole files section, so the whitespace rule travels
    // with it.
    changeFilesExclude = async (next: string[]): Promise<void> => {
        this.filesExclude = next;
        try {
            const saved = await saveFilesSettings({
                exclude: next,
                trimTrailingWhitespace: this.filesTrimTrailingWhitespace,
            });
            this.filesExclude = saved.exclude;
            this.filesTrimTrailingWhitespace = saved.trimTrailingWhitespace;
            this.host.refreshTree();
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // changeFilesTrimTrailingWhitespace saves the trimming rule without
    // touching the explorer's glob list.
    changeFilesTrimTrailingWhitespace = async (next: boolean): Promise<void> => {
        this.filesTrimTrailingWhitespace = next;
        try {
            const saved = await saveFilesSettings({
                exclude: this.filesExclude,
                trimTrailingWhitespace: next,
            });
            this.filesExclude = saved.exclude;
            this.filesTrimTrailingWhitespace = saved.trimTrailingWhitespace;
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    changeSearchExclude = async (next: string[]): Promise<void> => {
        this.searchExclude = next;
        try {
            this.searchExclude = await saveSearchExclude(next);
        } catch (error) {
            this.ctx.notice = errorMessage(error);
        }
    };

    // reloadSettings applies settings.json after it changed on disk. A broken
    // file keeps the current values and reports why, so a typo mid-edit does
    // not reset everything to defaults. Only sections that differ are applied,
    // so the overlay's own writes coming back through the watcher are no-ops.
    reloadSettings = async (): Promise<void> => {
        let next: Awaited<ReturnType<typeof readSettings>>;
        try {
            next = await readSettings();
        } catch (error) {
            this.ctx.notice = errorMessage(error);
            return;
        }
        const same = (a: unknown, b: unknown): boolean => JSON.stringify(a) === JSON.stringify(b);
        if (!same(next.editor, this.editorSettings)) {
            this.editorSettings = next.editor;
        }
        if (!same(next.ui, this.uiSettings)) {
            this.uiSettings = next.ui;
            applyUISettings(this.uiSettings);
        }
        // The tree on screen was listed with the old matcher.
        if (!same(next.files.exclude, this.filesExclude)) {
            this.filesExclude = next.files.exclude;
            this.host.refreshTree();
        }
        this.filesTrimTrailingWhitespace = next.files.trimTrailingWhitespace;
        if (!same(next.searchExclude, this.searchExclude)) {
            this.searchExclude = next.searchExclude;
        }
        if (!same(next.theme, this.themeSettings)) {
            this.themeSettings = next.theme;
            await this.applyCurrentTheme();
        }
    };

    // keybindingContext is what a `when` expression can test. Focus inside
    // Monaco's textarea still counts as editor focus.
    keybindingContext = (): KeyContext => {
        const focused = document.activeElement;
        return {
            editorFocus: focused instanceof Element && focused.closest('.monaco-editor') !== null,
            sidebarVisible: this.host.sidebarVisible(),
            hasActiveTab: this.ctx.activeId !== '',
            hasDirtyTab: this.ctx.unsaved,
        };
    };

    knownCommand = (id: string): boolean => {
        return COMMANDS.some((command) => command.id === id);
    };

    runKeybindingCommand = (id: string): void => {
        if (!this.knownCommand(id)) {
            this.ctx.notice = `Unknown keybinding command: ${id}`;
            return;
        }
        this.host.selectCommand(id);
    };

    refreshKeybindings = async (): Promise<void> => {
        let state;
        try {
            state = await GetKeybindings();
        } catch (error) {
            this.keybindingError = errorMessage(error);
            return;
        }
        this.keybindingError = state.error;
        // A broken file did not decode; keep the previous listener (or start
        // with the defaults) and native menu rather than silently dropping
        // every shortcut. A file that
        // decoded has an authoritative list even when entries were rejected.
        if (!state.parsed) {
            this.disposeKeybindings ??= installKeybindings(withDefaultKeybindings([]), {
                run: this.runKeybindingCommand,
                context: this.keybindingContext,
            });
            return;
        }
        const bindings = state.bindings ?? [];
        const actionable = bindings.filter((binding) => !binding.command.startsWith('-'));
        const usable = actionable.filter(isUsableBinding);
        if (usable.length < actionable.length) {
            const dropped = actionable.length - usable.length;
            this.keybindingError = `${this.keybindingError === '' ? '' : `${this.keybindingError}; `}${dropped} binding(s) need a modifier or a non-printing key`;
        }
        this.disposeKeybindings?.();
        // installKeybindings skips "-command" removals and unusable entries.
        this.disposeKeybindings = installKeybindings(withDefaultKeybindings(bindings), {
            run: this.runKeybindingCommand,
            context: this.keybindingContext,
        });
        if (this.keybindingError !== '') {
            this.ctx.notice = this.keybindingError;
        }
    };

    // changeEditorSettings applies the whole editor section, saves it, and moves
    // the View menu checkmarks. Saves are chained and only the newest result is
    // applied, so two quick changes cannot write the older values last.
    changeEditorSettings = (next: EditorSettings): void => {
        this.editorSettings = next;
        const version = ++this.editorSaveVersion;
        this.editorSaveQueue = this.editorSaveQueue
            .catch(() => {})
            .then(async () => {
                const saved = await saveEditorSettings(next);
                if (version !== this.editorSaveVersion) {
                    return;
                }
                this.editorSettings = saved;
                for (const option of VIEW_OPTIONS) {
                    void SetViewOption(option.id, saved[option.id]).catch((error) =>
                        console.error('Failed to update the View menu.', error),
                    );
                }
            })
            .catch((error) => {
                if (version === this.editorSaveVersion) {
                    this.ctx.notice = errorMessage(error);
                }
            });
    };

    // setViewOption applies a view toggle, saves it, and moves the View menu's
    // checkmark when the toggle came from the command palette instead.
    setViewOption = (option: ViewOption, checked: boolean): void => {
        if (this.editorSettings[option] === checked) {
            return;
        }
        this.changeEditorSettings({ ...this.editorSettings, [option]: checked });
    };

    toggleViewOption = (option: ViewOption): void => {
        this.setViewOption(option, !this.editorSettings[option]);
    };

    constructor(ctx: WindowContext, host: AppearanceHost) {
        this.ctx = ctx;
        this.host = host;
    }
}
