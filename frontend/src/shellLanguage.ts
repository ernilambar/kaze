import type * as monaco from 'monaco-editor';

// SHELL_LANGUAGE_ID is Monaco's own shell language, registered by the
// monaco-editor import (`.sh`, `.bash`, and the shell shebangs resolved in
// shebang.ts). Only its tokenizer is replaced here.
export const SHELL_LANGUAGE_ID = 'shell';

// shellLanguage replaces Monaco's bundled shell grammar
// (esm/vs/languages/definitions/shell/shell.js, MIT), whose `$(...)` and
// `${...}` bodies end at the first `)` or `}` they meet. A quoted one — as in
// `$(php -r "echo rawurlencode('$1');")` — pops the substitution early, leaves
// the rest of the line to the root rules, and the opening `"` there swallows
// every following line as a string. That whole-file repaint is why this grammar
// exists; keyword and builtin colors are unchanged from Monaco's.
//
// The nested bodies below push on `$(`/`${` and on a plain `(` (subshells and
// arithmetic), pop on the matching delimiter, and keep quotes opaque so a
// quoted delimiter cannot close the body. Everything else is copied verbatim.
export const shellLanguage: monaco.languages.IMonarchLanguage = {
    defaultToken: '',
    ignoreCase: true,
    tokenPostfix: '.shell',
    brackets: [
        { token: 'delimiter.bracket', open: '{', close: '}' },
        { token: 'delimiter.parenthesis', open: '(', close: ')' },
        { token: 'delimiter.square', open: '[', close: ']' },
    ],
    keywords: [
        'if',
        'then',
        'do',
        'else',
        'elif',
        'while',
        'until',
        'for',
        'in',
        'esac',
        'fi',
        'fin',
        'fil',
        'done',
        'exit',
        'set',
        'unset',
        'export',
        'function',
    ],
    builtins: [
        'ab',
        'awk',
        'bash',
        'beep',
        'cat',
        'cc',
        'cd',
        'chown',
        'chmod',
        'chroot',
        'clear',
        'cp',
        'curl',
        'cut',
        'diff',
        'echo',
        'find',
        'gawk',
        'gcc',
        'get',
        'git',
        'grep',
        'hg',
        'kill',
        'killall',
        'ln',
        'ls',
        'make',
        'mkdir',
        'openssl',
        'mv',
        'nc',
        'node',
        'npm',
        'ping',
        'ps',
        'restart',
        'rm',
        'rmdir',
        'sed',
        'service',
        'sh',
        'shopt',
        'shred',
        'source',
        'sort',
        'sleep',
        'ssh',
        'start',
        'stop',
        'su',
        'sudo',
        'svn',
        'tee',
        'telnet',
        'top',
        'touch',
        'vi',
        'vim',
        'wall',
        'wc',
        'wget',
        'who',
        'write',
        'yes',
        'zsh',
    ],
    startingWithDash: /-+\w+/,
    identifiersWithDashes: /[a-zA-Z]\w+(?:@startingWithDash)+/,
    symbols: /[=><!~?&|+\-*\/\^;\.,]+/,
    tokenizer: {
        root: [
            [/@identifiersWithDashes/, ''],
            [/(\s)((?:@startingWithDash)+)/, ['white', 'attribute.name']],
            [
                /[a-zA-Z]\w*/,
                {
                    cases: {
                        '@keywords': 'keyword',
                        '@builtins': 'type.identifier',
                        '@default': '',
                    },
                },
            ],
            { include: '@whitespace' },
            { include: '@strings' },
            { include: '@parameters' },
            { include: '@heredoc' },
            [/[{}\[\]()]/, '@brackets'],
            [/@symbols/, 'delimiter'],
            { include: '@numbers' },
            [/[,;]/, 'delimiter'],
        ],
        whitespace: [
            [/\s+/, 'white'],
            [/(^#!.*$)/, 'metatag'],
            [/(^#.*$)/, 'comment'],
        ],
        numbers: [
            [/\d*\.\d+([eE][-+]?\d+)?/, 'number.float'],
            [/0[xX][0-9a-fA-F_]*[0-9a-fA-F]/, 'number.hex'],
            [/\d+/, 'number'],
        ],
        // Strings, including those broken across lines.
        strings: [
            [/'/, 'string', '@stringBody'],
            [/"/, 'string', '@dblStringBody'],
        ],
        stringBody: [
            [/'/, 'string', '@popall'],
            [/./, 'string'],
        ],
        dblStringBody: [
            [/"/, 'string', '@popall'],
            [/./, 'string'],
        ],
        heredoc: [
            [
                /(<<[-<]?)(\s*)(['"`]?)([\w-]+)(['"`]?)/,
                [
                    'constants',
                    'white',
                    'string.heredoc.delimiter',
                    'string.heredoc',
                    'string.heredoc.delimiter',
                ],
            ],
        ],
        parameters: [
            [/\$\d+/, 'variable.predefined'],
            [/\$\w+/, 'variable'],
            [/\$[*@#?\-$!0_]/, 'variable'],
            [/\$'/, 'variable', '@parameterBodyQuote'],
            [/\$"/, 'variable', '@parameterBodyDoubleQuote'],
            [/\$\(/, 'variable', '@parameterBodyParen'],
            [/\$\{/, 'variable', '@parameterBodyCurlyBrace'],
        ],
        parameterBodyQuote: [
            [/[^#:%*@\-!_']+/, 'variable'],
            [/[#:%*@\-!_]/, 'delimiter'],
            [/'/, 'variable', '@pop'],
        ],
        parameterBodyDoubleQuote: [
            [/[^#:%*@\-!_"]+/, 'variable'],
            [/[#:%*@\-!_]/, 'delimiter'],
            [/"/, 'variable', '@pop'],
        ],
        // `$(...)`: parentheses nest, so only the `)` that balances the body
        // ends it. Quotes are opaque: a quoted `)` is part of the argument.
        parameterBodyParen: [
            [/\$\(/, 'variable', '@push'],
            [/\(/, 'delimiter.parenthesis', '@push'],
            [/\)/, 'variable', '@pop'],
            [/'/, 'string', '@parenSingle'],
            [/"/, 'string', '@parenDouble'],
            [/[#:%*@\-!_]+/, 'delimiter'],
            [/[^#:%*@\-!_'"()]+/, 'variable'],
        ],
        parenSingle: [
            [/'/, 'string', '@pop'],
            [/./, 'string'],
        ],
        parenDouble: [
            [/\\./, 'string'],
            [/"/, 'string', '@pop'],
            [/./, 'string'],
        ],
        // `${...}`: a nested `${` opens another body; a plain `{` is content.
        // Quotes are opaque here too, so `"}"` cannot end the expansion.
        parameterBodyCurlyBrace: [
            [/\$\{/, 'variable', '@push'],
            [/\}/, 'variable', '@pop'],
            [/'/, 'string', '@curlySingle'],
            [/"/, 'string', '@curlyDouble'],
            [/[#:%*@\-!_]+/, 'delimiter'],
            [/\{/, 'variable'],
            [/[^#:%*@\-!_'"{}]+/, 'variable'],
        ],
        curlySingle: [
            [/'/, 'string', '@pop'],
            [/./, 'string'],
        ],
        curlyDouble: [
            [/"/, 'string', '@pop'],
            [/./, 'string'],
        ],
    },
};
