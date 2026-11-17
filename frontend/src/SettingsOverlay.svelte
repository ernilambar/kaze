<script lang="ts">
    import { onMount } from 'svelte';
    import type { ThemeInfo } from './theme';
    import {
        defaultEditorSettings,
        defaultExcludeSettings,
        defaultThemeSettings,
        defaultUISettings,
    } from './settings';
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

    const THEME_MODES: readonly { id: ThemeMode; label: string }[] = [
        { id: 'system', label: 'System' },
        { id: 'light', label: 'Light' },
        { id: 'dark', label: 'Dark' },
    ];

    // VIEW_TOGGLES are the editor options the View menu mirrors. They stay in
    // the same order here so the panel and the native menu read alike.
    const VIEW_TOGGLES: readonly {
        key: BooleanEditorKey;
        label: string;
        description: string;
    }[] = [
        { key: 'wordWrap', label: 'Word wrap', description: 'Wrap long lines to the window width.' },
        {
            key: 'invisibleCharacters',
            label: 'Invisible characters',
            description: 'Show spaces, tabs and line endings as glyphs.',
        },
        { key: 'lineNumbers', label: 'Line numbers', description: 'Show the gutter line numbers.' },
        {
            key: 'indentGuides',
            label: 'Indent guides',
            description: 'Draw guides at each indentation level.',
        },
        {
            key: 'codeFolding',
            label: 'Code folding',
            description: 'Fold brackets, comments and regions.',
        },
        {
            key: 'autoCloseBrackets',
            label: 'Auto-close brackets and quotes',
            description: 'Complete brackets and quotes while typing.',
        },
        {
            key: 'smartIndent',
            label: 'Smart indent',
            description: "Match the previous line's indentation.",
        },
    ];

    // SIDEBAR_DEFAULTS mirrors defaultSettings in settings.go. The overlay only
    // needs the two booleans it can reset; the folder list has no default.
    const SIDEBAR_DEFAULTS = { visible: true, autoReveal: true } as const;

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

    function samePatterns(current: readonly string[], defaults: readonly string[]): boolean {
        return (
            current.length === defaults.length &&
            current.every((pattern, index) => pattern === defaults[index])
        );
    }

    function focusableElements(): HTMLElement[] {
        if (dialog === undefined) {
            return [];
        }
        return Array.from(
            dialog.querySelectorAll<HTMLElement>(
                [
                    'button:not([disabled])',
                    'input:not([disabled]):not([type="radio"])',
                    'input[type="radio"]:checked',
                    'select:not([disabled])',
                    'textarea:not([disabled])',
                    '[tabindex]:not([tabindex="-1"])',
                ].join(','),
            ),
        ).filter((element) => element.getClientRects().length > 0);
    }

    // onKeydown closes on Escape and keeps Tab inside the dialog. A modal that
    // lets focus escape behind the backdrop is unusable with a keyboard.
    function onKeydown(event: KeyboardEvent): void {
        if (event.key === 'Escape') {
            event.preventDefault();
            onClose();
            return;
        }
        if (event.key !== 'Tab') {
            return;
        }
        const items = focusableElements();
        if (items.length === 0) {
            return;
        }
        const first = items[0];
        const last = items[items.length - 1];
        const active = document.activeElement;
        if (active === dialog) {
            event.preventDefault();
            (event.shiftKey ? last : first).focus();
            return;
        }
        if (!dialog?.contains(active)) {
            event.preventDefault();
            first.focus();
            return;
        }
        if (event.shiftKey && active === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && active === last) {
            event.preventDefault();
            first.focus();
        }
    }

    onMount(() => dialog?.focus());
</script>

<svelte:window onkeydown={onKeydown} />

