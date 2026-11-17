import { onWindowEvent } from './windowstate';
import type { CaseTransform } from './case';
import type { LineTransform } from './text';

// ViewOption is the editor settings a View menu checkbox flips. The names match
// the fields of EditorSettings, and Go sends them verbatim.
export type ViewOption =
    | 'wordWrap'
    | 'invisibleCharacters'
    | 'lineNumbers'
    | 'indentGuides'
    | 'codeFolding'
    | 'autoCloseBrackets'
    | 'smartIndent';

// VIEW_OPTIONS is what the command palette lists and what the menu checkmarks
// are keyed by, in the order the View menu shows them.
export const VIEW_OPTIONS: readonly { id: ViewOption; label: string }[] = [
    { id: 'wordWrap', label: 'Word Wrap' },
    { id: 'invisibleCharacters', label: 'Invisible Characters' },
    { id: 'lineNumbers', label: 'Line Numbers' },
    { id: 'indentGuides', label: 'Indent Guides' },
    { id: 'codeFolding', label: 'Code Folding' },
    { id: 'autoCloseBrackets', label: 'Auto-Close Brackets' },
    { id: 'smartIndent', label: 'Smart Indent' },
];

// SelectionCommand names one Selection menu command. The app:selection-command
// event carries this id.
export type SelectionCommand =
    | 'add-cursor-above'
    | 'add-cursor-below'
    | 'add-next-occurrence'
    | 'split-into-lines'
    | 'toggle-column-selection'
    | 'move-line-up'
    | 'move-line-down'
    | 'copy-line-up'
    | 'copy-line-down'
    | 'join-lines'
    | 'delete-line'
    | 'toggle-comment';

export interface SelectionCommandSpec {
    id: SelectionCommand;
    label: string;
    // action is Monaco's editor action. Column selection is an editor option
    // instead, so it carries no action.
    action?: string;
}

// SELECTION_COMMANDS is the Selection menu order, shared by the command
// palette and the handler that runs it.
export const SELECTION_COMMANDS: readonly SelectionCommandSpec[] = [
    {
        id: 'add-cursor-above',
        label: 'Add Cursor Above',
        action: 'editor.action.insertCursorAbove',
    },
    {
        id: 'add-cursor-below',
        label: 'Add Cursor Below',
        action: 'editor.action.insertCursorBelow',
    },
    {
        id: 'add-next-occurrence',
        label: 'Add Next Occurrence',
        action: 'editor.action.addSelectionToNextFindMatch',
    },
    {
        id: 'split-into-lines',
        label: 'Split Selection into Lines',
        action: 'editor.action.insertCursorAtEndOfEachLineSelected',
    },
    { id: 'toggle-column-selection', label: 'Toggle Column Selection' },
    { id: 'move-line-up', label: 'Move Line Up', action: 'editor.action.moveLinesUpAction' },
    {
        id: 'move-line-down',
        label: 'Move Line Down',
        action: 'editor.action.moveLinesDownAction',
    },
    { id: 'copy-line-up', label: 'Copy Line Up', action: 'editor.action.copyLinesUpAction' },
    {
        id: 'copy-line-down',
        label: 'Copy Line Down',
        action: 'editor.action.copyLinesDownAction',
    },
    { id: 'join-lines', label: 'Join Lines', action: 'editor.action.joinLines' },
    { id: 'delete-line', label: 'Delete Line', action: 'editor.action.deleteLines' },
    { id: 'toggle-comment', label: 'Toggle Line Comment', action: 'editor.action.commentLine' },
];

// LineCommand names one whole-line transform. The ids are the command palette
// ids and the LineTransform values text.ts understands.
export type LineCommand = LineTransform;

// LINE_COMMANDS is the Edit menu order, shared by the command palette and the
// handler that runs it.
export const LINE_COMMANDS: readonly { id: LineCommand; label: string }[] = [
    { id: 'sort-lines-ascending', label: 'Sort Lines Ascending' },
    { id: 'sort-lines-descending', label: 'Sort Lines Descending' },
    { id: 'delete-duplicate-lines', label: 'Delete Duplicate Lines' },
    { id: 'reverse-lines', label: 'Reverse Lines' },
];

export type ZoomCommand = 'in' | 'out' | 'reset';

// IndentCommand names one indentation action. The ids are palette ids; the
// commands run Monaco's own editor actions, which act on the active buffer
// only.
export type IndentCommand =
    | 'indent-using-spaces'
    | 'indent-using-tabs'
    | 'convert-indentation-to-spaces'
    | 'convert-indentation-to-tabs';

export interface IndentCommandSpec {
    id: IndentCommand;
    label: string;
    // action is Monaco's built-in editor action.
    action: string;
}

// INDENT_COMMANDS is the order the indentation picker and the command palette
// show: the two choosers open Monaco's size picker, the two converters rewrite
// the buffer's leading whitespace.
export const INDENT_COMMANDS: readonly IndentCommandSpec[] = [
    {
        id: 'indent-using-spaces',
        label: 'Indent Using Spaces…',
        action: 'editor.action.indentUsingSpaces',
    },
    {
        id: 'indent-using-tabs',
        label: 'Indent Using Tabs…',
        action: 'editor.action.indentUsingTabs',
    },
    {
        id: 'convert-indentation-to-spaces',
        label: 'Convert Indentation to Spaces',
        action: 'editor.action.indentationToSpaces',
    },
    {
        id: 'convert-indentation-to-tabs',
        label: 'Convert Indentation to Tabs',
        action: 'editor.action.indentationToTabs',
    },
];

// ViewToggle is the app:toggle-view payload: the option and the state the menu
// checkbox now shows.
export interface ViewToggle {
    option: ViewOption;
    checked: boolean;
}

export interface EditorCommandHandlers {
    transformSelection: (transform: CaseTransform) => void;
    trimWhitespace: () => void;
    selectionCommand: (command: SelectionCommand) => void;
    zoom: (command: ZoomCommand) => void;
    togglePreview: () => void;
    setViewOption: (option: ViewOption, checked: boolean) => void;
    find: () => void;
    replace: () => void;
    goToLine: () => void;
    nextFindResult: () => void;
    previousFindResult: () => void;
}

// listenForEditorCommands runs the handler for each Edit, Selection or View
// menu command and returns one function that unsubscribes from all of them.
export function listenForEditorCommands(
    handlers: Partial<EditorCommandHandlers>,
): () => void {
    const offs = [
        onWindowEvent('app:transform-case', (data) =>
            handlers.transformSelection?.(data as CaseTransform),
        ),
        onWindowEvent('app:trim-whitespace', () => handlers.trimWhitespace?.()),
        onWindowEvent('app:selection-command', (data) =>
            handlers.selectionCommand?.(data as SelectionCommand),
        ),
        onWindowEvent('app:zoom', (data) => handlers.zoom?.(data as ZoomCommand)),
        onWindowEvent('app:toggle-preview', () => handlers.togglePreview?.()),
        onWindowEvent('app:toggle-view', (data) => {
            const toggle = data as ViewToggle | undefined;
            if (toggle !== undefined) {
                handlers.setViewOption?.(toggle.option, toggle.checked);
            }
        }),
        onWindowEvent('app:find', () => handlers.find?.()),
        onWindowEvent('app:replace', () => handlers.replace?.()),
        onWindowEvent('app:go-to-line', () => handlers.goToLine?.()),
        onWindowEvent('app:find-next-result', () => handlers.nextFindResult?.()),
        onWindowEvent('app:find-previous-result', () => handlers.previousFindResult?.()),
    ];
    return () => {
        for (const off of offs) {
            off();
        }
    };
}
