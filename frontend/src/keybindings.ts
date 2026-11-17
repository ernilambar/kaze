import type { Keybinding } from '../bindings/kaze/models';

// KeyContext is the small set of states a `when` expression can test. Unknown
// identifiers evaluate to false, matching VS Code's context keys.
export interface KeyContext {
    editorFocus: boolean;
    sidebarVisible: boolean;
    hasActiveTab: boolean;
    hasDirtyTab: boolean;
}

export interface KeybindingBinding {
    sequence: string[];
    command: string;
    when: string;
}

// normalizeChord mirrors normalizeChord in keybindings.go, so a menu
// accelerator and the JSON spelling compare equal.
export function normalizeChord(chord: string): string {
    const modifiers: string[] = [];
    let key = '';
    for (const raw of chord.split('+')) {
        const part = raw.trim().toLowerCase();
        if (part === '') {
            continue;
        }
        switch (part) {
            case 'cmd':
            case 'command':
            case 'cmdorctrl':
                modifiers.push('cmd');
                break;
            case 'ctrl':
            case 'control':
                modifiers.push('ctrl');
                break;
            case 'alt':
            case 'option':
            case 'optionoralt':
                modifiers.push('alt');
                break;
            case 'shift':
                modifiers.push('shift');
                break;
            case 'super':
            case 'win':
            case 'meta':
                modifiers.push('meta');
                break;
            default:
                if (key !== '') {
                    return '';
                }
                key = normalizeKeyName(part);
        }
    }
    if (key === '') {
        return '';
    }
    modifiers.sort();
    return [...modifiers, key].join('+');
}

export function normalizeSequence(key: string): string[] {
    return key
        .trim()
        .split(/\s+/)
        .filter((chord) => chord !== '')
        .map(normalizeChord);
}

function normalizeKeyName(key: string): string {
    switch (key) {
        case 'esc':
            return 'escape';
        case 'return':
            return 'enter';
        case 'arrowup':
            return 'up';
        case 'arrowdown':
            return 'down';
        case 'arrowleft':
            return 'left';
        case 'arrowright':
            return 'right';
        case 'del':
            return 'delete';
        case 'spacebar':
            return 'space';
        default:
            return key;
    }
}

const CODE_KEYS: Readonly<Record<string, string>> = {
    Backquote: '`',
    Minus: '-',
    Equal: '=',
    BracketLeft: '[',
    BracketRight: ']',
    Backslash: '\\',
    Semicolon: ';',
    Quote: "'",
    Comma: ',',
    Period: '.',
    Slash: '/',
};

const NAMED_KEYS: Readonly<Record<string, string>> = {
    ' ': 'space',
    space: 'space',
    escape: 'escape',
    esc: 'escape',
    enter: 'enter',
    tab: 'tab',
    backspace: 'backspace',
    delete: 'delete',
    arrowup: 'up',
    arrowdown: 'down',
    arrowleft: 'left',
    arrowright: 'right',
    home: 'home',
    end: 'end',
    pageup: 'pageup',
    pagedown: 'pagedown',
};

// keyFromEvent reads the physical key so combinations such as cmd+shift+/
// match the slash key even though the produced character is "?".
export function keyFromEvent(event: KeyboardEvent): string {
    const code = event.code;
    if (/^Key[A-Z]$/.test(code)) {
        return code.slice(3).toLowerCase();
    }
    if (/^Digit[0-9]$/.test(code)) {
        return code.slice(5);
    }
    if (/^F([1-9]|1[0-9]|2[0-4])$/.test(code)) {
        return code.toLowerCase();
    }
    const named = NAMED_KEYS[event.key.toLowerCase()];
    if (named !== undefined) {
        return named;
    }
    const punctuation = CODE_KEYS[code];
    if (punctuation !== undefined) {
        return punctuation;
    }
    return event.key.toLowerCase();
}

// chordFromEvent renders a key press the way keybindings.json writes it.
const MODIFIER_KEYS = new Set(['Meta', 'Control', 'Alt', 'Shift', 'CapsLock', 'Fn']);

