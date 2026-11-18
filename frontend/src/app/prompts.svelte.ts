// The window's modal prompts. Each ask* call shows one and returns a promise
// that resolves with the answer, so a save or close sequence reads top to
// bottom. App.svelte renders whichever prompt is set.

// UnsavedPrompt is the dialog asking what to do with modified buffers
// before they are closed. answer resolves the promise the caller is
// awaiting, so a save or close sequence reads top to bottom.
export interface UnsavedPrompt {
    title: string;
    message: string;
    details: string[];
    confirmLabel: string;
    answer: (answer: 'save' | 'discard' | 'cancel') => void;
}

// ConfirmPrompt asks whether a save may go ahead when it would lose
// something: another program's newer file, or bytes that were never valid
// UTF-8 and are now replacement characters. A conflict offers Reload as a
// third answer, so the prompt carries its actions instead of a boolean.
export interface ConfirmPrompt {
    title: string;
    message: string;
    actions: { id: string; label: string; primary?: boolean }[];
    answer: (id: string) => void;
}

// NamePromptState is the one-line dialog that asks for a name: a new file, a
// new folder, a rename or a duplicate. answer resolves what the operation
// awaiting it needs, so each command reads top to bottom.
export interface NamePromptState {
    title: string;
    label?: string;
    value: string;
    selectStem: boolean;
    selectFolder: boolean;
    submitLabel: string;
    error: string;
    answer: (value: string) => void;
    cancel: () => void;
}

export class Prompts {
    unsaved = $state<UnsavedPrompt | null>(null);
    confirm = $state<ConfirmPrompt | null>(null);
    name = $state<NamePromptState | null>(null);

    // askUnsaved shows the prompt and resolves with the action the user picked.
    askUnsaved = (options: Omit<UnsavedPrompt, 'answer'>): Promise<'save' | 'discard' | 'cancel'> => {
        return new Promise((resolve) => {
            this.unsaved = { ...options, answer: resolve };
        });
    };

    answerUnsaved = (answer: string): void => {
        if (answer !== 'save' && answer !== 'discard' && answer !== 'cancel') {
            return;
        }
        const prompt = this.unsaved;
        this.unsaved = null;
        prompt?.answer(answer);
    };

    askConfirm = (
        title: string,
        message: string,
        confirmLabel: string,
    ): Promise<boolean> => {
        return new Promise((resolve) => {
            this.confirm = {
                title,
                message,
                actions: [
                    { id: 'confirm', label: confirmLabel, primary: true },
                    { id: 'cancel', label: 'Cancel' },
                ],
                answer: (id) => resolve(id === 'confirm'),
            };
        });
    };

    // askConflict is the save prompt for a file another program changed. Reload
    // is a separate answer because the disk copy may be the newer one, and
    // discarding the buffer is then the right move.
    askConflict = (
        title: string,
        message: string,
    ): Promise<'overwrite' | 'reload' | 'cancel'> => {
        return new Promise((resolve) => {
            this.confirm = {
                title,
                message,
                actions: [
                    { id: 'overwrite', label: 'Overwrite', primary: true },
                    { id: 'reload', label: 'Reload from Disk' },
                    { id: 'cancel', label: 'Cancel' },
                ],
                answer: (id) =>
                    resolve(id === 'overwrite' || id === 'reload' ? id : 'cancel'),
            };
        });
    };

    answerConfirm = (id: string): void => {
        const prompt = this.confirm;
        this.confirm = null;
        prompt?.answer(id);
    };

    // askName opens the one-line dialog and resolves with the name the user
    // submitted, or an empty string when they cancelled. The dialog stays open
    // while a refusal is shown in it, so a taken name is corrected in place.
    askName = (options: {
        title: string;
        label?: string;
        value?: string;
        selectStem?: boolean;
        selectFolder?: boolean;
        submitLabel?: string;
        run: (value: string) => Promise<string>;
    }): Promise<boolean> => {
        return new Promise((resolve) => {
            const answer = (value: string): void => {
                // The dialog was already closed by the caller's submit path, so
                // the state is only read to know what to do next.
                void (async () => {
                    const refusal = await options.run(value);
                    if (refusal === '') {
                        resolve(true);
                        return;
                    }
                    this.name = {
                        title: options.title,
                        label: options.label ?? '',
                        value,
                        selectStem: options.selectStem ?? false,
                        selectFolder: options.selectFolder ?? false,
                        submitLabel: options.submitLabel ?? options.title,
                        error: refusal,
                        answer,
                        cancel: () => {
                            this.name = null;
                            resolve(false);
                        },
                    };
                })();
            };
            this.name = {
                title: options.title,
                label: options.label ?? '',
                value: options.value ?? '',
                selectStem: options.selectStem ?? false,
                selectFolder: options.selectFolder ?? false,
                submitLabel: options.submitLabel ?? options.title,
                error: '',
                answer,
                cancel: () => {
                    this.name = null;
                    resolve(false);
                },
            };
        });
    };

    // submitName closes the dialog and hands the value to the pending command,
    // which reopens it with the refusal if the backend says no.
    submitName = (value: string): void => {
        const prompt = this.name;
        this.name = null;
        prompt?.answer(value);
    };

    cancelName = (): void => {
        const prompt = this.name;
        this.name = null;
        prompt?.cancel();
    };
}
