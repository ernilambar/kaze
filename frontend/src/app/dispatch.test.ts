import { describe, expect, it, vi } from 'vitest';
import { COMMANDS } from '../commands';
import { dispatchCommand, findCommand, type CommandHandlers } from './dispatch';
import { CASE_TRANSFORMS } from '../case';
import {
    INDENT_COMMANDS,
    LINE_COMMANDS,
    SELECTION_COMMANDS,
    VIEW_OPTIONS,
} from '../editorcommands';

// recordingHandlers builds every handler from one spy, so a test can assert
// that an id ran something without enumerating the whole interface.
function recordingHandlers(): { handlers: CommandHandlers; ran: ReturnType<typeof vi.fn> } {
    const ran = vi.fn();
    const handlers = new Proxy({} as CommandHandlers, {
        get: () => ran,
    });
    return { handlers, ran };
}

describe('every palette command resolves', () => {
    it('maps each COMMANDS id to a handler', () => {
        const missing = COMMANDS.filter((command) => findCommand(command.id) === undefined);
        expect(missing.map((command) => command.id)).toEqual([]);
    });

    it('has no handler the palette would never offer', () => {
        const palette = new Set(COMMANDS.map((command) => command.id));
        const generated = [
            ...CASE_TRANSFORMS.map((entry) => `case-${entry.id}`),
            ...VIEW_OPTIONS.map((entry) => `toggle-${entry.id}`),
            ...SELECTION_COMMANDS.map((entry) => entry.id),
            ...LINE_COMMANDS.map((entry) => entry.id),
            ...INDENT_COMMANDS.map((entry) => entry.id),
            ...Array.from({ length: 9 }, (_, index) => `go-to-tab-${index + 1}`),
        ];
        const unknown = generated.filter((id) => !palette.has(id));
        expect(unknown).toEqual([]);
        expect(generated.every((id) => findCommand(id) !== undefined)).toBe(true);
    });
});

describe('dispatchCommand', () => {
    it('runs the case family with the transform the id names', () => {
        const transformSelection = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('case-camel', { ...handlers, transformSelection });
        expect(transformSelection).toHaveBeenCalledWith('camel');
    });

    it('runs the view family with the option the id names', () => {
        const toggleViewOption = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('toggle-wordWrap', { ...handlers, toggleViewOption });
        expect(toggleViewOption).toHaveBeenCalledWith('wordWrap');
    });

    it('runs a line command through the editor', () => {
        const transformLines = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('reverse-lines', { ...handlers, transformLines });
        expect(transformLines).toHaveBeenCalledWith('reverse-lines');
    });

    it('runs an indentation action', () => {
        const runIndentAction = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('convert-indentation-to-tabs', { ...handlers, runIndentAction });
        expect(runIndentAction).toHaveBeenCalledWith('convert-indentation-to-tabs');
    });

    it('numbers Go to Tab from the id', () => {
        const goToTab = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('go-to-tab-9', { ...handlers, goToTab });
        expect(goToTab).toHaveBeenCalledWith(9);
    });

    it('runs a plain command from the table', () => {
        const saveAll = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('save-all', { ...handlers, saveAll });
        expect(saveAll).toHaveBeenCalledTimes(1);
    });

    it('maps the Appearance ids onto their modes', () => {
        const setAppearanceMode = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('appearance-system', { ...handlers, setAppearanceMode });
        expect(setAppearanceMode).toHaveBeenCalledWith('system');
    });

    it('sends a Go to Line through Monaco', () => {
        const runEditorAction = vi.fn();
        const { handlers } = recordingHandlers();
        dispatchCommand('go-to-line', { ...handlers, runEditorAction });
        expect(runEditorAction).toHaveBeenCalledWith('editor.action.gotoLine');
    });

    it('ignores an unknown id instead of throwing', () => {
        const { handlers, ran } = recordingHandlers();
        expect(() => dispatchCommand('no-such-command', handlers)).not.toThrow();
        expect(ran).not.toHaveBeenCalled();
    });

    it('rejects a Go to Tab id with no number', () => {
        expect(findCommand('go-to-tab-')).toBeUndefined();
        expect(findCommand('go-to-tab-nope')).toBeUndefined();
    });

    it('rejects an Appearance id with an unknown mode', () => {
        expect(findCommand('appearance-dim')).toBeUndefined();
    });
});
