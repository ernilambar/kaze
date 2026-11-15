import { Events } from '@wailsio/runtime';
import type { CaseTransform } from './case';

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

export type ZoomCommand = 'in' | 'out' | 'reset';

// ViewToggle is the kova:toggle-view payload: the option and the state the menu
// checkbox now shows.
export interface ViewToggle {
    option: ViewOption;
    checked: boolean;
}

export interface EditorCommandHandlers {
    transformSelection: (transform: CaseTransform) => void;
    trimWhitespace: () => void;
    zoom: (command: ZoomCommand) => void;
    togglePreview: () => void;
    setViewOption: (option: ViewOption, checked: boolean) => void;
}

// listenForEditorCommands runs the handler for each Edit or View menu command
// and returns one function that unsubscribes from all of them.
export function listenForEditorCommands(
    handlers: Partial<EditorCommandHandlers>,
): () => void {
    const offs = [
        Events.On('kova:transform-case', (event) =>
            handlers.transformSelection?.(event.data as CaseTransform),
        ),
        Events.On('kova:trim-whitespace', () => handlers.trimWhitespace?.()),
        Events.On('kova:zoom', (event) => handlers.zoom?.(event.data as ZoomCommand)),
        Events.On('kova:toggle-preview', () => handlers.togglePreview?.()),
        Events.On('kova:toggle-view', (event) => {
            const toggle = event.data as ViewToggle | undefined;
            if (toggle !== undefined) {
                handlers.setViewOption?.(toggle.option, toggle.checked);
            }
        }),
    ];
    return () => {
        for (const off of offs) {
            off();
        }
    };
}