export function chordFromEvent(event: KeyboardEvent): string {
    // A modifier pressed on its own is not a key yet; treating it as one would
    // reset a pending chord between Cmd+K and its second key.
    if (MODIFIER_KEYS.has(event.key)) {
        return '';
    }
    const modifiers: string[] = [];
    if (event.metaKey) {
        modifiers.push('cmd');
    }
    if (event.ctrlKey) {
        modifiers.push('ctrl');
    }
    if (event.altKey) {
        modifiers.push('alt');
    }
    if (event.shiftKey) {
        modifiers.push('shift');
    }
    const key = keyFromEvent(event);
    if (key === '') {
        return '';
    }
    modifiers.sort();
    return [...modifiers, key].join('+');
}

// isUsableBinding rejects shortcuts that would swallow ordinary typing. A
// binding needs a modifier unless its key is inherently non-printable.
export function isUsableBinding(binding: Keybinding): boolean {
    if (binding.command.trim() === '' || binding.command.startsWith('-')) {
        return false;
    }
    const sequence = normalizeSequence(binding.key);
    if (sequence.length === 0 || sequence.some((chord) => chord === '')) {
        return false;
    }
    return sequence.every((chord) => {
        const parts = chord.split('+');
        const key = parts[parts.length - 1];
        const hasModifier = parts.length > 1;
        return hasModifier || /^f([1-9]|1[0-9]|2[0-4])$/.test(key) || NON_TYPING_KEYS.has(key);
    });
}

const NON_TYPING_KEYS = new Set([
    'escape',
    'enter',
    'tab',
    'backspace',
    'delete',
    'up',
    'down',
    'left',
    'right',
    'home',
    'end',
    'pageup',
    'pagedown',
]);

type Token = { kind: 'identifier'; value: string } | { kind: 'operator'; value: string };

function tokenize(expression: string): Token[] {
    const tokens: Token[] = [];
    let index = 0;
    while (index < expression.length) {
        const char = expression[index];
        if (/\s/.test(char)) {
            index++;
            continue;
        }
        if (char === '!' || char === '(' || char === ')') {
            tokens.push({ kind: 'operator', value: char });
            index++;
            continue;
        }
        if (char === '&' && expression[index + 1] === '&') {
            tokens.push({ kind: 'operator', value: '&&' });
            index += 2;
            continue;
        }
        if (char === '|' && expression[index + 1] === '|') {
            tokens.push({ kind: 'operator', value: '||' });
            index += 2;
            continue;
        }
        const start = index;
        while (index < expression.length && /[A-Za-z0-9_]/.test(expression[index])) {
            index++;
        }
        if (start === index) {
            // An unsupported character makes the expression unsatisfiable
            // rather than silently true.
            return [{ kind: 'identifier', value: '__invalid__' }];
        }
        tokens.push({ kind: 'identifier', value: expression.slice(start, index) });
    }
    return tokens;
}

// matchesWhen evaluates a VS Code-style `when` expression over the editor's context
// keys. An empty expression always matches.
export function matchesWhen(expression: string | undefined, context: KeyContext): boolean {
    if (expression === undefined || expression.trim() === '') {
        return true;
    }
    const tokens = tokenize(expression);
    let index = 0;

    const peek = (): Token | undefined => tokens[index];
    const take = (): Token | undefined => tokens[index++];

    function parseOr(): boolean {
        let value = parseAnd();
        while (peek()?.kind === 'operator' && peek()?.value === '||') {
            take();
            const right = parseAnd();
            value = value || right;
        }
        return value;
    }

    function parseAnd(): boolean {
        let value = parseUnary();
        while (peek()?.kind === 'operator' && peek()?.value === '&&') {
            take();
            const right = parseUnary();
            value = value && right;
        }
        return value;
    }

    function parseUnary(): boolean {
        const token = take();
        if (token === undefined) {
            return false;
        }
        if (token.kind === 'operator' && token.value === '!') {
            return !parseUnary();
        }
        if (token.kind === 'operator' && token.value === '(') {
            const value = parseOr();
            if (peek()?.kind === 'operator' && peek()?.value === ')') {
                take();
            }
            return value;
        }
        if (token.kind === 'identifier') {
            switch (token.value) {
                case 'true':
                    return true;
                case 'false':
                case '__invalid__':
                    return false;
                default:
                    return context[token.value as keyof KeyContext] === true;
            }
        }
        return false;
    }

    const result = parseOr();
    return index === tokens.length ? result : false;
}

