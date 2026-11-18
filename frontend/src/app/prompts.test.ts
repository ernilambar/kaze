import { describe, expect, it } from 'vitest';
import { Prompts } from './prompts.svelte';

describe('Prompts', () => {
    it('resolves askConfirm with the answer and clears the prompt', async () => {
        const prompts = new Prompts();
        const answer = prompts.askConfirm('Delete', 'Delete it?', 'Delete');
        expect(prompts.confirm?.actions.map((action) => action.id)).toEqual(['confirm', 'cancel']);
        prompts.answerConfirm('confirm');
        expect(prompts.confirm).toBeNull();
        await expect(answer).resolves.toBe(true);
    });

    it('maps an unknown conflict answer to cancel', async () => {
        const prompts = new Prompts();
        const answer = prompts.askConflict('Changed', 'Overwrite?');
        prompts.answerConfirm('other');
        await expect(answer).resolves.toBe('cancel');
    });

    it('ignores an unsaved answer outside save, discard and cancel', async () => {
        const prompts = new Prompts();
        const answer = prompts.askUnsaved({ title: 'Close', message: '', details: [], confirmLabel: 'Save' });
        prompts.answerUnsaved('maybe');
        expect(prompts.unsaved).not.toBeNull();
        prompts.answerUnsaved('discard');
        await expect(answer).resolves.toBe('discard');
    });

    it('reopens the name prompt with the refusal until a name is accepted', async () => {
        const prompts = new Prompts();
        const tried: string[] = [];
        const done = prompts.askName({
            title: 'Rename',
            value: 'a.txt',
            run: async (value) => {
                tried.push(value);
                return value === 'taken.txt' ? 'Name is taken' : '';
            },
        });
        prompts.submitName('taken.txt');
        await Promise.resolve();
        await Promise.resolve();
        expect(prompts.name?.error).toBe('Name is taken');
        expect(prompts.name?.value).toBe('taken.txt');
        prompts.submitName('free.txt');
        await expect(done).resolves.toBe(true);
        expect(tried).toEqual(['taken.txt', 'free.txt']);
        expect(prompts.name).toBeNull();
    });

    it('resolves askName with false on cancel', async () => {
        const prompts = new Prompts();
        const done = prompts.askName({ title: 'New File', run: async () => '' });
        prompts.cancelName();
        await expect(done).resolves.toBe(false);
    });
});
