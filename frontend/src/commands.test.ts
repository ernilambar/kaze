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
            command('settings', 'Settings', 'Application'),
            command('quick-open', 'Quick Open', 'File'),
        ];

        expect(rankCommands('quick', commands)[0]?.id).toBe('quick-open');
    });

    it('matches a command by category', () => {
        const commands = [
            command('open-themes-folder', 'Open Themes Folder', 'Settings'),
            command('quick-open', 'Quick Open', 'File'),
        ];

        expect(rankCommands('settings', commands)).toEqual([commands[0]]);
    });

    it('prefers a name match over a category match', () => {
        const commands = [
            command('category', 'Settings', 'Open'),
            command('name', 'Open File', 'Application'),
        ];

        expect(rankCommands('open', commands)[0]?.id).toBe('name');
    });

    it('excludes commands that do not match', () => {
        expect(rankCommands('zzz')).toEqual([]);
    });
});

describe('indentation commands', () => {
    it('lists the Monaco indentation actions under Indentation', () => {
        const indent = COMMANDS.filter((entry) => entry.detail === 'Indentation');
        expect(indent.map((entry) => entry.id)).toEqual([
            'indent-using-spaces',
            'indent-using-tabs',
            'convert-indentation-to-spaces',
            'convert-indentation-to-tabs',
        ]);
    });
});
