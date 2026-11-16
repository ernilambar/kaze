// Interpreter names mapped to the Monaco language that highlights them. Only
// interpreters Monaco bundles are listed; an unknown one falls back to plain
// text, which is better than a language id with no tokenizer.
const INTERPRETER_LANGUAGES: Record<string, string> = {
    sh: 'shell',
    bash: 'shell',
    zsh: 'shell',
    ksh: 'shell',
    dash: 'shell',
    ash: 'shell',
    fish: 'shell',
    csh: 'shell',
    tcsh: 'shell',
    python: 'python',
    python2: 'python',
    python3: 'python',
    pypy: 'python',
    pypy3: 'python',
    node: 'javascript',
    nodejs: 'javascript',
    bun: 'javascript',
    deno: 'javascript',
    qjs: 'javascript',
    ruby: 'ruby',
    jruby: 'ruby',
    macruby: 'ruby',
    perl: 'perl',
    perl5: 'perl',
    perl6: 'perl',
    raku: 'perl',
    php: 'php',
    php5: 'php',
    php7: 'php',
    php8: 'php',
    lua: 'lua',
    luajit: 'lua',
    tclsh: 'tcl',
    wish: 'tcl',
    expect: 'tcl',
    rscript: 'r',
    pwsh: 'powershell',
    powershell: 'powershell',
    elixir: 'elixir',
    escript: 'elixir',
    clojure: 'clojure',
    bb: 'clojure',
};

// executable is the program name a shebang token names. A path is reduced to
// its last component so /usr/bin/python3 and C:\Python\python.exe agree.
function executable(token: string): string {
    const base = token.split(/[\\/]/).pop() ?? token;
    return base.replace(/\.exe$/i, '').toLowerCase();
}

// shebangInterpreter reads the program out of a #! line. `env` is followed by
// optional -S/--split-string flags and NAME=value assignments before the real
// program, so those are skipped. An empty string means the line is not a
// shebang with a program.
export function shebangInterpreter(line: string): string {
    const trimmed = line.trimStart();
    if (!trimmed.startsWith('#!')) {
        return '';
    }
    const tokens = trimmed.slice(2).trim().split(/\s+/).filter((token) => token !== '');
    if (tokens.length === 0) {
        return '';
    }
    let program = executable(tokens[0]);
    if (program !== 'env') {
        return program;
    }
    for (let index = 1; index < tokens.length; index++) {
        const token = tokens[index];
        // `env -S python3 -u` and `env FOO=bar python3` both put arguments
        // before the command.
        if (token.startsWith('-') || /^[A-Za-z_][A-Za-z0-9_]*=/.test(token)) {
            continue;
        }
        program = executable(token);
        break;
    }
    return program === 'env' ? '' : program;
}

// languageForProgram maps an interpreter to a language, accepting a versioned
// name such as python3.12, ruby3.2 or php8.2 by stripping the trailing digits.
function languageForProgram(program: string): string | undefined {
    const direct = INTERPRETER_LANGUAGES[program];
    if (direct !== undefined) {
        return direct;
    }
    return INTERPRETER_LANGUAGES[program.replace(/[.\d]+$/, '')];
}

// shebangLanguage returns the Monaco language id a file's first line asks for,
// or undefined when the first line is not a shebang or names no bundled
// language. Only the first line is inspected: a later #! is a comment.
export function shebangLanguage(text: string): string | undefined {
    const newline = text.indexOf('\n');
    const firstLine = newline === -1 ? text : text.slice(0, newline);
    if (!firstLine.startsWith('#!')) {
        return undefined;
    }
    return languageForProgram(shebangInterpreter(firstLine));
}
