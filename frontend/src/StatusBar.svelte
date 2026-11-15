<script lang="ts">
    import { formatCursorPosition } from './status';

    interface Props {
        line: number;
        column: number;
        language: string;
    }

    let { line, column, language }: Props = $props();
</script>

<footer class="statusbar">
    <span class="statusbar__position">{formatCursorPosition(line, column)}</span>
    <span class="statusbar__language">{language}</span>
</footer>

<style>
    .statusbar {
        display: flex;
        justify-content: space-between;
        flex: 0 0 auto;
        height: 24px;
        padding: 0 12px;
        line-height: 24px;
        background: var(--bg-raised);
        border-top: 1px solid var(--border);
        color: var(--fg-muted);
        user-select: none;
    }
</style>
