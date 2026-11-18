<script lang="ts">
    import SettingRow from './SettingRow.svelte';

    interface Props {
        sidebarVisible: boolean;
        autoReveal: boolean;
        onSidebarVisible: (visible: boolean) => void;
        onAutoReveal: (autoReveal: boolean) => void;
    }

    let { sidebarVisible, autoReveal, onSidebarVisible, onAutoReveal }: Props = $props();

    // SIDEBAR_DEFAULTS mirrors defaultSettings in settings.go. The overlay only
    // needs the two booleans it can reset; the folder list has no default.
    const SIDEBAR_DEFAULTS = { visible: true, autoReveal: true } as const;
</script>

{#snippet toggle(
    name: string,
    description: string,
    checked: boolean,
    fallback: boolean,
    apply: (checked: boolean) => void,
)}
    <SettingRow {name} {description} modified={checked !== fallback} onReset={() => apply(fallback)}>
        {#snippet control()}
            <input
                class="settings__switch"
                type="checkbox"
                role="switch"
                {checked}
                onchange={(event) => apply(event.currentTarget.checked)}
            />
        {/snippet}
    </SettingRow>
{/snippet}

<section class="settings__section">
    <h3>Sidebar</h3>
    {@render toggle(
        'Show the sidebar',
        'Show the folder tree beside the editor.',
        sidebarVisible,
        SIDEBAR_DEFAULTS.visible,
        onSidebarVisible,
    )}
    {@render toggle(
        'Reveal the active file',
        'Expand the tree to the focused tab.',
        autoReveal,
        SIDEBAR_DEFAULTS.autoReveal,
        onAutoReveal,
    )}
</section>
