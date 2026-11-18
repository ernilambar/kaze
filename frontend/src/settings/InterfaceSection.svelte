<script lang="ts">
    import { defaultUISettings, type UISettings } from '../settings';
    import SettingRow from './SettingRow.svelte';

    interface Props {
        ui: UISettings;
        onUI: (next: UISettings) => void;
    }

    let { ui, onUI }: Props = $props();

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
</script>

{#snippet select(value: string, options: [string, string][], apply: (value: string) => void)}
    <span class="settings__select">
        <select {value} onchange={(event) => apply(event.currentTarget.value)}>
            {#each options as [id, label] (id)}
                <option value={id}>{label}</option>
            {/each}
        </select>
        <svg class="settings__select-arrow" aria-hidden="true" height="10" viewBox="0 0 12 10" width="12">
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

<section class="settings__section">
    <h3>Interface</h3>
    <SettingRow
        name="UI font"
        description="Font for the sidebar, tabs and menus."
        modified={ui.fontFamily !== defaultUISettings.fontFamily}
        onReset={() => setUIFontFamily(defaultUISettings.fontFamily)}
    >
        {#snippet control()}
            <input
                type="text"
                value={ui.fontFamily}
                placeholder="System font"
                spellcheck="false"
                onchange={(event) => setUIFontFamily(event.currentTarget.value)}
            />
        {/snippet}
    </SettingRow>
    <SettingRow
        name="UI size"
        description="Chrome text size in pixels."
        modified={ui.fontSize !== defaultUISettings.fontSize}
        onReset={() => setUIFontSize(String(defaultUISettings.fontSize))}
    >
        {#snippet control()}
            <span class="settings__field">
                <input
                    type="number"
                    min="9"
                    max="48"
                    step="1"
                    value={ui.fontSize}
                    onchange={(event) => setUIFontSize(event.currentTarget.value)}
                />
                <span class="settings__unit">px</span>
            </span>
        {/snippet}
    </SettingRow>
    <SettingRow
        name="Tab icons"
        description="Show file icons in the tab bar."
        modified={ui.tabIcons !== defaultUISettings.tabIcons}
        onReset={() => setTabIcons(defaultUISettings.tabIcons)}
    >
        {#snippet control()}
            <input
                class="settings__switch"
                type="checkbox"
                role="switch"
                checked={ui.tabIcons}
                onchange={(event) => setTabIcons(event.currentTarget.checked)}
            />
        {/snippet}
    </SettingRow>
    <SettingRow
        name="Icon style"
        description="Colorful or monochrome file icons."
        modified={ui.iconStyle !== defaultUISettings.iconStyle}
        onReset={() => setIconStyle(defaultUISettings.iconStyle)}
    >
        {#snippet control()}
            {@render select(
                ui.iconStyle,
                [
                    ['colorful', 'Colorful'],
                    ['monochrome', 'Monochrome'],
                ],
                setIconStyle,
            )}
        {/snippet}
    </SettingRow>
    <SettingRow
        name="Sidebar side"
        description="Which side the sidebar opens on."
        modified={ui.sidebarPosition !== defaultUISettings.sidebarPosition}
        onReset={() => setSidebarPosition(defaultUISettings.sidebarPosition)}
    >
        {#snippet control()}
            {@render select(
                ui.sidebarPosition,
                [
                    ['left', 'Left'],
                    ['right', 'Right'],
                ],
                setSidebarPosition,
            )}
        {/snippet}
    </SettingRow>
</section>
