<script lang="ts">
    import {
        defaultEditorSettings,
        type EditorSettings,
        type ScrollBeyondLastLine,
    } from '../settings';
    import SettingRow from './SettingRow.svelte';

    type NumericEditorKey =
        | 'fontSize'
        | 'fontWeight'
        | 'tabSize'
        | 'cursorWidth'
        | 'cursorSurroundingLines';
    type BooleanEditorKey = Exclude<
        keyof EditorSettings,
        'fontFamily' | 'scrollBeyondLastLine' | NumericEditorKey | 'rulers'
    >;

    interface Props {
        editor: EditorSettings;
        onEditor: (next: EditorSettings) => void;
    }

    let { editor, onEditor }: Props = $props();

    // CSS defines nine weight names. settings.json accepts any positive weight,
    // so the current value is added when it falls between them.
    const FONT_WEIGHTS: readonly { value: number; label: string }[] = [
        { value: 100, label: '100 Thin' },
        { value: 200, label: '200 Extra Light' },
        { value: 300, label: '300 Light' },
        { value: 400, label: '400 Regular' },
        { value: 500, label: '500 Medium' },
        { value: 600, label: '600 Semi Bold' },
        { value: 700, label: '700 Bold' },
        { value: 800, label: '800 Extra Bold' },
        { value: 900, label: '900 Black' },
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

    const SCROLL_BEYOND_MODES: readonly { id: ScrollBeyondLastLine; label: string }[] = [
        { id: 'off', label: 'Off' },
        { id: 'half_page', label: 'Half page' },
        { id: 'one_page', label: 'One page' },
    ];

    let customFontWeight = $derived(
        FONT_WEIGHTS.some((entry) => entry.value === editor.fontWeight)
            ? null
            : editor.fontWeight,
    );

    function updateEditor<K extends keyof EditorSettings>(
        key: K,
        value: EditorSettings[K],
    ): void {
        onEditor({ ...editor, [key]: value });
    }

    function resetEditor(key: keyof EditorSettings): void {
        updateEditor(key, defaultEditorSettings[key]);
    }

    function modified(key: Exclude<keyof EditorSettings, 'rulers'>): boolean {
        return editor[key] !== defaultEditorSettings[key];
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

    function setFontWeight(value: string): void {
        const parsed = Number(value);
        if (Number.isFinite(parsed)) {
            updateEditor('fontWeight', parsed);
        }
    }

    // commitRulers parses a comma or space separated column list, drops
    // unusable values, sorts it and writes the result back into the field.
    function commitRulers(event: Event): void {
        const input = event.currentTarget as HTMLInputElement;
        const rulers = [
            ...new Set(
                input.value
                    .split(/[,\s]+/)
                    .map((value) => Number(value))
                    .filter((value) => Number.isInteger(value) && value > 0),
            ),
        ].sort((a, b) => a - b);
        input.value = rulers.join(', ');
        if (rulers.join(',') !== editor.rulers.join(',')) {
            updateEditor('rulers', rulers);
        }
    }
</script>

{#snippet toggle(key: BooleanEditorKey, name: string, description: string)}
    <SettingRow {name} {description} modified={modified(key)} onReset={() => resetEditor(key)}>
        {#snippet control()}
            <input
                class="settings__switch"
                type="checkbox"
                role="switch"
                checked={editor[key]}
                onchange={(event) => updateEditor(key, event.currentTarget.checked)}
            />
        {/snippet}
    </SettingRow>
{/snippet}

{#snippet number(key: NumericEditorKey, min: number, max: number, step: number)}
    <input
        type="number"
        {min}
        {max}
        {step}
        value={editor[key]}
        onchange={(event) => commitNumber(event, key, min, max, step === 1)}
    />
{/snippet}

{#snippet field(key: NumericEditorKey, min: number, max: number, step: number, unit: string)}
    <span class="settings__field">
        {@render number(key, min, max, step)}
        <span class="settings__unit">{unit}</span>
    </span>
{/snippet}

<section class="settings__section">
    <h3>Typography</h3>
    <SettingRow
        name="Font family"
        description="Monospace font used by the editor."
        modified={modified('fontFamily')}
        onReset={() => resetEditor('fontFamily')}
    >
        {#snippet control()}
            <input
                class="settings__font-input"
                type="text"
                value={editor.fontFamily}
                placeholder="Menlo"
                spellcheck="false"
                onchange={(event) => updateEditor('fontFamily', event.currentTarget.value)}
            />
        {/snippet}
    </SettingRow>
    <SettingRow
        name="Font size"
        description="Text size in pixels."
        modified={modified('fontSize')}
        onReset={() => resetEditor('fontSize')}
    >
        {#snippet control()}
            {@render field('fontSize', 6, 72, 0.5, 'px')}
        {/snippet}
    </SettingRow>
    <SettingRow
        name="Font weight"
        description="The font must include the chosen weight."
        modified={modified('fontWeight')}
        onReset={() => resetEditor('fontWeight')}
    >
        {#snippet control()}
            <span class="settings__select">
                <select
                    value={editor.fontWeight}
                    onchange={(event) => setFontWeight(event.currentTarget.value)}
                >
                    {#if customFontWeight !== null}
                        <option value={customFontWeight}>{customFontWeight}</option>
                    {/if}
                    {#each FONT_WEIGHTS as weight (weight.value)}
                        <option value={weight.value}>{weight.label}</option>
                    {/each}
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
        {/snippet}
    </SettingRow>
    {@render toggle('ligatures', 'Ligatures', 'Render joined glyphs such as -> and !=.')}
    <div class="settings__preview">
        <code
            style:font-family={editor.fontFamily.trim() === '' ? undefined : editor.fontFamily}
            style:font-size={`${editor.fontSize}px`}
            style:font-weight={`${editor.fontWeight}`}
            style:font-variant-ligatures={editor.ligatures ? 'common-ligatures' : 'none'}
        >const message = "Hello, Kaze"; // -> != => ===</code>
    </div>
</section>

<section class="settings__section">
    <h3>Indentation</h3>
    <SettingRow
        name="Tab size"
        description="Columns per indentation level."
        modified={modified('tabSize')}
        onReset={() => resetEditor('tabSize')}
    >
        {#snippet control()}
            {@render number('tabSize', 1, 12, 1)}
        {/snippet}
    </SettingRow>
    {@render toggle('hardTabs', 'Use hard tabs', 'Insert tab characters instead of spaces.')}
</section>

<section class="settings__section">
    <h3>Cursor</h3>
    <SettingRow
        name="Cursor width"
        description="Caret thickness in pixels."
        modified={modified('cursorWidth')}
        onReset={() => resetEditor('cursorWidth')}
    >
        {#snippet control()}
            {@render field('cursorWidth', 1, 20, 1, 'px')}
        {/snippet}
    </SettingRow>
    {@render toggle('cursorBlink', 'Cursor blinks', 'Animate the caret while typing.')}
    <SettingRow
        name="Surrounding lines"
        description="Lines kept visible above and below the cursor."
        modified={modified('cursorSurroundingLines')}
        onReset={() => resetEditor('cursorSurroundingLines')}
    >
        {#snippet control()}
            {@render field('cursorSurroundingLines', 0, 30, 1, 'lines')}
        {/snippet}
    </SettingRow>
    <SettingRow
        name="Scroll beyond last line"
        description="Space below the last line: none, half a page, or a full page."
        modified={modified('scrollBeyondLastLine')}
        onReset={() => resetEditor('scrollBeyondLastLine')}
    >
        {#snippet control()}
            <span class="settings__select">
                <select
                    value={editor.scrollBeyondLastLine}
                    onchange={(event) =>
                        updateEditor(
                            'scrollBeyondLastLine',
                            event.currentTarget.value as ScrollBeyondLastLine,
                        )}
                >
                    {#each SCROLL_BEYOND_MODES as mode (mode.id)}
                        <option value={mode.id}>{mode.label}</option>
                    {/each}
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
        {/snippet}
    </SettingRow>
</section>

<section class="settings__section">
    <h3>View</h3>
    {#each VIEW_TOGGLES as entry (entry.key)}
        {@render toggle(entry.key, entry.label, entry.description)}
    {/each}
    {@render toggle('minimap', 'Minimap', 'Show the file overview beside the scrollbar.')}
    {@render toggle('bracketPairColors', 'Bracket pair colors', 'Color and highlight matching brackets.')}
    <SettingRow
        name="Rulers"
        description="Comma-separated columns for vertical guides."
        modified={editor.rulers.length > 0}
        onReset={() => updateEditor('rulers', [])}
    >
        {#snippet control()}
            <input
                type="text"
                value={editor.rulers.join(', ')}
                placeholder="80, 120"
                spellcheck="false"
                onchange={commitRulers}
            />
        {/snippet}
    </SettingRow>
</section>

<section class="settings__section">
    <h3>Saving</h3>
    {@render toggle(
        'ensureFinalNewline',
        'Ensure a final newline',
        'Append a newline on save when it is missing.',
    )}
</section>

<style>
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
</style>
