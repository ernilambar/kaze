<script lang="ts">
    import { defaultExcludePatterns, defaultFileSettings } from '../settings';
    import ResetButton from './ResetButton.svelte';
    import SettingRow from './SettingRow.svelte';

    interface Props {
        filesExclude: string[];
        trimTrailingWhitespace: boolean;
        searchExclude: string[];
        onFilesExclude: (next: string[]) => void;
        onTrimTrailingWhitespace: (trim: boolean) => void;
        onSearchExclude: (next: string[]) => void;
    }

    let {
        filesExclude,
        trimTrailingWhitespace,
        searchExclude,
        onFilesExclude,
        onTrimTrailingWhitespace,
        onSearchExclude,
    }: Props = $props();

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
</script>

{#snippet patternList(
    title: string,
    hint: string,
    current: string[],
    defaults: readonly string[],
    apply: (next: string[]) => void,
)}
    <section class="settings__section">
        <div class="settings__section-head">
            <h3>{title}</h3>
            {#if !samePatterns(current, defaults)}
                <ResetButton label={title} onclick={() => apply([...defaults])} />
            {/if}
        </div>
        <p class="settings__hint">{hint}</p>
        <textarea
            class="settings__patterns"
            rows="6"
            spellcheck="false"
            value={current.join('\n')}
            onchange={(event) => apply(patterns(event.currentTarget.value))}
        ></textarea>
    </section>
{/snippet}

{@render patternList(
    'Explorer excludes',
    'One glob per line. A bare name matches at any depth, a leading slash anchors it to a ' +
        'sidebar root, and a trailing slash matches folders only.',
    filesExclude,
    defaultFileSettings.exclude,
    onFilesExclude,
)}
{@render patternList(
    'Search excludes',
    "Find in Files skips these globs as well as the explorer's list and every .gitignore " +
        'it walks past.',
    searchExclude,
    defaultExcludePatterns,
    onSearchExclude,
)}
<section class="settings__section">
    <h3>Saving</h3>
    <SettingRow
        name="Trim trailing whitespace"
        description="Remove spaces and tabs at line ends when saving."
        modified={trimTrailingWhitespace !== defaultFileSettings.trimTrailingWhitespace}
        onReset={() => onTrimTrailingWhitespace(defaultFileSettings.trimTrailingWhitespace)}
    >
        {#snippet control()}
            <input
                class="settings__switch"
                type="checkbox"
                role="switch"
                checked={trimTrailingWhitespace}
                onchange={(event) => onTrimTrailingWhitespace(event.currentTarget.checked)}
            />
        {/snippet}
    </SettingRow>
</section>

<style>
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
</style>
