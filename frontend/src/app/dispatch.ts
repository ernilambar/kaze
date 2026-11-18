import { CASE_TRANSFORMS, type CaseTransform } from '../case';
import {
    INDENT_COMMANDS,
    LINE_COMMANDS,
    SELECTION_COMMANDS,
    VIEW_OPTIONS,
    type LineCommand,
    type SelectionCommand,
    type ViewOption,
    type ZoomCommand,
} from '../editorcommands';
import type { SidebarView } from '../git';
import type { ThemeMode } from '../settings';
import type { CopyPathKind } from '../workspacecommands';

// CommandHandlers is everything the command palette, the menus and the native
// keybindings can ask the window to do. Each entry is defined by App.svelte; the
// dispatch table only decides which one an id runs, so the same id means the
// same action wherever it came from.
export interface CommandHandlers {
    newFile: () => void;
    newFolder: () => void;
    newBuffer: () => void;
    openFile: () => void;
    save: () => void;
    saveAs: () => void;
    saveAll: () => void;
    print: () => void;
    trimWhitespace: () => void;
    transformSelection: (transform: CaseTransform) => void;
    transformLines: (command: LineCommand) => void;
    runSelectionCommand: (command: SelectionCommand) => void;
    runIndentAction: (id: string) => void;
    runEditorAction: (action: string) => void;
    toggleViewOption: (option: ViewOption) => void;
    showLanguagePalette: () => void;
    showQuickOpen: () => void;
    showCommandPalette: () => void;
    showRecentFiles: () => void;
    clearRecent: () => void;
    showFindInFiles: () => void;
    showSettings: () => void;
    showAbout: () => void;
    stepFindResult: (direction: 1 | -1) => void;
    togglePreview: () => void;
    switchSidebarView: (view: SidebarView) => void;
    zoom: (command: ZoomCommand) => void;
    addFolder: () => void;
    closeActiveTab: () => void;
    goToTab: (position: number) => void;
    stepTab: (direction: 1 | -1) => void;
    reopenClosedTab: () => void;
    closeAllTabs: () => void;
    closeAllFolders: () => void;
    moveTabToNewWindow: () => void;
    openSettingsJson: () => void;
    openDefaultSettings: () => void;
    openThemesFolder: () => void;
    openKeybindings: () => void;
    installCommandLineTool: () => void;
    setAppearanceMode: (mode: ThemeMode) => void;
    openWorkspace: () => void;
    saveWorkspaceAs: () => void;
    closeWorkspace: () => void;
    refreshWorkspace: () => void;
    toggleSidebar: () => void;
    revealActiveFile: () => void;
    copyActivePath: (kind: CopyPathKind) => void;
    minimiseWindow: () => void;
    zoomWindow: () => void;
    toggleFullScreen: () => void;
    newWindow: () => void;
    openFolderInNewWindow: () => void;
    closeWindow: () => void;
    bringAllToFront: () => void;
}

// CommandAction runs one command id against the window's handlers.
type CommandAction = (handlers: CommandHandlers) => void;

