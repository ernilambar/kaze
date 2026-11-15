import { describe, expect, it } from 'vitest';
import { renderMarkdown } from './markdown';

describe('renderMarkdown', () => {
    it('renders headings and paragraphs', () => {
        expect(renderMarkdown('# Title\n\nBody text')).toBe(
            '<h1>Title</h1><p>Body text</p>',
        );
    });

    it('escapes raw HTML instead of passing it through', () => {
        expect(renderMarkdown('<b>hi</b>')).toBe('<p>&lt;b&gt;hi&lt;/b&gt;</p>');
    });

    it('renders inline code, emphasis, strong and strikethrough', () => {
        expect(renderMarkdown('**bold** *em* `code` ~~gone~~')).toBe(
            '<p><strong>bold</strong> <em>em</em> <code>code</code> <del>gone</del></p>',
        );
    });

    it('keeps underscores inside a word literal', () => {
        expect(renderMarkdown('snake_case_name')).toBe('<p>snake_case_name</p>');
    });

    it('links safe URLs and drops script schemes', () => {
        expect(renderMarkdown('[Docs](https://example.com)')).toBe(
            '<p><a href="https://example.com">Docs</a></p>',
        );
        expect(renderMarkdown('[bad](javascript:prompt)')).toBe('<p>bad</p>');
    });

    it('renders a fenced code block without interpreting its content', () => {
        expect(renderMarkdown('```go\nfmt.Println("<hi>")\n```')).toBe(
            '<pre><code class="language-go">fmt.Println(&quot;&lt;hi&gt;&quot;)</code></pre>',
        );
    });

    it('renders unordered, ordered and task lists', () => {
        expect(renderMarkdown('- one\n- two')).toBe('<ul><li>one</li><li>two</li></ul>');
        expect(renderMarkdown('1. one\n2. two')).toBe('<ol><li>one</li><li>two</li></ol>');
        expect(renderMarkdown('- [x] done')).toBe(
            '<ul><li><input type="checkbox" disabled checked> done</li></ul>',
        );
    });

    it('renders blockquotes and rules', () => {
        expect(renderMarkdown('> quoted')).toBe('<blockquote><p>quoted</p></blockquote>');
        expect(renderMarkdown('---')).toBe('<hr>');
    });

    it('renders a table', () => {
        const source = '| a | b |\n| --- | --- |\n| 1 | 2 |';

        expect(renderMarkdown(source)).toBe(
            '<table><thead><tr><th>a</th><th>b</th></tr></thead>' +
                '<tbody><tr><td>1</td><td>2</td></tr></tbody></table>',
        );
    });
});
