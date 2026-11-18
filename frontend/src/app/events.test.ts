import { beforeEach, describe, expect, it, vi } from 'vitest';

const { listeners, onWindowEvent, getLaunchRequest } = vi.hoisted(() => {
    const listeners = new Map<string, number>();
    const onWindowEvent = vi.fn((name: string) => {
        listeners.set(name, (listeners.get(name) ?? 0) + 1);
        return () => listeners.set(name, (listeners.get(name) ?? 0) - 1);
    });
    const getLaunchRequest = vi.fn(async () => ({ paths: [], buffers: [] }));
    return { listeners, onWindowEvent, getLaunchRequest };
});

vi.mock('../windowstate', () => ({ onWindowEvent }));
vi.mock('../../bindings/kaze/internal/app/app', () => ({ GetLaunchRequest: getLaunchRequest }));

import { subscribeAll, type AppEventHandlers } from './events';

function handlers(): AppEventHandlers {
    const noop = (): void => {};
    return {
        launch: vi.fn(),
        addFolder: noop,
        closeTab: noop,
        quickOpen: noop,
        commandPalette: noop,
        menuCommand: noop,
        fileCommands: {},
        closeRequest: noop,
        editorCommands: {},
        watchedChanges: noop,
        workspaceCommands: {},
        tabCommands: {
            next: noop,
            previous: noop,
            reopen: noop,
            goToTab: noop,
            moveToNewWindow: noop,
        },
        configChange: noop,
        appearance: noop,
        settingsOverlay: noop,
    };
}

describe('subscribeAll', () => {
    beforeEach(() => {
        listeners.clear();
        onWindowEvent.mockClear();
    });

    it('hands over the launch request before resolving', async () => {
        const h = handlers();
        await subscribeAll('main', h);
        expect(getLaunchRequest).toHaveBeenCalledWith('main');
        expect(h.launch).toHaveBeenCalledTimes(1);
    });

    it('removes every listener it installed', async () => {
        const off = await subscribeAll('main', handlers());
        expect(onWindowEvent).toHaveBeenCalled();
        expect([...listeners.values()].some((count) => count > 0)).toBe(true);
        off();
        expect([...listeners.values()].every((count) => count === 0)).toBe(true);
    });
});
