import { describe, expect, it, vi } from 'vitest';
import type { Keybinding } from '../bindings/kaze/models';
import {
    chordFromEvent,
    installKeybindings,
    isUsableBinding,
    matchesWhen,
    normalizeChord,
    normalizeSequence,
    withDefaultKeybindings,
    type KeyContext,
} from './keybindings';

const CONTEXT: KeyContext = {
    editorFocus: true,
    sidebarVisible: true,
    hasActiveTab: true,
    hasDirtyTab: false,
};

function press(options: KeyboardEventInit): KeyboardEvent {
    const event = new KeyboardEvent('keydown', {
        bubbles: true,
        cancelable: true,
        ...options,
    });
    window.dispatchEvent(event);
    return event;
}

describe('normalizeChord', () => {
    it('folds aliases and sorts modifiers', () => {
        expect(normalizeChord('CmdOrCtrl+Shift+P')).toBe('cmd+shift+p');
        expect(normalizeChord('cmd+shift+p')).toBe('cmd+shift+p');
        expect(normalizeChord('option+cmd+k')).toBe('alt+cmd+k');
        expect(normalizeChord('esc')).toBe('escape');
    });

    it('rejects chords with two keys', () => {
        expect(normalizeChord('cmd+a+b')).toBe('');
    });

    it('keeps a chord sequence', () => {
        expect(normalizeSequence('cmd+k cmd+t')).toEqual(['cmd+k', 'cmd+t']);
    });
});

describe('chordFromEvent', () => {
    it('ignores a modifier pressed on its own', () => {
        expect(chordFromEvent(new KeyboardEvent('keydown', { metaKey: true, key: 'Meta', code: 'MetaLeft' }))).toBe('');
    });

    it('uses the physical key for punctuation', () => {
        const event = new KeyboardEvent('keydown', {
            metaKey: true,
            shiftKey: true,
            key: '?',
            code: 'Slash',
        });
        expect(chordFromEvent(event)).toBe('cmd+shift+/');
    });

    it('normalizes letters, digits and named keys', () => {
        expect(chordFromEvent(new KeyboardEvent('keydown', { metaKey: true, key: 'B', code: 'KeyB' }))).toBe('cmd+b');
        expect(chordFromEvent(new KeyboardEvent('keydown', { ctrlKey: true, key: '1', code: 'Digit1' }))).toBe('ctrl+1');
        expect(chordFromEvent(new KeyboardEvent('keydown', { key: 'Escape', code: 'Escape' }))).toBe('escape');
    });
});

describe('isUsableBinding', () => {
    it('requires a modifier or a non-printing key', () => {
        expect(isUsableBinding({ key: 'b', command: 'toggle-sidebar' })).toBe(false);
        expect(isUsableBinding({ key: 'cmd+b', command: 'toggle-sidebar' })).toBe(true);
        expect(isUsableBinding({ key: 'f5', command: 'refresh' })).toBe(true);
        expect(isUsableBinding({ key: 'escape', command: 'settings' })).toBe(true);
        expect(isUsableBinding({ key: 'cmd+b', command: '-toggle-sidebar' })).toBe(false);
    });
});

describe('matchesWhen', () => {
    it('evaluates context keys and negation', () => {
        expect(matchesWhen('editorFocus', CONTEXT)).toBe(true);
        expect(matchesWhen('!editorFocus', CONTEXT)).toBe(false);
        expect(matchesWhen('!hasDirtyTab', CONTEXT)).toBe(true);
        expect(matchesWhen('sidebarVisible && editorFocus', CONTEXT)).toBe(true);
        expect(matchesWhen('sidebarVisible && !editorFocus', CONTEXT)).toBe(false);
    });

    it('handles or, parentheses and unknown keys', () => {
        expect(matchesWhen('hasDirtyTab || editorFocus', CONTEXT)).toBe(true);
        expect(matchesWhen('(hasDirtyTab || sidebarVisible) && hasActiveTab', CONTEXT)).toBe(true);
        expect(matchesWhen('unknownKey', CONTEXT)).toBe(false);
        expect(matchesWhen('true', CONTEXT)).toBe(true);
        expect(matchesWhen('', CONTEXT)).toBe(true);
    });
});

