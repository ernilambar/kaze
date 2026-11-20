import raw from './fileIcons.json';

// `fileIcons.json` is generated data from the File Icons project: glyphs come
// from file-icons/icons and its companion fonts (MFixx, DevOpicons, Font
// Awesome 4.7), and filename patterns from file-icons/atom's `config.cson`.
// See `assets/LICENSES.md` for sources and licensing.
//
// Two rules are hand-edited. Dotenv sits first and matches `*.env` as well as
// `.env*`, so the icon set lines up with what editor.ts detects as dotenv;
// `.bak` names are excluded so backups keep the backup icon. A markdown rule
// also sits ahead of the bookkeeping rule, so a Markdown README keeps the
// markdown icon instead of the book. Reapply those changes if the data is
// regenerated.

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

// Glyphs the bundled fonts carry but the generated data does not name.
// `markdown-icon` uses Devicons' markdown outline instead of the Font Awesome
// file outline the data points at.
const EXTRA_GLYPHS: Record<string, Glyph> = {
    'devicons-markdown-icon': ['\ue63e', 'Devicons', 16, 3, 0],
};

// Icons whose generated data names a different mark than the one this tree
// wants. The rule's colour is kept.
const GLYPH_OVERRIDES: Record<string, string> = {
    'markdown-icon': 'devicons-markdown-icon',
};

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
    const key = GLYPH_OVERRIDES[icon] ?? SUBSTITUTES[icon] ?? icon;
    const glyph = glyphs[key] ?? EXTRA_GLYPHS[key];
    if (glyph === undefined) {
        return FALLBACK;
    }
    const [character, family, size, top, left] = glyph;
    return { glyph: character, family, size, top, left, color: color ?? 'var(--fg-muted)' };
}

// folderIcon picks the mark for a folder's basename, or nothing when the
// tree should draw its plain folder shape. Only names whose meaning is lost as
// a generic folder get an entry.
const FOLDER_ICONS: Record<string, FileIcon> = {
    '.github': {
        glyph: '\uf09b',
        family: 'FontAwesome',
        size: 15,
        top: 0,
        left: 0,
        color: 'var(--fg)',
    },
    node_modules: {
        glyph: '\ue618',
        family: 'Devicons',
        size: 16,
        top: 3,
        left: 0,
        color: '#90a959',
    },
};

export function folderIcon(name: string): FileIcon | undefined {
    return FOLDER_ICONS[name.toLowerCase()];
}

// fileIcon picks the icon for a file's basename. Directory names are not
// resolved here: see `folderIcon`.
export function fileIcon(name: string): FileIcon {
    const cached = cache.get(name);
    if (cached !== undefined) {
        return cached;
    }
    // `.dist` marks a shipped template (`phpunit.xml.dist`,
    // `package.json.dist`): the wrapped name picks the icon, which the
    // extension rules would otherwise miss.
    const iconName = name.toLowerCase().endsWith('.dist')
        ? name.slice(0, -'.dist'.length)
        : name;
    let icon: FileIcon | undefined;
    for (const rule of matchers) {
        if (rule.pattern.test(iconName)) {
            icon = describe(rule.icon, rule.color);
            break;
        }
    }
    icon ??= FALLBACK;
    cache.set(name, icon);
    return icon;
}
