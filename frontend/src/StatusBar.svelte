<script lang="ts">
    import { formatCursorPosition } from './status';

    interface Props {
        line: number;
        column: number;
        language: string;
        folder?: string;
        notice?: string;
    }

    let { line, column, language, folder = '', notice = '' }: Props = $props();
</script>

<footer class="statusbar">
    <span class="statusbar__position">{formatCursorPosition(line, column)}</span>
    {#if notice !== ''}
        <span class="statusbar__notice">{notice}</span>
    {/if}
    {#if folder !== ''}
        <span class="statusbar__folder">{folder}</span>
    {/if}
    <span class="statusbar__language">{language}</span>
</footer>

<style>
    .statusbar {
        display: flex;
        justify-content: space-between;
        gap: 12px;
        flex: 0 0 auto;
        height: 24px;
        padding: 0 12px;
        line-height: 24px;
        background: var(--bg-raised);
        border-top: 1px solid var(--border);
        color: var(--fg-muted);
        user-select: none;
    }

    .statusbar__notice {
        color: var(--fg);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .statusbar__folder {
        margin-left: auto;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
