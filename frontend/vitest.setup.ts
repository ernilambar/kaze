import { vi } from 'vitest';

// The Wails runtime expects a live Wails window and starts DOM listeners on
// import. Unit tests only need the pure helpers behind the bindings, so the
// runtime is replaced with inert stubs before any test module loads.
vi.mock('@wailsio/runtime', () => ({
    Events: {
        On: () => () => {},
    },
    Call: {
        ByID: async () => null,
    },
    CancellablePromise: Promise,
}));
