<script lang="ts">
    import type { SearchFileGroup } from './search';
    import { relativePath } from './sidebar';

    interface Props {
        query: string;
        include: string;
        replacement: string;
        regex: boolean;
        caseSensitive: boolean;
        groups: SearchFileGroup[];
        summary: string;
        error: string;
        busy: boolean;
        // focusToken changes each time the search tab is shown or reactivated,
        // so the query field takes focus without the panel owning a command.
        focusToken?: number;
        // roots are the sidebar folders, used to print result paths relative to
        // the project they belong to.
        roots?: readonly string[];
        onSearch: () => void;
        onReplaceAll: () => void;
        onOpenMatch: (path: string, line: number, column: number) => void;
    }

    let {
        query = $bindable(''),
        include = $bindable(''),
        replacement = $bindable(''),
        regex = $bindable(false),
        caseSensitive = $bindable(false),
        groups,
        summary,
        error,
        busy,
        focusToken = 0,
        roots = [],
        onSearch,
        onReplaceAll,
        onOpenMatch,
    }: Props = $props();

    let queryInput = $state<HTMLInputElement>();
    let lastFocusToken = -1;

    // The panel is mounted once per results view, but showing it again from the
    // menu must put the caret back in the query field, so the token is tracked
    // rather than running only on mount.
    $effect(() => {
        const token = focusToken;
        if (token === lastFocusToken) {
            return;
        }
        lastFocusToken = token;
        queryInput?.focus();
        queryInput?.select();
    });

    function submit(event: SubmitEvent): void {
        event.preventDefault();
        onSearch();
    }
</script>

<div class="search">
    <form class="search__bar" onsubmit={submit}>
        <input
            class="search__input"
            bind:this={queryInput}
            bind:value={query}
            aria-label="Search query"
            placeholder="Search across folders"
            spellcheck="false"
            autocomplete="off"
        />
        <button
            class="search__toggle"
            class:search__toggle--on={regex}
            type="button"
            title="Match a regular expression"
            aria-pressed={regex}
            onclick={() => (regex = !regex)}
        >.*</button>
        <button
            class="search__toggle"
            class:search__toggle--on={caseSensitive}
            type="button"
            title="Match case"
            aria-pressed={caseSensitive}
            onclick={() => (caseSensitive = !caseSensitive)}
        >Aa</button>
        <button class="search__button" type="submit" disabled={busy || query.trim() === ''}>
            {busy ? 'Searching…' : 'Search'}
        </button>
    </form>

    <div class="search__options">
        <input
            class="search__input search__input--small"
            bind:value={include}
            aria-label="Files to include"
            placeholder="files to include, e.g. *.go, src/**"
            spellcheck="false"
            autocomplete="off"
        />
        <input
            class="search__input search__input--small"
            bind:value={replacement}
            aria-label="Replacement text"
            placeholder="replace with"
            spellcheck="false"
            autocomplete="off"
        />
        <button
            class="search__button"
            type="button"
            disabled={busy || query.trim() === ''}
            onclick={onReplaceAll}
        >Replace All</button>
    </div>

    {#if error !== ''}
        <p class="search__error">{error}</p>
    {:else}
        <p class="search__summary">{summary}</p>
    {/if}

    <ul class="search__results">
        {#each groups as group (group.path)}
            <li class="search__file">
                <span class="search__file-name">{group.name}</span>
                <span class="search__file-path" title={group.path}>
                    {relativePath(group.path, roots)}
                </span>
            </li>
            {#each group.matches as match (`${match.path}:${match.line}:${match.column}`)}
                <li>
                    <button
                        class="search__match"
                        type="button"
                        title={`${group.path}:${match.line}`}
                        onclick={() => onOpenMatch(match.path, match.line, match.column)}
                    >
                        <span class="search__line">{match.line}</span>
                        <span class="search__text">{match.text}</span>
                    </button>
                </li>
            {/each}
        {/each}
    </ul>
</div>

<style>
    .search {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        gap: 6px;
        padding: 10px 12px;
        background: var(--bg);
        overflow: hidden;
    }

    .search__bar,
    .search__options {
        display: flex;
        align-items: center;
        gap: 6px;
        flex: 0 0 auto;
    }

    .search__input {
        flex: 1 1 auto;
        min-width: 0;
        height: 26px;
        padding: 0 8px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg-raised);
        color: var(--fg);
        font: inherit;
    }

    .search__input:focus {
        outline: 1px solid var(--fg-muted);
        outline-offset: -1px;
    }

    .search__input--small {
        height: 24px;
        font-size: 12px;
    }

    .search__toggle {
        flex: 0 0 auto;
        min-width: 28px;
        height: 26px;
        padding: 0 6px;
        border: 1px solid transparent;
        border-radius: 4px;
        background: none;
        color: var(--fg-muted);
        font: inherit;
        font-size: 12px;
        cursor: pointer;
    }

    .search__toggle:hover {
        background: var(--bg-hover);
    }

    .search__toggle--on {
        border-color: var(--border);
        background: var(--bg-raised);
        color: var(--fg);
    }

    .search__button {
        flex: 0 0 auto;
        height: 26px;
        padding: 0 12px;
        border: 1px solid var(--border);
        border-radius: 4px;
        background: var(--bg-raised);
        color: var(--fg);
        font: inherit;
        cursor: pointer;
    }

    .search__button:hover:enabled {
        background: var(--bg-hover);
    }

    .search__button:disabled {
        cursor: default;
        opacity: 0.5;
    }

    .search__summary,
    .search__error {
        flex: 0 0 auto;
        margin: 0;
        color: var(--fg-muted);
        font-size: 12px;
    }

    .search__error {
        color: var(--fg-danger);
    }

    .search__results {
        flex: 1 1 auto;
        margin: 0;
        padding: 0 0 8px;
        list-style: none;
        overflow: auto;
    }

    .search__file {
        display: flex;
        align-items: baseline;
        gap: 8px;
        margin-top: 8px;
        padding: 2px 4px;
        font-size: 12px;
    }

    .search__file-name {
        color: var(--fg);
        font-weight: 600;
    }

    .search__file-path {
        overflow: hidden;
        color: var(--fg-muted);
        text-overflow: ellipsis;
        white-space: nowrap;
        direction: rtl;
    }

    .search__match {
        display: flex;
        align-items: baseline;
        gap: 8px;
        width: 100%;
        padding: 2px 4px 2px 4px;
        border: none;
        background: none;
        color: var(--fg);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .search__match:hover {
        background: var(--bg-hover);
    }

    .search__line {
        flex: 0 0 44px;
        color: var(--fg-muted);
        font-size: 12px;
        text-align: right;
    }

    .search__text {
        overflow: hidden;
        font-family: monospace;
        font-size: 12px;
        text-overflow: ellipsis;
        white-space: pre;
    }
</style>
