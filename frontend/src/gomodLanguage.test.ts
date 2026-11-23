import { describe, expect, it } from 'vitest';
import * as monaco from 'monaco-editor';
import './editor';
import { GOMOD_LANGUAGE_ID } from './gomodLanguage';

// A representative go.mod: every directive form the go command writes —
// bare and parenthesised requires, toolchain and godebug, replace with a local
// path, retract with a range, pseudo-versions, a comment written without a
// space before it, and a standalone line comment.
const GO_MOD = `module example.com/foo

go 1.22

toolchain go1.23.4

require golang.org/x/text v0.21.0 // indirect

require (
	golang.org/x/sys v0.28.0
	gopkg.in/yaml.v3 v3.0.1
)

replace example.com/bar => ./local/bar

retract [v1.0.0, v1.2.0]

godebug default=go1.21// generated

// standalone
`;

// Monarch appends the tokenPostfix to each type; strip the `.gomod` suffix
// so the assertions read like the grammar's own token names.
function tokensOf(line: number, text: string = GO_MOD): string[] {
    return monaco.editor.tokenize(text, GOMOD_LANGUAGE_ID)[line].map((token) =>
        token.type.endsWith(`.${GOMOD_LANGUAGE_ID}`) ? token.type.slice(0, -GOMOD_LANGUAGE_ID.length - 1) : token.type);
}

describe('gomod grammar', () => {
    it('tokenizes directives, paths and versions', () => {
        // `module example.com/foo`
        expect(tokensOf(0)).toEqual(['keyword', '', 'key']);
        // `go 1.22`
        expect(tokensOf(2)).toEqual(['keyword', '', 'number']);
        // `toolchain go1.23.4`
        expect(tokensOf(4)).toEqual(['keyword', '', 'number']);
        // `require golang.org/x/text v0.21.0 // indirect`
        expect(tokensOf(6)).toEqual(['keyword', '', 'key', '', 'number', '', 'comment']);
        // `replace example.com/bar => ./local/bar`
        expect(tokensOf(13)).toEqual(['keyword', '', 'key', '', 'operator', '', 'key']);
        // `retract [v1.0.0, v1.2.0]`
        expect(tokensOf(15)).toEqual([
            'keyword',
            '',
            'delimiter.bracket',
            'number',
            'delimiter',
            '',
            'number',
            'delimiter.bracket',
        ]);
        // `godebug default=go1.21// generated`: the comment starts even though
        // no space precedes it.
        expect(tokensOf(17)).toEqual(['keyword', '', 'key', 'comment']);
    });

    it('tokenizes a parenthesised block and line comments', () => {
        // `require (` opening the block.
        expect(tokensOf(8)).toEqual(['keyword', '', 'delimiter.bracket']);
        // `golang.org/x/sys v0.28.0` inside it: a `gopkg.in` path must stay
        // one key token, not split at the `go` its name starts with.
        expect(tokensOf(9)).toEqual(['', 'key', '', 'number']);
        expect(tokensOf(10)).toEqual(['', 'key', '', 'number']);
        expect(tokensOf(19)).toEqual(['comment']);
    });

    it('keeps an escaped quote inside a quoted string', () => {
        const tokens = tokensOf(0, 'replace a => "./b\\"c"\n').slice(6);
        expect(tokens.length).toBeGreaterThan(0);
        expect(tokens.every((token) => token === 'string')).toBe(true);
    });

    it('tokenizes a go.work use block', () => {
        const work = 'use (\n\t./foo\n\t./bar\n)\n';
        expect(tokensOf(0, work)).toEqual(['keyword', '', 'delimiter.bracket']);
        expect(tokensOf(1, work)).toEqual(['', 'key']);
        expect(tokensOf(3, work)).toEqual(['delimiter.bracket']);
    });
});
