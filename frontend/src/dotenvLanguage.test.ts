import { describe, expect, it } from 'vitest';
import { DOTENV_NAME } from './dotenvLanguage';

// languageForPath lowercases the basename before testing, so the test does too.
describe('DOTENV_NAME', () => {
    it.each([
        '.env',
        '.env.local',
        '.env.production.example',
        '.env.test.local',
        '.env.vault',
        '.ENV.LOCAL',
    ])('treats %s as dotenv', (name) => {
        expect(DOTENV_NAME.test(name.toLowerCase())).toBe(true);
    });

    it.each(['.envrc', '.env.bak', '.env.local.bak', '.ENV.LOCAL.BAK', 'env', 'prod.env.local'])(
        'does not treat %s as dotenv',
        (name) => {
            expect(DOTENV_NAME.test(name.toLowerCase())).toBe(false);
        },
    );
});
