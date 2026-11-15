import { describe, expect, it } from 'vitest';
import { splitWords, transformCase } from './case';

describe('splitWords', () => {
    it.each([
        ['fooBar', ['foo', 'Bar']],
        ['HTTPServer', ['HTTP', 'Server']],
        ['snake_case', ['snake', 'case']],
        ['kebab-case-2', ['kebab', 'case', '2']],
        ['Hello, world!', ['Hello', 'world']],
        ['version2Final', ['version', '2', 'Final']],
    ])('splits %s', (text, expected) => {
        expect(splitWords(text)).toEqual(expected);
    });
});

describe('transformCase', () => {
    it('upper and lower keep the text otherwise intact', () => {
        expect(transformCase('Hello, World!', 'upper')).toBe('HELLO, WORLD!');
        expect(transformCase('Hello, World!', 'lower')).toBe('hello, world!');
    });

    it('title case capitalizes each word without lowering the rest', () => {
        expect(transformCase("it's an iPhone app", 'title')).toBe("It's An IPhone App");
    });

    it.each([
        ['HTTP server_error-2', 'camel', 'httpServerError2'],
        ['HTTP server_error-2', 'pascal', 'HttpServerError2'],
        ['HTTP server_error-2', 'snake', 'http_server_error_2'],
        ['HTTP server_error-2', 'kebab', 'http-server-error-2'],
        ['HTTP server_error-2', 'constant', 'HTTP_SERVER_ERROR_2'],
        ['alreadyCamel', 'snake', 'already_camel'],
    ] as const)('converts %s to %s', (text, transform, expected) => {
        expect(transformCase(text, transform)).toBe(expected);
    });

    it('drops separators when there are no words', () => {
        expect(transformCase('---', 'camel')).toBe('');
        expect(transformCase('', 'pascal')).toBe('');
    });
});
