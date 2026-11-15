<script lang="ts">
    import {
        changePath,
        statusLabel,
        statusMark,
        type GitChange,
    } from './git';

    interface Props {
        changes: GitChange[];
        busy?: boolean;
        onOpenDiff: (path: string) => void;
        onRefresh: () => void;
    }

    let { changes, busy = false, onOpenDiff, onRefresh }: Props = $props();
</script>

<aside class="changes">
    <div class="changes__header">
        <span class="changes__title">Changes</span>
        <button
            class="changes__action"
            disabled={busy}
            onclick={onRefresh}
            title="Refresh Changes"
        >
            <svg aria-hidden="true" height="12" viewBox="0 0 12 12" width="12">
                <path
                    d="M10 3.5A4.5 4.5 0 1 0 10.5 6"
                    fill="none"
                    stroke="currentColor"
                    stroke-linecap="round"
                    stroke-width="1.3"
                />
                <path d="M8 1.5 L10.5 3.5 L8.5 5.5" fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="1.3" />
            </svg>
        </button>
    </div>

    {#if changes.length === 0}
        <p class="changes__empty">No changes.</p>
    {:else}
        <ul class="changes__list">
            {#each changes as change (change.path)}
                {@const path = changePath(change)}
                <li>
                    <button
                        class="changes__row"
                        onclick={() => onOpenDiff(change.path)}
                        title={`${change.path}\n${statusLabel(change.status)}`}
                    >
                        <span
                            class="changes__mark changes__mark--{change.status}"
                            aria-label={statusLabel(change.status)}
                        >{statusMark(change.status)}</span>
                        <span class="changes__path">{path}</span>
                    </button>
                </li>
            {/each}
        </ul>
    {/if}
</aside>

<style>
    .changes {
        display: flex;
        flex-direction: column;
        flex: 0 0 auto;
        width: 260px;
        min-width: 160px;
        max-width: 40vw;
        background: var(--bg-sidebar);
        border-right: 1px solid var(--border);
        user-select: none;
        overflow: hidden;
    }

    .changes__header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        height: 32px;
        padding: 0 6px 0 12px;
    }

    .changes__title {
        color: var(--fg-muted);
        font-size: 11px;
        font-weight: 600;
        letter-spacing: 0.06em;
        text-transform: uppercase;
    }

    .changes__action {
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

    .changes__action:hover:enabled {
        background: var(--bg-hover);
        color: var(--fg);
    }

    .changes__action:disabled {
        cursor: default;
        opacity: 0.5;
    }

    .changes__list {
        flex: 1 1 auto;
        margin: 0;
        padding: 0 0 8px;
        list-style: none;
        overflow: auto;
    }

    .changes__row {
        display: flex;
        align-items: center;
        gap: 6px;
        width: 100%;
        min-width: 0;
        padding: 3px 12px;
        border: none;
        background: none;
        color: var(--fg);
        font: inherit;
        text-align: left;
        cursor: pointer;
    }

    .changes__row:hover {
        background: var(--bg-raised);
    }

    .changes__row:focus-visible {
        outline: 1px solid var(--fg-muted);
        outline-offset: -1px;
    }

    .changes__mark {
        flex: 0 0 auto;
        width: 12px;
        font-family: Menlo, monospace;
        font-size: 12px;
        font-weight: 600;
        text-align: center;
    }

    .changes__mark--modified {
        color: var(--git-modified);
    }

    .changes__mark--added,
    .changes__mark--untracked {
        color: var(--git-added);
    }

    .changes__mark--deleted {
        color: var(--git-deleted);
    }

    .changes__mark--conflicted {
        color: var(--git-conflicted);
    }

    .changes__path {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .changes__empty {
        margin: 0;
        padding: 4px 12px 12px;
        color: var(--fg-muted);
    }
</style>
