<script lang="ts">
    import { onMount } from 'svelte';
    import type { ThemeInfo } from './theme';
    import type {
        EditorSettings,
        ThemeMode,
        ThemeSettings,
        UISettings,
    } from './settings';

    type Section = 'editor' | 'appearance' | 'interface' | 'files' | 'sidebar' | 'advanced';
    type NumericEditorKey =
        | 'fontSize'
        | 'fontWeight'
        | 'tabSize'
        | 'cursorWidth'
        | 'cursorSurroundingLines';
    type BooleanEditorKey = Exclude<keyof EditorSettings, 'fontFamily' | NumericEditorKey>;

    interface Props {
        editor: EditorSettings;
        theme: ThemeSettings;
        ui: UISettings;
        themes: ThemeInfo[];
        filesExclude: string[];
        searchExclude: string[];
        folders: string[];
        sidebarVisible: boolean;
        autoReveal: boolean;
        warning?: string;
        onEditor: (next: EditorSettings) => void;
        onTheme: (next: ThemeSettings) => void;
        onUI: (next: UISettings) => void;
        onFilesExclude: (next: string[]) => void;
        onSearchExclude: (next: string[]) => void;
        onSidebarVisible: (visible: boolean) => void;
        onAutoReveal: (autoReveal: boolean) => void;
        onAddFolder: () => void;
        onRemoveFolder: (path: string) => void;
        onOpenSettings: () => void;
        onOpenDefaults: () => void;
        onOpenThemes: () => void;
        onOpenKeybindings: () => void;
        onClose: () => void;
    }

    let {
        editor,
        theme,
        ui,
        themes,
        filesExclude,
        searchExclude,
        folders,
        sidebarVisible,
        autoReveal,
        warning = '',
        onEditor,
        onTheme,
        onUI,
        onFilesExclude,
        onSearchExclude,
        onSidebarVisible,
        onAutoReveal,
        onAddFolder,
        onRemoveFolder,
        onOpenSettings,
        onOpenDefaults,
        onOpenThemes,
        onOpenKeybindings,
        onClose,
    }: Props = $props();

    const SECTIONS: readonly { id: Section; label: string }[] = [
        { id: 'editor', label: 'Editor' },
        { id: 'appearance', label: 'Appearance' },
        { id: 'interface', label: 'Interface' },
        { id: 'files', label: 'Files' },
        { id: 'sidebar', label: 'Sidebar' },
        { id: 'advanced', label: 'Advanced' },
    ];

    // VIEW_TOGGLES are the editor options the View menu mirrors. They stay in
    // the same order here so the panel and the native menu read alike.
    const VIEW_TOGGLES: readonly { key: BooleanEditorKey; label: string }[] = [
        { key: 'wordWrap', label: 'Word wrap' },
        { key: 'invisibleCharacters', label: 'Invisible characters' },
        { key: 'lineNumbers', label: 'Line numbers' },
        { key: 'indentGuides', label: 'Indent guides' },
        { key: 'codeFolding', label: 'Code folding' },
        { key: 'autoCloseBrackets', label: 'Auto-close brackets and quotes' },
        { key: 'smartIndent', label: 'Smart indent' },
    ];

    let dialog = $state<HTMLElement>();
    let section = $state<Section>('editor');

    let lightThemes = $derived(themes.filter((entry) => entry.appearance === 'light'));
    let darkThemes = $derived(themes.filter((entry) => entry.appearance === 'dark'));

    function updateEditor<K extends keyof EditorSettings>(
        key: K,
        value: EditorSettings[K],
    ): void {
        onEditor({ ...editor, [key]: value });
    }

    function commitText(event: Event, key: 'fontFamily'): void {
        updateEditor(key, (event.currentTarget as HTMLInputElement).value);
    }

    // commitNumber clamps on commit and writes the clamped value back to the
    // input, so an out-of-range or empty field never stays on screen.
    function commitNumber(
        event: Event,
        key: NumericEditorKey,
        min: number,
        max: number,
        integer = false,
    ): void {
        const input = event.currentTarget as HTMLInputElement;
        const parsed = Number(input.value);
        let value = Number.isFinite(parsed)
            ? Math.min(max, Math.max(min, parsed))
            : editor[key];
        if (integer) {
            value = Math.round(value);
        }
        input.value = String(value);
        if (value !== editor[key]) {
            updateEditor(key, value);
        }
    }

    function commitToggle(event: Event, key: BooleanEditorKey): void {
        updateEditor(key, (event.currentTarget as HTMLInputElement).checked);
    }

    function setMode(mode: string): void {
        if (mode === 'system' || mode === 'light' || mode === 'dark') {
            onTheme({ ...theme, mode: mode as ThemeMode });
        }
    }

    function setLight(name: string): void {
        onTheme({ ...theme, light: name });
    }

    function setDark(name: string): void {
        onTheme({ ...theme, dark: name });
    }

    function setUIFontFamily(value: string): void {
        onUI({ ...ui, fontFamily: value });
    }

    function setUIFontSize(value: string): void {
        const parsed = Number(value);
        if (Number.isFinite(parsed) && parsed > 0) {
            onUI({ ...ui, fontSize: parsed });
        }
    }

    function setTabIcons(checked: boolean): void {
        onUI({ ...ui, tabIcons: checked });
    }

    function setIconStyle(value: string): void {
        onUI({ ...ui, iconStyle: value === 'monochrome' ? 'monochrome' : 'colorful' });
    }

    function setSidebarPosition(value: string): void {
        onUI({ ...ui, sidebarPosition: value === 'right' ? 'right' : 'left' });
    }

    // patterns parses a one-per-line list. The backend trims, deduplicates and
    // drops blanks too; doing it here keeps the field stable while typing.
    function patterns(text: string): string[] {
        return text
            .split('\n')
            .map((line) => line.trim())
            .filter((line) => line !== '');
    }

    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onClose();
        }
    }

    onMount(() => dialog?.focus());
