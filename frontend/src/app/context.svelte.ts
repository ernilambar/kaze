import type EditorPane from '../EditorPane.svelte';
import type { OpenDocument } from '../document';

// WindowContext is the state every window controller shares: the tab list, the
// sidebar roots, the status-bar notice and the editor pane. Controllers read it
// through the instance, never through a destructured copy, so they always see
// the current value.
export class WindowContext {
    tabs = $state<OpenDocument[]>([]);
    activeId = $state('');
    folders = $state<string[]>([]);
    notice = $state('');
    busy = $state(false);
    // windowName is this window's Wails identity. Every per-window backend call
    // and event listener uses it, so a menu command or close prompt lands in
    // the window it belongs to.
    windowName = $state('');
    // searchActive is whether the Find in Files results replace the editor.
    // Opening or focusing any tab clears it, so it lives with the tab state.
    searchActive = $state(false);

    active = $derived(this.tabs.find((tab) => tab.id === this.activeId));
    unsaved = $derived(this.tabs.some((tab) => tab.dirty));

    readonly #editorPane: () => EditorPane | undefined;

    // editorPane is a bind:this ref owned by App.svelte, so the context keeps a
    // getter rather than a copy that would go stale.
    constructor(editorPane: () => EditorPane | undefined) {
        this.#editorPane = editorPane;
    }

    get editorPane(): EditorPane | undefined {
        return this.#editorPane();
    }

    findTab = (id: string): OpenDocument | undefined => {
        return this.tabs.find((tab) => tab.id === id);
    };
}
