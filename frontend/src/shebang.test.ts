import { describe, expect, it } from 'vitest';
import { shebangInterpreter, shebangLanguage } from './shebang';

describe('shebangInterpreter', () => {
    it.each([
        ['#!/bin/sh', 'sh'],
        ['#!/usr/bin/env bash', 'bash'],
        ['#!/usr/bin/env -S python3 -u', 'python3'],
        ['#!/usr/bin/env FOO=bar node', 'node'],
        ['#!C:\\Python\\python.exe', 'python'],
        ['#!/bin/zsh -l', 'zsh'],
    ])('reads %s', (line, expected) => {
        expect(shebangInterpreter(line)).toBe(expected);
    });

    it('rejects lines that are not shebangs', () => {
        expect(shebangInterpreter('print("hi")')).toBe('');
        expect(shebangInterpreter('#!')).toBe('');
        expect(shebangInterpreter('#!/usr/bin/env')).toBe('');
    });
});

describe('shebangLanguage', () => {
    it.each([
        ['#!/bin/bash\nset -e\n', 'shell'],
        ['#!/usr/bin/env python3\nprint("hi")\n', 'python'],
        ['#!/usr/bin/env python3.12\nprint("hi")\n', 'python'],
        ['#!/usr/bin/env php8.2\n<?php\n', 'php'],
        ['#!/usr/bin/env node\nconsole.log(1);\n', 'javascript'],
        ['#!/usr/bin/ruby\nputs 1\n', 'ruby'],
    ])('detects a language for %s', (text, expected) => {
        expect(shebangLanguage(text)).toBe(expected);
    });

    it('looks only at the first line', () => {
        expect(shebangLanguage('#!/bin/sh\necho hi\n')).toBe('shell');
        expect(shebangLanguage('echo hi\n#!/bin/sh\n')).toBeUndefined();
    });

    it('returns nothing for a shebang with no bundled language', () => {
        expect(shebangLanguage('#!/usr/bin/osascript')).toBeUndefined();
        expect(shebangLanguage('#!/usr/bin/awk -f')).toBeUndefined();
    });

    it('returns nothing for a file without a shebang', () => {
        expect(shebangLanguage('')).toBeUndefined();
        expect(shebangLanguage('hello\n')).toBeUndefined();
    });
});
