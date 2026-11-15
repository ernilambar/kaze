import { describe, expect, it } from 'vitest';
import { COMMANDS, rankCommands, type Command } from './commands';

function command(id: string, name: string, detail: string): Command {
    return { id, name, detail };
}

describe('rankCommands', () => {
    it('returns every command for an empty query', () => {
        expect(rankCommands('')).toEqual(COMMANDS);
    });

    it('matches a command by name', () => {
        const commands = [
            command('settings', 'Settings', 'Kova'),
            command('quick-open', 'Quick Open', 'File'),
        ];

        expect(rankCommands('quick', commands)[0]?.id).toBe('quick-open');
    });

    it('matches a command by category', () => {
        const matches = rankCommands('kova');

        expect(matches.map((entry) => entry.id)).toEqual([
            'settings',
            'open-keybindings',
            'open-settings-json',
            'open-themes-folder',
            'open-default-settings',
            'install-command-line-tool',
        ]);
    });

    it('prefers a name match over a category match', () => {
        const commands = [
            command('category', 'Settings', 'Open'),
            command('name', 'Open File', 'Kova'),
        ];

        expect(rankCommands('open', commands)[0]?.id).toBe('name');
    });

    it('excludes commands that do not match', () => {
        expect(rankCommands('zzz')).toEqual([]);
    });
});
