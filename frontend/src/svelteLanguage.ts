import * as monaco from 'monaco-editor';

// SVELTE_LANGUAGE_ID is the id Monaco registers, and the one the status bar and
// Change Language Mode list read back from the registry.
export const SVELTE_LANGUAGE_ID = 'svelte';

// SVELTE_EXTENSIONS is what Auto Detect matches. Everything else (the palette
// entry, the status label) follows from the registration in editor.ts.
export const SVELTE_EXTENSIONS = ['.svelte'];

const EMPTY_ELEMENTS = [
    'area',
    'base',
    'br',
    'col',
    'embed',
    'hr',
    'img',
    'input',
    'keygen',
    'link',
    'menuitem',
    'meta',
    'param',
    'source',
    'track',
    'wbr',
];

// svelteConf is HTML's language configuration: Svelte markup is HTML with
// `{...}` holes, so comments, brackets and auto-closing pairs behave the same.
export const svelteConf: monaco.languages.LanguageConfiguration = {
    wordPattern:
        /(-?\d*\.\d\w*)|([^\`\~\!\@\$\^\&\*\(\)\=\+\[\{\]\}\\\|\;\:\'\"\,\.\<\>\/\s]+)/g,
    comments: {
        blockComment: ['<!--', '-->'],
    },
    brackets: [
        ['<!--', '-->'],
        ['<', '>'],
        ['{', '}'],
        ['(', ')'],
    ],
    autoClosingPairs: [
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '(', close: ')' },
        { open: '"', close: '"' },
        { open: "'", close: "'" },
    ],
    surroundingPairs: [
        { open: '"', close: '"' },
        { open: "'", close: "'" },
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '(', close: ')' },
        { open: '<', close: '>' },
    ],
    onEnterRules: [
        {
            beforeText: new RegExp(
                `<(?!(?:${EMPTY_ELEMENTS.join('|')}))([_:\\w][_:\\w-.\\d]*)([^/>]*(?!/)>)[^<]*$`,
                'i',
            ),
            afterText: /^<\/([_:\w][_:\w-.\d]*)\s*>$/i,
            action: { indentAction: monaco.languages.IndentAction.IndentOutdent },
        },
        {
            beforeText: new RegExp(
                `<(?!(?:${EMPTY_ELEMENTS.join('|')}))(\\w[\\w\\d]*)([^/>]*(?!/)>)[^<]*$`,
                'i',
            ),
            action: { indentAction: monaco.languages.IndentAction.Indent },
        },
    ],
};

// svelteLanguage is Monaco's HTML Monarch grammar with Svelte's two extra
// syntaxes layered on: `{...}` template expressions and blocks (embedded
// JavaScript), and `<script lang="ts">` / `<style lang="scss">` bodies.
// Duplicating the HTML states keeps the grammar self-contained, since Monaco
// does not ship a Svelte tokenizer and exposes no API to borrow HTML's.
export const svelteLanguage: monaco.languages.IMonarchLanguage = {
    defaultToken: '',
    tokenPostfix: '.svelte',
    ignoreCase: true,
    tokenizer: {
        root: [
            [/<!DOCTYPE/, 'metatag', '@doctype'],
            [/<!--/, 'comment', '@comment'],
            // Block openers and closers: `{#if x}`, `{:else}`, `{/if}`,
            // `{@html value}`. The marker is the keyword; the rest of the tag
            // is a JavaScript expression. This rule must precede the plain
            // expression rule below.
            [
                /\{(?:[#/:@][a-z]*)/,
                {
                    token: 'keyword',
                    next: '@expression',
                    nextEmbedded: 'text/javascript',
                },
            ],
            // A plain `{expression}` anywhere in the markup.
            [
                /\{/,
                {
                    token: 'delimiter.bracket',
                    next: '@expression',
                    nextEmbedded: 'text/javascript',
                },
            ],
            [/(<)((?:[\w\-]+:)?[\w\-]+)(\s*)(\/>)/, ['delimiter', 'tag', '', 'delimiter']],
            [/(<)(script)/, ['delimiter', { token: 'tag', next: '@script' }]],
            [/(<)(style)/, ['delimiter', { token: 'tag', next: '@style' }]],
            [/(<)((?:[\w\-]+:)?[\w\-]+)/, ['delimiter', { token: 'tag', next: '@otherTag' }]],
            [/(<\/)((?:[\w\-]+:)?[\w\-]+)/, ['delimiter', { token: 'tag', next: '@otherTag' }]],
            [/</, 'delimiter'],
            [/[^<{]+/, ''], // text
        ],
        doctype: [
            [/[^>]+/, 'metatag.content'],
            [/>/, 'metatag', '@pop'],
        ],
        comment: [
            [/-->/, 'comment', '@pop'],
            [/[^-]+/, 'comment.content'],
            [/./, 'comment.content'],
        ],
        // The expression state exists only to hand its contents to JavaScript
        // and to pop back out at the closing brace. Nested braces inside the
        // expression are JavaScript's business.
        expression: [
            [/\}/, { token: 'delimiter.bracket', next: '@pop', nextEmbedded: '@pop' }],
        ],
        otherTag: [
            [
                /\{/,
                {
                    token: 'delimiter.bracket',
                    next: '@expression',
                    nextEmbedded: 'text/javascript',
                },
            ],
            [/\/?>/, 'delimiter', '@pop'],
            [/"([^"]*)"/, 'attribute.value'],
            [/'([^']*)'/, 'attribute.value'],
            [/[\w\-]+/, 'attribute.name'],
            [/=/, 'delimiter'],
            [/[ \t\r\n]+/, ''], // whitespace
        ],
        // -- BEGIN <script> tags handling
        script: [
            [/(?:type|lang)/, 'attribute.name', '@scriptAfterType'],
            [/"([^"]*)"/, 'attribute.value'],
            [/'([^']*)'/, 'attribute.value'],
            [/[\w\-]+/, 'attribute.name'],
            [/=/, 'delimiter'],
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@scriptEmbedded',
                    nextEmbedded: 'text/javascript',
                },
            ],
            [/[ \t\r\n]+/, ''], // whitespace
            [/(<\/)(script\s*)(>)/, ['delimiter', 'tag', { token: 'delimiter', next: '@pop' }]],
        ],
        // After <script ... type|lang
        scriptAfterType: [
            [/=/, 'delimiter', '@scriptAfterTypeEquals'],
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@scriptEmbedded',
                    nextEmbedded: 'text/javascript',
                },
            ],
            [/[ \t\r\n]+/, ''], // whitespace
            // cover invalid e.g. <script type>
            [/<\/script\s*>/, { token: '@rematch', next: '@pop' }],
        ],
        // After <script ... type|lang =
        scriptAfterTypeEquals: [
            [
                /"module"/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.text/javascript',
                },
            ],
            [
                /'module'/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.text/javascript',
                },
            ],
            // Svelte writes the script language as `lang="ts"`; a bare
            // language name is not a registered id, so map the shorthands.
            [
                /"(?:ts|typescript)"/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.typescript',
                },
            ],
            [
                /'(?:ts|typescript)'/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.typescript',
                },
            ],
            [
                /"(?:js|javascript)"/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.text/javascript',
                },
            ],
            [
                /'(?:js|javascript)'/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.text/javascript',
                },
            ],
            [
                /"([^"]*)"/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.$1',
                },
            ],
            [
                /'([^']*)'/,
                {
                    token: 'attribute.value',
                    switchTo: '@scriptWithCustomType.$1',
                },
            ],
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@scriptEmbedded',
                    nextEmbedded: 'text/javascript',
                },
            ],
            [/[ \t\r\n]+/, ''], // whitespace
            // cover invalid e.g. <script type=>
            [/<\/script\s*>/, { token: '@rematch', next: '@pop' }],
        ],
        // After <script ... type|lang = $S2
        scriptWithCustomType: [
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@scriptEmbedded.$S2',
                    nextEmbedded: '$S2',
                },
            ],
            [/"([^"]*)"/, 'attribute.value'],
            [/'([^']*)'/, 'attribute.value'],
            [/[\w\-]+/, 'attribute.name'],
            [/=/, 'delimiter'],
            [/[ \t\r\n]+/, ''], // whitespace
            [/<\/script\s*>/, { token: '@rematch', next: '@pop' }],
        ],
        scriptEmbedded: [
            [/<\/script/, { token: '@rematch', next: '@pop', nextEmbedded: '@pop' }],
            [/[^<]+/, ''],
        ],
        // -- END <script> tags handling
        // -- BEGIN <style> tags handling
        style: [
            [/(?:type|lang)/, 'attribute.name', '@styleAfterType'],
            [/"([^"]*)"/, 'attribute.value'],
            [/'([^']*)'/, 'attribute.value'],
            [/[\w\-]+/, 'attribute.name'],
            [/=/, 'delimiter'],
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@styleEmbedded',
                    nextEmbedded: 'text/css',
                },
            ],
            [/[ \t\r\n]+/, ''], // whitespace
            [/(<\/)(style\s*)(>)/, ['delimiter', 'tag', { token: 'delimiter', next: '@pop' }]],
        ],
        // After <style ... type|lang
        styleAfterType: [
            [/=/, 'delimiter', '@styleAfterTypeEquals'],
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@styleEmbedded',
                    nextEmbedded: 'text/css',
                },
            ],
            [/[ \t\r\n]+/, ''], // whitespace
            [/<\/style\s*>/, { token: '@rematch', next: '@pop' }],
        ],
        // After <style ... type|lang =
        styleAfterTypeEquals: [
            [
                /"([^"]*)"/,
                {
                    token: 'attribute.value',
                    switchTo: '@styleWithCustomType.$1',
                },
            ],
            [
                /'([^']*)'/,
                {
                    token: 'attribute.value',
                    switchTo: '@styleWithCustomType.$1',
                },
            ],
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@styleEmbedded',
                    nextEmbedded: 'text/css',
                },
            ],
            [/[ \t\r\n]+/, ''], // whitespace
            [/<\/style\s*>/, { token: '@rematch', next: '@pop' }],
        ],
        // After <style ... type|lang = $S2
        styleWithCustomType: [
            [
                />/,
                {
                    token: 'delimiter',
                    next: '@styleEmbedded.$S2',
                    nextEmbedded: '$S2',
                },
            ],
            [/"([^"]*)"/, 'attribute.value'],
            [/'([^']*)'/, 'attribute.value'],
            [/[\w\-]+/, 'attribute.name'],
            [/=/, 'delimiter'],
            [/[ \t\r\n]+/, ''], // whitespace
            [/<\/style\s*>/, { token: '@rematch', next: '@pop' }],
        ],
        styleEmbedded: [
            [/<\/style/, { token: '@rematch', next: '@pop', nextEmbedded: '@pop' }],
            [/[^<]+/, ''],
        ],
        // -- END <style> tags handling
    },
};