</script>

<svelte:window onkeydown={onKeydown} />

<div class="settings">
    <button
        class="settings__backdrop"
        aria-label="Close Settings"
        tabindex="-1"
        onclick={onClose}
    ></button>
    <div
        class="settings__dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
        tabindex="-1"
        bind:this={dialog}
    >
        <header class="settings__head">
            <h2 id="settings-title">Settings</h2>
            <button class="settings__close" aria-label="Close Settings" onclick={onClose}>
                <svg aria-hidden="true" height="9" viewBox="0 0 9 9" width="9">
                    <path d="M1 1 L8 8 M8 1 L1 8" fill="none" stroke="currentColor" stroke-width="1.2" />
                </svg>
            </button>
        </header>

        <div class="settings__layout">
            <nav class="settings__nav" aria-label="Settings sections">
                {#each SECTIONS as entry (entry.id)}
                    <button
                        type="button"
                        class="settings__nav-item"
                        class:settings__nav-item--active={section === entry.id}
                        aria-current={section === entry.id ? 'page' : undefined}
                        onclick={() => (section = entry.id)}
                    >
                        {entry.label}
                    </button>
                {/each}
            </nav>

            <div class="settings__body">
                {#if section === 'editor'}
                    <section class="settings__section">
                        <h3>Typography</h3>
                        <label class="settings__row">
                            <span>Font family</span>
                            <input
                                type="text"
                                value={editor.fontFamily}
                                placeholder="Menlo"
                                spellcheck="false"
                                onchange={(event) => commitText(event, 'fontFamily')}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Font size</span>
                            <input
                                type="number"
                                min="6"
                                max="72"
                                step="0.5"
                                value={editor.fontSize}
                                onchange={(event) =>
                                    commitNumber(event, 'fontSize', 6, 72)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Font weight</span>
                            <input
                                type="number"
                                min="100"
                                max="900"
                                step="10"
                                value={editor.fontWeight}
                                onchange={(event) =>
                                    commitNumber(event, 'fontWeight', 100, 900)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Ligatures</span>
                            <input
                                type="checkbox"
                                checked={editor.ligatures}
                                onchange={(event) => commitToggle(event, 'ligatures')}
                            />
                        </label>
                    </section>

                    <section class="settings__section">
                        <h3>Indentation</h3>
                        <label class="settings__row">
                            <span>Tab size</span>
                            <input
                                type="number"
                                min="1"
                                max="12"
                                step="1"
                                value={editor.tabSize}
                                onchange={(event) =>
                                    commitNumber(event, 'tabSize', 1, 12, true)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Use hard tabs</span>
                            <input
                                type="checkbox"
                                checked={editor.hardTabs}
                                onchange={(event) => commitToggle(event, 'hardTabs')}
                            />
                        </label>
                    </section>

                    <section class="settings__section">
                        <h3>Cursor</h3>
                        <label class="settings__row">
                            <span>Cursor width</span>
                            <input
                                type="number"
                                min="1"
                                max="20"
                                step="1"
                                value={editor.cursorWidth}
                                onchange={(event) =>
                                    commitNumber(event, 'cursorWidth', 1, 20, true)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Cursor blinks</span>
                            <input
                                type="checkbox"
                                checked={editor.cursorBlink}
                                onchange={(event) => commitToggle(event, 'cursorBlink')}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Surrounding lines</span>
                            <input
                                type="number"
                                min="0"
                                max="30"
                                step="1"
                                value={editor.cursorSurroundingLines}
                                onchange={(event) =>
                                    commitNumber(event, 'cursorSurroundingLines', 0, 30, true)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Scroll beyond last line</span>
                            <input
                                type="checkbox"
                                checked={editor.scrollBeyondLastLine}
                                onchange={(event) =>
                                    commitToggle(event, 'scrollBeyondLastLine')}
                            />
                        </label>
                    </section>

                    <section class="settings__section">
                        <h3>View</h3>
                        {#each VIEW_TOGGLES as toggle (toggle.key)}
                            <label class="settings__row">
                                <span>{toggle.label}</span>
                                <input
                                    type="checkbox"
                                    checked={editor[toggle.key]}
                                    onchange={(event) => commitToggle(event, toggle.key)}
                                />
                            </label>
                        {/each}
                    </section>

                    <section class="settings__section">
                        <h3>Saving</h3>
                        <label class="settings__row">
                            <span>Ensure a final newline</span>
                            <input
                                type="checkbox"
                                checked={editor.ensureFinalNewline}
                                onchange={(event) =>
                                    commitToggle(event, 'ensureFinalNewline')}
                            />
                        </label>
                    </section>
                {:else if section === 'appearance'}
                    <section class="settings__section">
                        <h3>Appearance</h3>
                        <label class="settings__row">
                            <span>Mode</span>
                            <select
                                value={theme.mode}
                                onchange={(event) => setMode(event.currentTarget.value)}
                            >
                                <option value="system">System</option>
                                <option value="light">Light</option>
                                <option value="dark">Dark</option>
                            </select>
                        </label>
                        <label class="settings__row">
                            <span>Light theme</span>
                            <select
                                value={theme.light}
                                onchange={(event) => setLight(event.currentTarget.value)}
                            >
                                {#each lightThemes as entry (entry.name)}
                                    <option value={entry.name}>{entry.family} — {entry.name}</option>
                                {/each}
                            </select>
                        </label>
                        <label class="settings__row">
                            <span>Dark theme</span>
                            <select
                                value={theme.dark}
                                onchange={(event) => setDark(event.currentTarget.value)}
                            >
                                {#each darkThemes as entry (entry.name)}
                                    <option value={entry.name}>{entry.family} — {entry.name}</option>
                                {/each}
                            </select>
                        </label>
                    </section>
                {:else if section === 'interface'}
                    <section class="settings__section">
                        <h3>Interface</h3>
                        <label class="settings__row">
                            <span>UI font</span>
                            <input
                                type="text"
                                value={ui.fontFamily}
                                placeholder="System font"
                                spellcheck="false"
                                onchange={(event) =>
                                    setUIFontFamily(event.currentTarget.value)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>UI size</span>
                            <input
                                type="number"
                                min="9"
                                max="48"
                                step="1"
                                value={ui.fontSize}
                                onchange={(event) =>
                                    setUIFontSize(event.currentTarget.value)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Tab icons</span>
                            <input
                                type="checkbox"
                                checked={ui.tabIcons}
                                onchange={(event) => setTabIcons(event.currentTarget.checked)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Icon style</span>
                            <select
                                value={ui.iconStyle}
                                onchange={(event) => setIconStyle(event.currentTarget.value)}
                            >
                                <option value="colorful">Colorful</option>
                                <option value="monochrome">Monochrome</option>
                            </select>
                        </label>
                        <label class="settings__row">
                            <span>Sidebar side</span>
                            <select
                                value={ui.sidebarPosition}
                                onchange={(event) =>
                                    setSidebarPosition(event.currentTarget.value)}
                            >
                                <option value="left">Left</option>
                                <option value="right">Right</option>
                            </select>
                        </label>
                    </section>
                {:else if section === 'files'}
                    <section class="settings__section">
                        <h3>Explorer excludes</h3>
                        <p class="settings__hint">
                            One glob per line. A bare name matches at any depth, a leading
                            slash anchors it to a sidebar root, and a trailing slash matches
                            folders only.
                        </p>
                        <textarea
                            class="settings__patterns"
                            rows="6"
                            spellcheck="false"
                            value={filesExclude.join('\n')}
                            onchange={(event) =>
                                onFilesExclude(patterns(event.currentTarget.value))}
                        ></textarea>
                    </section>
                    <section class="settings__section">
                        <h3>Search excludes</h3>
                        <p class="settings__hint">
                            Find in Files skips these globs as well as the explorer's list
                            and every .gitignore it walks past.
                        </p>
                        <textarea
                            class="settings__patterns"
                            rows="6"
                            spellcheck="false"
                            value={searchExclude.join('\n')}
                            onchange={(event) =>
                                onSearchExclude(patterns(event.currentTarget.value))}
                        ></textarea>
                    </section>
                {:else if section === 'sidebar'}
                    <section class="settings__section">
                        <h3>Sidebar</h3>
                        <label class="settings__row">
                            <span>Show the sidebar</span>
                            <input
                                type="checkbox"
                                checked={sidebarVisible}
                                onchange={(event) =>
                                    onSidebarVisible(event.currentTarget.checked)}
                            />
                        </label>
                        <label class="settings__row">
                            <span>Reveal the active file</span>
                            <input
                                type="checkbox"
                                checked={autoReveal}
                                onchange={(event) =>
                                    onAutoReveal(event.currentTarget.checked)}
                            />
                        </label>
                    </section>
                    <section class="settings__section">
                        <h3>Folders</h3>
                        {#if folders.length === 0}
                            <p class="settings__hint">No folders are open.</p>
                        {:else}
                            <ul class="settings__list">
                                {#each folders as folder (folder)}
                                    <li class="settings__list-item">
                                        <span class="settings__path" title={folder}>{folder}</span>
                                        <button
                                            type="button"
                                            class="settings__icon-button"
                                            aria-label={`Remove ${folder}`}
                                            onclick={() => onRemoveFolder(folder)}
                                        >
                                            <svg
                                                aria-hidden="true"
                                                height="9"
                                                viewBox="0 0 9 9"
                                                width="9"
                                            >
                                                <path
                                                    d="M1 1 L8 8 M8 1 L1 8"
                                                    fill="none"
                                                    stroke="currentColor"
                                                    stroke-width="1.2"
                                                />
                                            </svg>
                                        </button>
                                    </li>
                                {/each}
                            </ul>
                        {/if}
                        <div class="settings__buttons">
                            <button type="button" onclick={onAddFolder}>Add Folder…</button>
                        </div>
                    </section>
                {:else}
                    <section class="settings__section">
                        <h3>Settings files</h3>
                        <p class="settings__hint">
                            These sections write to settings.json as you change them. The
                            file and the editor defaults are still available for reference.
                        </p>
                        <div class="settings__buttons">
                            <button type="button" onclick={onOpenSettings}>Settings JSON</button>
                            <button type="button" onclick={onOpenDefaults}>Default Settings</button>
                            <button type="button" onclick={onOpenThemes}>Themes Folder</button>
                            <button type="button" onclick={onOpenKeybindings}>Keybindings</button>
                        </div>
                    </section>
                {/if}

                {#if warning !== ''}
                    <p class="settings__warning">{warning}</p>
                {/if}
            </div>
        </div>
    </div>
</div>

<style>
    .settings {
        position: fixed;
        inset: 0;
        z-index: 100;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .settings__backdrop {
        position: absolute;
        inset: 0;
        border: none;
        background: rgba(0, 0, 0, 0.35);
        cursor: default;
    }

    .settings__dialog {
        position: relative;
        display: flex;
        flex-direction: column;
        width: min(760px, calc(100vw - 48px));
        height: min(620px, calc(100vh - 64px));
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg-raised);
        box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
        outline: none;
    }

    .settings__head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        padding: 12px 16px;
        border-bottom: 1px solid var(--border);
    }

    .settings__head h2 {
        margin: 0;
        font-size: var(--ui-font-size);
        font-weight: 600;
    }

    .settings__close {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        padding: 0;
        border: none;
        border-radius: 3px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
    }

    .settings__close:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__layout {
        display: flex;
        flex: 1 1 auto;
        min-height: 0;
    }

    .settings__nav {
        display: flex;
        flex-direction: column;
        gap: 2px;
        flex: 0 0 150px;
        padding: 10px 8px;
        border-right: 1px solid var(--border);
        overflow-y: auto;
    }

    .settings__nav-item {
        padding: 5px 8px;
        border: none;
        border-radius: 4px;
        background: none;
        color: var(--fg-muted);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .settings__nav-item:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__nav-item--active {
        background: var(--bg-hover);
        color: var(--fg);
        font-weight: 600;
    }

    .settings__body {
        flex: 1 1 auto;
        min-width: 0;
        padding: 4px 16px 16px;
        overflow-y: auto;
    }

    .settings__section {
        padding-top: 14px;
    }

    .settings__section h3 {
        margin: 0 0 8px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        font-weight: 600;
        letter-spacing: 0.06em;
        text-transform: uppercase;
    }

    .settings__row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        min-height: 28px;
    }

    .settings__row select,
    .settings__row input[type='text'],
    .settings__row input[type='number'] {
        width: 280px;
        min-width: 0;
        padding: 3px 6px;
        border: 1px solid var(--border);
        border-radius: 3px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        outline: none;
    }

    .settings__row input[type='checkbox'] {
        flex: 0 0 auto;
    }

    .settings__row select:focus,
    .settings__row input:focus,
    .settings__patterns:focus {
        border-color: var(--fg-muted);
    }

    .settings__patterns {
        display: block;
        box-sizing: border-box;
        width: 100%;
        padding: 6px 8px;
        border: 1px solid var(--border);
        border-radius: 3px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        resize: vertical;
        outline: none;
    }

    .settings__hint {
        margin: 0 0 8px;
        color: var(--fg-muted);
        line-height: 1.4;
    }

    .settings__list {
        margin: 0 0 8px;
        padding: 0;
        list-style: none;
    }

    .settings__list-item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
        min-height: 26px;
    }

    .settings__path {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-family: var(--editor-font-family, monospace);
        font-size: var(--ui-font-small);
    }

    .settings__icon-button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        width: 20px;
        height: 20px;
        padding: 0;
        border: none;
        border-radius: 3px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
    }

    .settings__icon-button:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }

    .settings__buttons button {
        padding: 4px 10px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        cursor: pointer;
    }

    .settings__buttons button:hover {
        background: var(--bg-hover);
    }

    .settings__warning {
        margin: 12px 0 0;
        color: var(--fg-danger);
    }
</style>