// COMMAND_TABLE is every palette id that names exactly one command. The case,
// view, selection, line and indentation families are generated from their
// shared tables, so they are resolved by COMMAND_PREFIXES instead of being
// listed here a second time.
const COMMAND_TABLE: Record<string, CommandAction> = {
    'new-file': (handlers) => handlers.newFile(),
    'new-folder': (handlers) => handlers.newFolder(),
    'new-buffer': (handlers) => handlers.newBuffer(),
    'open-file': (handlers) => handlers.openFile(),
    save: (handlers) => handlers.save(),
    'save-as': (handlers) => handlers.saveAs(),
    'save-all': (handlers) => handlers.saveAll(),
    print: (handlers) => handlers.print(),
    'trim-whitespace': (handlers) => handlers.trimWhitespace(),
    'change-language-mode': (handlers) => handlers.showLanguagePalette(),
    find: (handlers) => handlers.runEditorAction('actions.find'),
    replace: (handlers) => handlers.runEditorAction('editor.action.startFindReplaceAction'),
    'go-to-line': (handlers) => handlers.runEditorAction('editor.action.gotoLine'),
    'go-to-symbol': (handlers) => handlers.runEditorAction('editor.action.quickOutline'),
    'find-next-result': (handlers) => handlers.stepFindResult(1),
    'find-previous-result': (handlers) => handlers.stepFindResult(-1),
    'toggle-preview': (handlers) => handlers.togglePreview(),
    'show-explorer': (handlers) => handlers.switchSidebarView('explorer'),
    'show-changes': (handlers) => handlers.switchSidebarView('changes'),
    'zoom-in': (handlers) => handlers.zoom('in'),
    'zoom-out': (handlers) => handlers.zoom('out'),
    'zoom-reset': (handlers) => handlers.zoom('reset'),
    'quick-open': (handlers) => handlers.showQuickOpen(),
    'add-folder': (handlers) => handlers.addFolder(),
    'close-tab': (handlers) => handlers.closeActiveTab(),
    'command-palette': (handlers) => handlers.showCommandPalette(),
    'open-recent': (handlers) => handlers.showRecentFiles(),
    'clear-recent': (handlers) => handlers.clearRecent(),
    'find-in-files': (handlers) => handlers.showFindInFiles(),
    'close-all-tabs': (handlers) => handlers.closeAllTabs(),
    'close-all-folders': (handlers) => handlers.closeAllFolders(),
    'reopen-closed-tab': (handlers) => handlers.reopenClosedTab(),
    'next-tab': (handlers) => handlers.stepTab(1),
    'previous-tab': (handlers) => handlers.stepTab(-1),
    'move-tab-new-window': (handlers) => handlers.moveTabToNewWindow(),
    refresh: (handlers) => handlers.refreshWorkspace(),
    'toggle-sidebar': (handlers) => handlers.toggleSidebar(),
    'reveal-file': (handlers) => handlers.revealActiveFile(),
    'copy-file-path': (handlers) => handlers.copyActivePath('file'),
    'copy-relative-path': (handlers) => handlers.copyActivePath('relative'),
    'copy-file-name': (handlers) => handlers.copyActivePath('name'),
    'open-workspace': (handlers) => handlers.openWorkspace(),
    'save-workspace-as': (handlers) => handlers.saveWorkspaceAs(),
    'close-workspace': (handlers) => handlers.closeWorkspace(),
    settings: (handlers) => handlers.showSettings(),
    about: (handlers) => handlers.showAbout(),
    'open-settings-json': (handlers) => handlers.openSettingsJson(),
    'open-default-settings': (handlers) => handlers.openDefaultSettings(),
    'open-themes-folder': (handlers) => handlers.openThemesFolder(),
    'open-keybindings': (handlers) => handlers.openKeybindings(),
    'install-command-line-tool': (handlers) => handlers.installCommandLineTool(),
    'minimize-window': (handlers) => handlers.minimiseWindow(),
    'zoom-window': (handlers) => handlers.zoomWindow(),
    'toggle-full-screen': (handlers) => handlers.toggleFullScreen(),
    'new-window': (handlers) => handlers.newWindow(),
    'open-folder-new-window': (handlers) => handlers.openFolderInNewWindow(),
    'close-window': (handlers) => handlers.closeWindow(),
    'bring-all-to-front': (handlers) => handlers.bringAllToFront(),
};

// Go to Tab ids are numbered by position, and the Appearance ids name the mode
// the Appearance menu chose.
const GO_TO_TAB_PREFIX = 'go-to-tab-';
const APPEARANCE_PREFIX = 'appearance-';
const APPEARANCE_MODES: readonly ThemeMode[] = ['system', 'light', 'dark'];

// findCommand returns the action for one command id, or undefined when the id
// names nothing. The generated families are tried first because their ids come
// from tables that must stay the single source of truth.
export function findCommand(id: string): CommandAction | undefined {
    const transform = CASE_TRANSFORMS.find((entry) => `case-${entry.id}` === id);
    if (transform !== undefined) {
        return (handlers) => handlers.transformSelection(transform.id);
    }
    const viewOption = VIEW_OPTIONS.find((entry) => `toggle-${entry.id}` === id);
    if (viewOption !== undefined) {
        return (handlers) => handlers.toggleViewOption(viewOption.id);
    }
    const selection = SELECTION_COMMANDS.find((entry) => entry.id === id);
    if (selection !== undefined) {
        return (handlers) => handlers.runSelectionCommand(selection.id);
    }
    const line = LINE_COMMANDS.find((entry) => entry.id === id);
    if (line !== undefined) {
        return (handlers) => handlers.transformLines(line.id);
    }
    const indent = INDENT_COMMANDS.find((entry) => entry.id === id);
    if (indent !== undefined) {
        return (handlers) => handlers.runIndentAction(indent.id);
    }
    if (id.startsWith(GO_TO_TAB_PREFIX)) {
        const position = Number(id.slice(GO_TO_TAB_PREFIX.length));
        if (Number.isInteger(position) && position > 0) {
            return (handlers) => handlers.goToTab(position);
        }
        return undefined;
    }
    if (id.startsWith(APPEARANCE_PREFIX)) {
        const mode = APPEARANCE_MODES.find((entry) => entry === id.slice(APPEARANCE_PREFIX.length));
        if (mode !== undefined) {
            return (handlers) => handlers.setAppearanceMode(mode);
        }
        return undefined;
    }
    return COMMAND_TABLE[id];
}

// dispatchCommand runs the command id names. An unknown id is ignored: a stale
// keybinding must not break the window.
export function dispatchCommand(id: string, handlers: CommandHandlers): void {
    findCommand(id)?.(handlers);
}
