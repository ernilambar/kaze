import raw from './fileIcons.json';

// `fileIcons.json` is generated data from the File Icons project: glyphs come
// from file-icons/icons and its companion fonts (MFixx, DevOpicons, Font
// Awesome 4.7), and filename patterns from file-icons/atom's `config.cson`.
// See `assets/LICENSES.md` for sources and licensing.
//
// One rule is hand-edited: dotenv sits first and matches `*.env` as well as
// `.env*`, so the icon set lines up with what editor.ts detects as dotenv.
// `.bak` names are excluded so backups keep the backup icon. Reapply that
// change if the data is regenerated.

// A glyph from one of the fonts in `assets/`. `size`, `top` and `left` are the
// per-icon tweaks the File Icons stylesheet applies to keep mixed families
// aligned on a 16px grid.
export interface FileIcon {
    glyph: string;
    family: string;
    size: number;
    top: number;
    left: number;
    color: string;
}

type Glyph = [glyph: string, family: string, size: number, top: number, left: number];
type Rule = [pattern: string, flags: string, icon: string, color: string | null];

interface IconData {
    glyphs: Record<string, Glyph>;
    rules: Rule[];
}

const { glyphs, rules } = raw as unknown as IconData;

// Unknown files get Font Awesome's hollow file outline rather than an empty
// slot, so every row keeps the same left edge.
const FALLBACK: FileIcon = {
    glyph: '\uf016',
    family: 'FontAwesome',
    size: 15,
    top: 0,
    left: 0,
    color: 'var(--fg-muted)',
};

const matchers = rules.map(([source, flags, icon, color]) => ({
    pattern: new RegExp(source, flags),
    icon,
    color,
}));

// Nine icons in the rule data were added to the File Icons project after its
// last font build, so the bundled font has no glyph for them and would paint a
// missing-glyph box. Each maps to the closest glyph the font does carry; the
// rule's colour is kept. Remove an entry once the fonts are regenerated.
const SUBSTITUTES: Record<string, string> = {
    'bikeshed-icon': 'css3-icon',
    'dotenv-icon': 'config-icon',
    'figma-icon': 'sketch-icon',
    'imgbot-icon': 'robot-icon',
    'mint-icon': 'js-icon',
    'mixin-icon': 'json-icon',
    'photorec-icon': 'terminal-icon',
    'shellcheck-icon': 'checklist-icon',
    'windi-icon': 'tailwind-icon',
};

// Names repeat across rows and refreshes, so the first match is kept. The
// rules are ordered by specificity by the generator.
const cache = new Map<string, FileIcon>();

function describe(icon: string, color: string | null): FileIcon {
    const glyph = glyphs[SUBSTITUTES[icon] ?? icon];
    if (glyph === undefined) {
        return FALLBACK;
    }
    const [character, family, size, top, left] = glyph;
    return { glyph: character, family, size, top, left, color: color ?? 'var(--fg-muted)' };
}

// fileIcon picks the icon for a file's basename. Directory names are not
// resolved here: the tree draws one folder shape for all of them.
export function fileIcon(name: string): FileIcon {
    const cached = cache.get(name);
    if (cached !== undefined) {
        return cached;
    }
    let icon: FileIcon | undefined;
    for (const rule of matchers) {
        if (rule.pattern.test(name)) {
            icon = describe(rule.icon, rule.color);
            break;
        }
    }
    icon ??= FALLBACK;
    cache.set(name, icon);
    return icon;
}