// DEFAULT_KEYBINDINGS are keys a native menu item cannot hold: second keys,
// chords, and commands with no menu item. keybindings.json entries win.
export const DEFAULT_KEYBINDINGS: readonly Keybinding[] = [
    { key: 'f1', command: 'command-palette' },
    { key: 'cmd+shift+e', command: 'show-explorer' },
    { key: 'ctrl+shift+g', command: 'show-changes' },
    // Inside the editor Monaco runs these (installEditorKeybindings), so its
    // own Cmd+K chords keep working.
    { key: 'cmd+k cmd+w', command: 'close-all-tabs', when: '!editorFocus' },
    { key: 'cmd+k f', command: 'close-all-folders', when: '!editorFocus' },
    { key: 'cmd+shift+]', command: 'next-tab' },
    { key: 'cmd+shift+[', command: 'previous-tab' },
    // F4 is Launchpad on Apple keyboards unless fn is held.
    { key: 'cmd+alt+g', command: 'find-next-result' },
    { key: 'cmd+alt+shift+g', command: 'find-previous-result' },
];

// withDefaultKeybindings appends the defaults the user has not removed with a
// "-command" entry or taken by binding the same first chord.
export function withDefaultKeybindings(user: readonly Keybinding[]): Keybinding[] {
    const removed = new Set(
        user.filter((binding) => binding.command.startsWith('-')).map((binding) => binding.command.slice(1)),
    );
    const claimed = new Set(user.map((binding) => normalizeSequence(binding.key)[0]));
    const defaults = DEFAULT_KEYBINDINGS.filter(
        (binding) => !removed.has(binding.command) && !claimed.has(normalizeSequence(binding.key)[0]),
    );
    return [...user, ...defaults];
}

export interface KeybindingHandlers {
    run: (command: string) => void;
    context: () => KeyContext;
}

interface Candidate {
    sequence: string[];
    command: string;
    when: string;
}

// installKeybindings installs one capture-phase keydown listener. It returns a
// dispose function. Sequences wait for their next chord for one second.
export function installKeybindings(
    bindings: readonly Keybinding[],
    handlers: KeybindingHandlers,
): () => void {
    const candidates: Candidate[] = [];
    for (const binding of bindings) {
        if (!isUsableBinding(binding)) {
            continue;
        }
        const sequence = normalizeSequence(binding.key);
        candidates.push({
            sequence,
            command: binding.command,
            when: binding.when ?? '',
        });
    }
    if (candidates.length === 0) {
        return () => {};
    }

    let pending: string[] = [];
    let pendingAt = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;

    function reset(): void {
        pending = [];
        if (timer !== undefined) {
            clearTimeout(timer);
            timer = undefined;
        }
    }

    function onKeydown(event: KeyboardEvent): void {
        if (event.isComposing || event.repeat) {
            return;
        }
        const chord = chordFromEvent(event);
        if (chord === '') {
            return;
        }

        if (pending.length > 0 && Date.now() - pendingAt > 1200) {
            reset();
        }

        const next = [...pending, chord];
        const context = handlers.context();
        // when is tested on every prefix: a chord that does not apply here must
        // not swallow its first key.
        const continuations = candidates.filter(
            (candidate) =>
                candidate.sequence.length >= next.length &&
                candidate.sequence.slice(0, next.length).every((part, index) => part === next[index]) &&
                matchesWhen(candidate.when, context),
        );
        if (continuations.length === 0) {
            reset();
            return;
        }

        const longer = continuations.some((candidate) => candidate.sequence.length > next.length);
        if (longer) {
            pending = next;
            pendingAt = Date.now();
            if (timer !== undefined) {
                clearTimeout(timer);
            }
            timer = setTimeout(reset, 1200);
            event.preventDefault();
            event.stopPropagation();
            return;
        }

        const exact = continuations.find((candidate) => candidate.sequence.length === next.length);
        reset();
        if (exact === undefined) {
            return;
        }
        event.preventDefault();
        event.stopPropagation();
        handlers.run(exact.command);
    }

    window.addEventListener('keydown', onKeydown, true);
    return () => {
        reset();
        window.removeEventListener('keydown', onKeydown, true);
    };
}