describe('installKeybindings', () => {
    it('leaves the first key of a chord whose when is false', () => {
        const run = vi.fn();
        const dispose = installKeybindings(
            [{ key: 'cmd+k cmd+w', command: 'close-all-tabs', when: '!editorFocus' }],
            { run, context: () => CONTEXT },
        );
        const event = press({ metaKey: true, key: 'k', code: 'KeyK' });
        expect(event.defaultPrevented).toBe(false);
        press({ metaKey: true, key: 'w', code: 'KeyW' });
        expect(run).not.toHaveBeenCalled();
        dispose();
    });

    it('runs a matching chord and prevents the default action', () => {
        const run = vi.fn();
        const dispose = installKeybindings(
            [{ key: 'cmd+b', command: 'toggle-sidebar' }],
            { run, context: () => CONTEXT },
        );
        const event = press({ metaKey: true, key: 'b', code: 'KeyB' });
        expect(run).toHaveBeenCalledWith('toggle-sidebar');
        expect(event.defaultPrevented).toBe(true);
        dispose();
    });

    it('waits for the second chord of a sequence', () => {
        const run = vi.fn();
        const dispose = installKeybindings(
            [{ key: 'cmd+k cmd+t', command: 'toggle-preview' }],
            { run, context: () => CONTEXT },
        );

        const first = press({ metaKey: true, key: 'k', code: 'KeyK' });
        expect(run).not.toHaveBeenCalled();
        expect(first.defaultPrevented).toBe(true);

        const second = press({ metaKey: true, key: 't', code: 'KeyT' });
        expect(run).toHaveBeenCalledWith('toggle-preview');
        expect(second.defaultPrevented).toBe(true);
        dispose();
    });

    it('falls through to a later binding when an earlier when is false', () => {
        const run = vi.fn();
        const dispose = installKeybindings(
            [
                { key: 'cmd+b', command: 'toggle-sidebar', when: 'editorFocus' },
                { key: 'cmd+b', command: 'settings', when: '!editorFocus' },
            ],
            { run, context: () => ({ ...CONTEXT, editorFocus: false }) },
        );
        press({ metaKey: true, key: 'b', code: 'KeyB' });
        expect(run).toHaveBeenCalledWith('settings');
        dispose();
    });

    it('does not run when the when expression is false', () => {
        const run = vi.fn();
        const dispose = installKeybindings(
            [{ key: 'cmd+b', command: 'toggle-sidebar', when: 'editorFocus' }],
            { run, context: () => ({ ...CONTEXT, editorFocus: false }) },
        );
        press({ metaKey: true, key: 'b', code: 'KeyB' });
        expect(run).not.toHaveBeenCalled();
        dispose();
    });

    it('ignores a binding that would swallow typing', () => {
        const run = vi.fn();
        const dispose = installKeybindings([{ key: 'b', command: 'new-file' }], {
            run,
            context: () => CONTEXT,
        });
        const event = press({ key: 'b', code: 'KeyB' });
        expect(run).not.toHaveBeenCalled();
        expect(event.defaultPrevented).toBe(false);
        dispose();
    });

    it('stops listening after dispose', () => {
        const run = vi.fn();
        const dispose = installKeybindings([{ key: 'cmd+b', command: 'toggle-sidebar' }], {
            run,
            context: () => CONTEXT,
        });
        dispose();
        press({ metaKey: true, key: 'b', code: 'KeyB' });
        expect(run).not.toHaveBeenCalled();
    });
});

describe('withDefaultKeybindings', () => {
    it('appends the defaults after user entries', () => {
        const user: Keybinding[] = [{ key: 'cmd+k', command: 'save' }];
        const merged = withDefaultKeybindings(user);
        expect(merged[0]).toEqual(user[0]);
        expect(merged.some((binding) => binding.command === 'next-tab')).toBe(true);
    });

    it('drops a default the user removed or whose key the user took', () => {
        const merged = withDefaultKeybindings([
            { key: 'cmd+shift+]', command: '-next-tab' },
            { key: 'cmd+alt+g', command: 'save' },
        ]);
        expect(merged.some((binding) => binding.command === 'next-tab')).toBe(false);
        expect(merged.some((binding) => binding.command === 'find-next-result')).toBe(false);
        expect(merged.some((binding) => binding.command === 'previous-tab')).toBe(true);
    });
});

describe('Keybinding type compatibility', () => {
    it('accepts the generated shape', () => {
        const binding: Keybinding = { key: 'cmd+b', command: 'toggle-sidebar' };
        expect(binding.key).toBe('cmd+b');
    });
});
