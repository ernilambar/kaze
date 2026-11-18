<script lang="ts">
    interface Props {
        label: string;
        onclick: () => void;
    }

    let { label, onclick }: Props = $props();
</script>

<button
    type="button"
    class="settings__reset"
    title="Reset to default"
    aria-label={`Reset ${label} to default`}
    {onclick}
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

<style>
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
</style>