{#snippet resetButton(label: string, action: () => void)}
    <button
        type="button"
        class="settings__reset"
        title="Reset to default"
        aria-label={`Reset ${label} to default`}
        onclick={action}
    >
        <svg aria-hidden="true" height="12" viewBox="0 0 12 12" width="12">
            <path
                d="M3.1 4.4a3.4 3.4 0 1 1-.5 2.4"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-width="1.2"
            />
            <path
                d="M1.7 2.3 2.8 4.7 5.2 4.1"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.2"
            />
        </svg>
    </button>
{/snippet}

{#snippet themeGroup(
    title: string,
    appearance: 'light' | 'dark',
    list: ThemeInfo[],
    selected: string,
    fallback: string,
    apply: (name: string) => void,
)}
    <div class="settings__theme-group">
        <div class="settings__theme-head">
            <h4>{title}</h4>
            {#if selected !== fallback}
                {@render resetButton(title, () => apply(fallback))}
            {/if}
        </div>
        {#if list.length === 0}
            <p class="settings__hint">No {appearance} themes are installed.</p>
        {:else}
            <div class="settings__theme-grid">
                {#each list as entry (entry.name)}
                    <button
                        type="button"
                        class="settings__theme-card"
                        class:settings__theme-card--active={entry.name === selected}
                        aria-pressed={entry.name === selected}
                        aria-label={`Use ${entry.name} as the ${title.toLowerCase()}`}
                        onclick={() => apply(entry.name)}
                    >
                        <span
                            class="settings__theme-preview"
                            style:background-color={entry.preview.background || 'var(--bg)'}
                            style:color={entry.preview.foreground || 'var(--fg)'}
                        >
                            <span class="settings__theme-code">
                                <span style:color={entry.preview.comment || 'inherit'}>// Kaze</span>
                            </span>
                            <span class="settings__theme-code">
                                <span style:color={entry.preview.keyword || 'inherit'}>const</span>
                                name
                                <span style:color={entry.preview.foreground || 'inherit'}>=</span>
                                <span style:color={entry.preview.string || 'inherit'}>"kaze"</span>;
                            </span>
                            <span class="settings__theme-code">
                                <span style:color={entry.preview.accent || 'inherit'}>/**/</span>
                            </span>
                        </span>
                        <span class="settings__theme-meta">
                            <span class="settings__theme-name">{entry.name}</span>
                            <span class="settings__theme-family">{entry.family}</span>
                        </span>
                        {#if entry.name === selected}
                            <span class="settings__theme-check" aria-hidden="true">
                                <svg height="10" viewBox="0 0 10 10" width="10">
                                    <path
                                        d="M1.5 5.3 4 7.7 8.5 2.5"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="1.5"
                                    />
                                </svg>
                            </span>
                        {/if}
                    </button>
                {/each}
            </div>
        {/if}
    </div>
{/snippet}

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
            <div class="settings__head-actions">
                <span class="settings__autosave">Changes save automatically</span>
                <button class="settings__close" aria-label="Close Settings" onclick={onClose}>
                    <svg aria-hidden="true" height="9" viewBox="0 0 9 9" width="9">
                        <path
                            d="M1 1 L8 8 M8 1 L1 8"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.2"
                        />
                    </svg>
                </button>
            </div>
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
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Font family</span>
                                    <span class="settings__description">
                                        Monospace font used by the editor.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__font-input"
                                        type="text"
                                        value={editor.fontFamily}
                                        placeholder="Menlo"
                                        spellcheck="false"
                                        onchange={(event) => commitText(event, 'fontFamily')}
                                    />
                                </span>
                            </label>
                            {#if editor.fontFamily !== defaultEditorSettings.fontFamily}
                                {@render resetButton('Font family', () =>
                                    updateEditor('fontFamily', defaultEditorSettings.fontFamily))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Font size</span>
                                    <span class="settings__description">Text size in pixels.</span>
                                </span>
                                <span class="settings__control">
                                    <span class="settings__field">
                                        <input
                                            type="number"
                                            min="6"
                                            max="72"
                                            step="0.5"
                                            value={editor.fontSize}
                                            onchange={(event) =>
                                                commitNumber(event, 'fontSize', 6, 72)}
                                        />
                                        <span class="settings__unit">px</span>
                                    </span>
                                </span>
                            </label>
                            {#if editor.fontSize !== defaultEditorSettings.fontSize}
                                {@render resetButton('Font size', () =>
                                    updateEditor('fontSize', defaultEditorSettings.fontSize))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Font weight</span>
                                    <span class="settings__description">
                                        100–900; the font must include the weight.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        type="number"
                                        min="100"
                                        max="900"
                                        step="10"
                                        value={editor.fontWeight}
                                        onchange={(event) =>
                                            commitNumber(event, 'fontWeight', 100, 900)}
                                    />
                                </span>
                            </label>
                            {#if editor.fontWeight !== defaultEditorSettings.fontWeight}
                                {@render resetButton('Font weight', () =>
                                    updateEditor('fontWeight', defaultEditorSettings.fontWeight))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Ligatures</span>
                                    <span class="settings__description">
                                        Render joined glyphs such as -&gt; and !=.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={editor.ligatures}
                                        onchange={(event) => commitToggle(event, 'ligatures')}
                                    />
                                </span>
                            </label>
                            {#if editor.ligatures !== defaultEditorSettings.ligatures}
                                {@render resetButton('Ligatures', () =>
                                    updateEditor('ligatures', defaultEditorSettings.ligatures))}
                            {/if}
                        </div>
                        <div class="settings__preview">
                            <code
                                style:font-family={editor.fontFamily.trim() === ''
                                    ? undefined
                                    : editor.fontFamily}
                                style:font-size={`${editor.fontSize}px`}
                                style:font-weight={`${editor.fontWeight}`}
                                style:font-variant-ligatures={editor.ligatures
                                    ? 'common-ligatures'
                                    : 'none'}
                            >const message = "Hello, Kaze"; // -> != => ===</code>
                        </div>
                    </section>

                    <section class="settings__section">
                        <h3>Indentation</h3>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Tab size</span>
                                    <span class="settings__description">
                                        Columns per indentation level.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        type="number"
                                        min="1"
                                        max="12"
                                        step="1"
                                        value={editor.tabSize}
                                        onchange={(event) =>
                                            commitNumber(event, 'tabSize', 1, 12, true)}
                                    />
                                </span>
                            </label>
                            {#if editor.tabSize !== defaultEditorSettings.tabSize}
                                {@render resetButton('Tab size', () =>
                                    updateEditor('tabSize', defaultEditorSettings.tabSize))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Use hard tabs</span>
                                    <span class="settings__description">
                                        Insert tab characters instead of spaces.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={editor.hardTabs}
                                        onchange={(event) => commitToggle(event, 'hardTabs')}
                                    />
                                </span>
                            </label>
                            {#if editor.hardTabs !== defaultEditorSettings.hardTabs}
                                {@render resetButton('Use hard tabs', () =>
                                    updateEditor('hardTabs', defaultEditorSettings.hardTabs))}
                            {/if}
                        </div>
                    </section>

                    <section class="settings__section">
                        <h3>Cursor</h3>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Cursor width</span>
                                    <span class="settings__description">
                                        Caret thickness in pixels.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <span class="settings__field">
                                        <input
                                            type="number"
                                            min="1"
                                            max="20"
                                            step="1"
                                            value={editor.cursorWidth}
                                            onchange={(event) =>
                                                commitNumber(event, 'cursorWidth', 1, 20, true)}
                                        />
                                        <span class="settings__unit">px</span>
                                    </span>
                                </span>
                            </label>
                            {#if editor.cursorWidth !== defaultEditorSettings.cursorWidth}
                                {@render resetButton('Cursor width', () =>
                                    updateEditor('cursorWidth', defaultEditorSettings.cursorWidth))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Cursor blinks</span>
                                    <span class="settings__description">
                                        Animate the caret while typing.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={editor.cursorBlink}
                                        onchange={(event) => commitToggle(event, 'cursorBlink')}
                                    />
                                </span>
                            </label>
                            {#if editor.cursorBlink !== defaultEditorSettings.cursorBlink}
                                {@render resetButton('Cursor blinks', () =>
                                    updateEditor('cursorBlink', defaultEditorSettings.cursorBlink))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Surrounding lines</span>
                                    <span class="settings__description">
                                        Lines kept visible above and below the cursor.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <span class="settings__field">
                                        <input
                                            type="number"
                                            min="0"
                                            max="30"
                                            step="1"
                                            value={editor.cursorSurroundingLines}
                                            onchange={(event) =>
                                                commitNumber(
                                                    event,
                                                    'cursorSurroundingLines',
                                                    0,
                                                    30,
                                                    true,
                                                )}
                                        />
                                        <span class="settings__unit">lines</span>
                                    </span>
                                </span>
                            </label>
                            {#if editor.cursorSurroundingLines !==
                                defaultEditorSettings.cursorSurroundingLines}
                                {@render resetButton('Surrounding lines', () =>
                                    updateEditor(
                                        'cursorSurroundingLines',
                                        defaultEditorSettings.cursorSurroundingLines,
                                    ))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Scroll beyond last line</span>
                                    <span class="settings__description">
                                        Allow scrolling past the end of the file.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={editor.scrollBeyondLastLine}
                                        onchange={(event) =>
                                            commitToggle(event, 'scrollBeyondLastLine')}
                                    />
                                </span>
                            </label>
                            {#if editor.scrollBeyondLastLine !==
                                defaultEditorSettings.scrollBeyondLastLine}
                                {@render resetButton('Scroll beyond last line', () =>
                                    updateEditor(
                                        'scrollBeyondLastLine',
                                        defaultEditorSettings.scrollBeyondLastLine,
                                    ))}
                            {/if}
                        </div>
                    </section>

                    <section class="settings__section">
                        <h3>View</h3>
                        {#each VIEW_TOGGLES as toggle (toggle.key)}
                            <div class="settings__row">
                                <label class="settings__row-main">
                                    <span class="settings__text">
                                        <span class="settings__name">{toggle.label}</span>
                                        <span class="settings__description">
                                            {toggle.description}
                                        </span>
                                    </span>
                                    <span class="settings__control">
                                        <input
                                            class="settings__switch"
                                            type="checkbox"
                                            role="switch"
                                            checked={editor[toggle.key]}
                                            onchange={(event) => commitToggle(event, toggle.key)}
                                        />
                                    </span>
                                </label>
                                {#if editor[toggle.key] !== defaultEditorSettings[toggle.key]}
                                    {@render resetButton(toggle.label, () =>
                                        updateEditor(
                                            toggle.key,
                                            defaultEditorSettings[toggle.key],
                                        ))}
                                {/if}
                            </div>
                        {/each}
                    </section>

                    <section class="settings__section">
                        <h3>Saving</h3>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Ensure a final newline</span>
                                    <span class="settings__description">
                                        Append a newline on save when it is missing.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={editor.ensureFinalNewline}
                                        onchange={(event) =>
                                            commitToggle(event, 'ensureFinalNewline')}
                                    />
                                </span>
                            </label>
                            {#if editor.ensureFinalNewline !==
                                defaultEditorSettings.ensureFinalNewline}
                                {@render resetButton('Ensure a final newline', () =>
                                    updateEditor(
                                        'ensureFinalNewline',
                                        defaultEditorSettings.ensureFinalNewline,
                                    ))}
                            {/if}
                        </div>
                    </section>
                {:else if section === 'appearance'}
                    <section class="settings__section">
                        <div class="settings__section-head">
                            <h3>Appearance</h3>
                            {#if theme.mode !== defaultThemeSettings.mode ||
                                theme.light !== defaultThemeSettings.light ||
                                theme.dark !== defaultThemeSettings.dark}
                                {@render resetButton('Appearance', () =>
                                    onTheme({ ...defaultThemeSettings }))}
                            {/if}
                        </div>
                        <p class="settings__hint">
                            The light and dark palettes are used by the matching appearance.
                        </p>
                        <div class="settings__row settings__row--static">
                            <div class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Mode</span>
                                    <span class="settings__description">
                                        System follows the macOS appearance.
                                    </span>
                                </span>
                                <div
                                    class="settings__segmented"
                                    role="group"
                                    aria-label="Appearance mode"
                                >
                                    {#each THEME_MODES as mode (mode.id)}
                                        <label
                                            class="settings__segment"
                                            class:settings__segment--active={theme.mode === mode.id}
                                        >
                                            <input
                                                type="radio"
                                                name="theme-mode"
                                                value={mode.id}
                                                checked={theme.mode === mode.id}
                                                onchange={() => setMode(mode.id)}
                                            />
                                            {mode.label}
                                        </label>
                                    {/each}
                                </div>
                            </div>
                        </div>
                        <div class="settings__themes">
                            {@render themeGroup(
                                'Light theme',
                                'light',
                                lightThemes,
                                theme.light,
                                defaultThemeSettings.light,
                                setLight,
                            )}
                            {@render themeGroup(
                                'Dark theme',
                                'dark',
                                darkThemes,
                                theme.dark,
                                defaultThemeSettings.dark,
                                setDark,
                            )}
                        </div>
                    </section>
                {:else if section === 'interface'}
                    <section class="settings__section">
                        <h3>Interface</h3>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">UI font</span>
                                    <span class="settings__description">
                                        Font for the sidebar, tabs and menus.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        type="text"
                                        value={ui.fontFamily}
                                        placeholder="System font"
                                        spellcheck="false"
                                        onchange={(event) =>
                                            setUIFontFamily(event.currentTarget.value)}
                                    />
                                </span>
                            </label>
                            {#if ui.fontFamily !== defaultUISettings.fontFamily}
                                {@render resetButton('UI font', () =>
                                    setUIFontFamily(defaultUISettings.fontFamily))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">UI size</span>
                                    <span class="settings__description">
                                        Chrome text size in pixels.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <span class="settings__field">
                                        <input
                                            type="number"
                                            min="9"
                                            max="48"
                                            step="1"
                                            value={ui.fontSize}
                                            onchange={(event) =>
                                                setUIFontSize(event.currentTarget.value)}
                                        />
                                        <span class="settings__unit">px</span>
                                    </span>
                                </span>
                            </label>
                            {#if ui.fontSize !== defaultUISettings.fontSize}
                                {@render resetButton('UI size', () =>
                                    setUIFontSize(String(defaultUISettings.fontSize)))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Tab icons</span>
                                    <span class="settings__description">
                                        Show file icons in the tab bar.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={ui.tabIcons}
                                        onchange={(event) => setTabIcons(event.currentTarget.checked)}
                                    />
                                </span>
                            </label>
                            {#if ui.tabIcons !== defaultUISettings.tabIcons}
                                {@render resetButton('Tab icons', () =>
                                    setTabIcons(defaultUISettings.tabIcons))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Icon style</span>
                                    <span class="settings__description">
                                        Colorful or monochrome file icons.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <span class="settings__select">
                                        <select
                                            value={ui.iconStyle}
                                            onchange={(event) =>
                                                setIconStyle(event.currentTarget.value)}
                                        >
                                            <option value="colorful">Colorful</option>
                                            <option value="monochrome">Monochrome</option>
                                        </select>
                                        <svg
                                            class="settings__select-arrow"
                                            aria-hidden="true"
                                            height="10"
                                            viewBox="0 0 12 10"
                                            width="12"
                                        >
                                            <path
                                                d="M2.5 3.5 6 7 9.5 3.5"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-linecap="round"
                                                stroke-linejoin="round"
                                                stroke-width="1.3"
                                            />
                                        </svg>
                                    </span>
                                </span>
                            </label>
                            {#if ui.iconStyle !== defaultUISettings.iconStyle}
                                {@render resetButton('Icon style', () =>
                                    setIconStyle(defaultUISettings.iconStyle))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Sidebar side</span>
                                    <span class="settings__description">
                                        Which side the sidebar opens on.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <span class="settings__select">
                                        <select
                                            value={ui.sidebarPosition}
                                            onchange={(event) =>
                                                setSidebarPosition(event.currentTarget.value)}
                                        >
                                            <option value="left">Left</option>
                                            <option value="right">Right</option>
                                        </select>
                                        <svg
                                            class="settings__select-arrow"
                                            aria-hidden="true"
                                            height="10"
                                            viewBox="0 0 12 10"
                                            width="12"
                                        >
                                            <path
                                                d="M2.5 3.5 6 7 9.5 3.5"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-linecap="round"
                                                stroke-linejoin="round"
                                                stroke-width="1.3"
                                            />
                                        </svg>
                                    </span>
                                </span>
                            </label>
                            {#if ui.sidebarPosition !== defaultUISettings.sidebarPosition}
                                {@render resetButton('Sidebar side', () =>
                                    setSidebarPosition(defaultUISettings.sidebarPosition))}
                            {/if}
                        </div>
                    </section>
                {:else if section === 'files'}
                    <section class="settings__section">
                        <div class="settings__section-head">
                            <h3>Explorer excludes</h3>
                            {#if !samePatterns(filesExclude, defaultExcludeSettings.files)}
                                {@render resetButton('Explorer excludes', () =>
                                    onFilesExclude([...defaultExcludeSettings.files]))}
                            {/if}
                        </div>
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
                        <div class="settings__section-head">
                            <h3>Search excludes</h3>
                            {#if !samePatterns(searchExclude, defaultExcludeSettings.search)}
                                {@render resetButton('Search excludes', () =>
                                    onSearchExclude([...defaultExcludeSettings.search]))}
                            {/if}
                        </div>
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
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Show the sidebar</span>
                                    <span class="settings__description">
                                        Show the folder tree beside the editor.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={sidebarVisible}
                                        onchange={(event) =>
                                            onSidebarVisible(event.currentTarget.checked)}
                                    />
                                </span>
                            </label>
                            {#if sidebarVisible !== SIDEBAR_DEFAULTS.visible}
                                {@render resetButton('Show the sidebar', () =>
                                    onSidebarVisible(SIDEBAR_DEFAULTS.visible))}
                            {/if}
                        </div>
                        <div class="settings__row">
                            <label class="settings__row-main">
                                <span class="settings__text">
                                    <span class="settings__name">Reveal the active file</span>
                                    <span class="settings__description">
                                        Expand the tree to the focused tab.
                                    </span>
                                </span>
                                <span class="settings__control">
                                    <input
                                        class="settings__switch"
                                        type="checkbox"
                                        role="switch"
                                        checked={autoReveal}
                                        onchange={(event) =>
                                            onAutoReveal(event.currentTarget.checked)}
                                    />
                                </span>
                            </label>
                            {#if autoReveal !== SIDEBAR_DEFAULTS.autoReveal}
                                {@render resetButton('Reveal the active file', () =>
                                    onAutoReveal(SIDEBAR_DEFAULTS.autoReveal))}
                            {/if}
                        </div>
                    </section>
                {:else}
                    <section class="settings__section">
                        <h3>Settings files</h3>
                        <p class="settings__hint">
                            These sections write to settings.json as you change them. The raw
                            files remain the source of truth.
                        </p>
                        <div class="settings__actions">
                            <button type="button" class="settings__action" onclick={onOpenSettings}>
                                <span class="settings__action-name">Settings JSON</span>
                                <span class="settings__action-detail">
                                    Values, unknown keys and formatting
                                </span>
                            </button>
                            <button type="button" class="settings__action" onclick={onOpenDefaults}>
                                <span class="settings__action-name">Default Settings</span>
                                <span class="settings__action-detail">
                                    Reference file written by Kaze
                                </span>
                            </button>
                            <button type="button" class="settings__action" onclick={onOpenThemes}>
                                <span class="settings__action-name">Themes Folder</span>
                                <span class="settings__action-detail">
                                    Zed theme files, one family per file
                                </span>
                            </button>
                            <button
                                type="button"
                                class="settings__action"
                                onclick={onOpenKeybindings}
                            >
                                <span class="settings__action-name">Keybindings</span>
                                <span class="settings__action-detail">
                                    VS Code-shaped keys and when clauses
                                </span>
                            </button>
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
        background: rgba(0, 0, 0, 0.4);
        cursor: default;
    }

    .settings__dialog {
        position: relative;
        display: flex;
        flex-direction: column;
        width: min(840px, calc(100vw - 48px));
        height: min(700px, calc(100vh - 64px));
        border: 1px solid var(--border);
        border-radius: 10px;
        background: var(--bg-raised);
        box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
        outline: none;
        overflow: hidden;
    }

    .settings__head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        flex: 0 0 auto;
        padding: 14px 18px;
        border-bottom: 1px solid var(--border);
    }

    .settings__head h2 {
        margin: 0;
        font-size: calc(var(--ui-font-size) + 1px);
        font-weight: 600;
        letter-spacing: -0.01em;
    }

    .settings__head-actions {
        display: flex;
        align-items: center;
        gap: 14px;
    }

    .settings__autosave {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }

    .settings__close {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;
        padding: 0;
        border: none;
        border-radius: 5px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
    }

    .settings__close:hover {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .settings__close:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
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
        flex: 0 0 176px;
        padding: 12px 10px;
        border-right: 1px solid var(--border);
        background: var(--bg-sidebar);
        overflow-y: auto;
    }

    .settings__nav-item {
        width: 100%;
        padding: 7px 10px;
        border: none;
        border-radius: 5px;
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

    .settings__nav-item:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
    }

    .settings__nav-item--active {
        background: var(--bg-selected);
        box-shadow: inset 2px 0 0 var(--link, #4daafc);
        color: var(--fg);
        font-weight: 600;
    }

    .settings__body {
        flex: 1 1 auto;
        min-width: 0;
        padding: 2px 22px 24px;
        overflow-y: auto;
    }

    .settings__section {
        padding: 16px 0 6px;
    }

    .settings__section + .settings__section {
        border-top: 1px solid var(--border);
    }

    .settings__section h3 {
        margin: 0 0 6px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        font-weight: 600;
        letter-spacing: 0.06em;
        text-transform: uppercase;
    }

    .settings__section-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
    }

    .settings__section-head h3 {
        margin: 0;
    }

    .settings__row {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 7px 8px;
        margin: 0 -8px;
        border-radius: 6px;
    }

    .settings__row + .settings__row {
        box-shadow: inset 0 1px 0 var(--border);
    }

    .settings__row:hover {
        background: var(--bg-hover);
    }

    .settings__row-main {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 18px;
        flex: 1 1 auto;
        min-width: 0;
        cursor: pointer;
    }

    .settings__row--static .settings__row-main {
        cursor: default;
    }

    .settings__text {
        display: flex;
        flex-direction: column;
        gap: 2px;
        min-width: 0;
    }

    .settings__name {
        color: var(--fg);
    }

    .settings__description {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        line-height: 1.35;
    }

    .settings__control {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        flex: 0 0 auto;
    }

    .settings__control input[type='text'],
    .settings__control input[type='number'],
    .settings__control select {
        box-sizing: border-box;
        border: 1px solid var(--border);
        border-radius: 5px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        outline: none;
        transition:
            border-color 120ms ease,
            background 120ms ease;
    }

    .settings__control input[type='text'] {
        width: 260px;
        padding: 5px 8px;
    }

    .settings__control input.settings__font-input {
        font-family: var(--editor-font-family, monospace);
    }

    .settings__control input[type='number'] {
        width: 76px;
        padding: 5px 6px 5px 8px;
    }

    .settings__control select {
        width: 190px;
        padding: 5px 28px 5px 8px;
        appearance: none;
        -webkit-appearance: none;
    }

    .settings__control input:hover,
    .settings__control select:hover {
        border-color: var(--fg-muted);
    }

    .settings__control input:focus,
    .settings__control select:focus {
        border-color: var(--link, #4daafc);
    }

    .settings__field {
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .settings__unit {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }

    .settings__switch {
        appearance: none;
        -webkit-appearance: none;
        position: relative;
        flex: 0 0 auto;
        width: 32px;
        height: 18px;
        margin: 0;
        border-radius: 9px;
        background: var(--border);
        cursor: pointer;
        transition: background 120ms ease;
    }

    .settings__switch::after {
        content: '';
        position: absolute;
        top: 2px;
        left: 2px;
        width: 14px;
        height: 14px;
        border-radius: 50%;
        background: #ffffff;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.4);
        transition: transform 120ms ease;
    }

    .settings__switch:checked {
        background: var(--link, #4daafc);
    }

    .settings__switch:checked::after {
        transform: translateX(14px);
    }

    .settings__switch:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 2px;
    }

    .settings__segmented {
        display: inline-flex;
        padding: 2px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg);
    }

    .settings__segment {
        position: relative;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-width: 64px;
        padding: 4px 10px;
        border-radius: 4px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        cursor: pointer;
        transition:
            background 120ms ease,
            color 120ms ease;
    }

    .settings__segment input {
        position: absolute;
        opacity: 0;
        pointer-events: none;
    }

    .settings__segment:hover {
        color: var(--fg);
    }

    .settings__segment--active {
        background: var(--bg-selected);
        color: var(--fg);
        font-weight: 600;
    }

    .settings__segment:focus-within {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
    }

    .settings__select {
        position: relative;
        display: inline-flex;
        align-items: center;
    }

    .settings__select-arrow {
        position: absolute;
        right: 8px;
        top: 50%;
        transform: translateY(-50%);
        color: var(--fg-muted);
        pointer-events: none;
    }

    .settings__reset {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
        width: 22px;
        height: 22px;
        padding: 0;
        border: none;
        border-radius: 4px;
        background: none;
        color: var(--fg-muted);
        cursor: pointer;
        opacity: 0.75;
    }

    .settings__reset:hover {
        background: var(--bg-selected);
        color: var(--fg);
        opacity: 1;
    }

    .settings__reset:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 1px;
    }

    .settings__hint {
        margin: 0 0 10px;
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
        line-height: 1.45;
    }

    .settings__preview {
        margin: 10px 0 2px;
        padding: 14px 16px;
        border: 1px solid var(--border);
        border-radius: 7px;
        background: var(--bg);
        overflow: hidden;
    }

    .settings__preview code {
        display: block;
        color: var(--fg);
        line-height: 1.5;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__themes {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(min(240px, 100%), 1fr));
        gap: 18px;
        align-items: start;
        margin-top: 14px;
    }

    .settings__theme-group {
        min-width: 0;
    }

    .settings__theme-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        margin-bottom: 8px;
    }

    .settings__theme-head h4 {
        margin: 0;
        color: var(--fg);
        font-size: var(--ui-font-size);
        font-weight: 600;
    }

    .settings__theme-grid {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .settings__theme-card {
        position: relative;
        display: flex;
        flex-direction: column;
        min-width: 0;
        padding: 0;
        border: 1px solid var(--border);
        border-radius: 7px;
        background: var(--bg);
        color: var(--fg);
        text-align: left;
        cursor: pointer;
        overflow: hidden;
        transition:
            border-color 120ms ease,
            background 120ms ease;
    }

    .settings__theme-card:hover {
        border-color: var(--fg-muted);
        background: var(--bg-hover);
    }

    .settings__theme-card--active {
        border-color: var(--link, #4daafc);
        box-shadow: 0 0 0 1px var(--link, #4daafc);
    }

    .settings__theme-card:focus-visible {
        outline: 2px solid var(--link, #4daafc);
        outline-offset: 2px;
    }

    .settings__theme-preview {
        display: flex;
        flex-direction: column;
        gap: 1px;
        height: 58px;
        padding: 8px 10px;
        overflow: hidden;
        font-family: var(--editor-font-family, monospace);
        font-size: 9px;
        line-height: 1.5;
    }

    .settings__theme-code {
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__theme-meta {
        display: flex;
        flex-direction: column;
        gap: 1px;
        min-width: 0;
        padding: 7px 10px 9px;
    }

    .settings__theme-name {
        font-size: var(--ui-font-small);
        font-weight: 600;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__theme-family {
        color: var(--fg-muted);
        font-size: var(--ui-font-tiny);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .settings__theme-check {
        position: absolute;
        top: 8px;
        right: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 18px;
        height: 18px;
        border-radius: 50%;
        background: var(--link, #4daafc);
        color: #ffffff;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
    }

    .settings__patterns {
        display: block;
        box-sizing: border-box;
        width: 100%;
        min-height: 120px;
        padding: 8px 10px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg);
        color: var(--fg);
        font-family: var(--editor-font-family, monospace);
        font-size: var(--ui-font-small);
        line-height: 1.5;
        resize: vertical;
        outline: none;
    }

    .settings__patterns:hover {
        border-color: var(--fg-muted);
    }

    .settings__patterns:focus {
        border-color: var(--link, #4daafc);
    }

    .settings__actions {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(min(220px, 100%), 1fr));
        gap: 8px;
    }

    .settings__action {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 2px;
        padding: 9px 11px;
        border: 1px solid var(--border);
        border-radius: 6px;
        background: var(--bg);
        color: var(--fg);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .settings__action:hover {
        background: var(--bg-hover);
        border-color: var(--fg-muted);
    }

    .settings__action-name {
        font-weight: 600;
    }

    .settings__action-detail {
        color: var(--fg-muted);
        font-size: var(--ui-font-small);
    }

    .settings__warning {
        margin: 12px 0 0;
        color: var(--fg-danger);
    }
</style>
