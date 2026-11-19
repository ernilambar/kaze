import { describe, expect, it } from 'vitest';
import { toWorkspaceItems } from './workspace';

describe('toWorkspaceItems', () => {
    it('labels a workspace with its file name and its folder', () => {
        expect(toWorkspaceItems(['/projects/kaze/kaze.kaze-workspace'])).toEqual([
            {
                id: '/projects/kaze/kaze.kaze-workspace',
                name: 'kaze',
                detail: '/projects/kaze',
            },
        ]);
    });

    it('keeps a name containing a dot intact', () => {
        const [item] = toWorkspaceItems(['/projects/api/v2.api.kaze-workspace']);

        expect(item?.name).toBe('v2.api');
    });

    it('has no detail when the file sits at the root', () => {
        const [item] = toWorkspaceItems(['/work.kaze-workspace']);

        expect(item?.detail).toBe('');
    });

    it('keeps the given order', () => {
        const items = toWorkspaceItems(['/b/second.kaze-workspace', '/a/first.kaze-workspace']);

        expect(items.map((item) => item.name)).toEqual(['second', 'first']);
    });
});
