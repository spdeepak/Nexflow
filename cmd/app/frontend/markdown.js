// --- Minimal, dependency-free markdown renderer ---
// Content is escaped at the leaf level (when it becomes HTML text/attributes),
// so raw HTML from model output can never inject markup. Block syntax is parsed
// on the raw markdown, then each leaf is escaped while inline styles are applied.

function markdownToHtml(src) {
    if (!src) return '';
    src = String(src).replace(/\r\n/g, '\n');

    // Pull out fenced code blocks so their contents are not processed as markdown.
    const blocks = [];
    src = src.replace(/^```[^\n]*\n?([\s\S]*?)```/gm, (_m, code) => {
        const idx = blocks.length;
        blocks.push('<pre><code>' + escapeHtml(code.replace(/\n$/, '')) + '</code></pre>');
        return '<md-code-block-' + idx + '>';
    });

    return renderBlocks(src, blocks);
}

function renderBlocks(src, blocks) {
    const lines = src.split('\n');
    const out = [];
    let i = 0;
    let para = [];

    const flushPara = () => {
        if (para.length) {
            out.push('<p>' + inlineMarkdown(para.join('\n')) + '</p>');
            para = [];
        }
    };
    const restoreCodeBlocks = (html) => {
        return html.replace(/<md-code-block-(\d+)>/g, (_m, idx) => blocks[+idx]);
    };

    while (i < lines.length) {
        const trimmed = lines[i].trim();

        if (trimmed === '') {
            flushPara();
            i++;
            continue;
        }

        const codeSentinel = /^<md-code-block-(\d+)>$/.exec(trimmed);
        if (codeSentinel) {
            flushPara();
            out.push(blocks[+codeSentinel[1]]);
            i++;
            continue;
        }

        if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
            flushPara();
            out.push('<hr>');
            i++;
            continue;
        }

        const heading = /^(#{1,6})\s+(.*)$/.exec(trimmed);
        if (heading) {
            flushPara();
            const level = heading[1].length;
            out.push('<h' + level + '>' + inlineMarkdown(heading[2]) + '</h' + level + '>');
            i++;
            continue;
        }

        if (/^>\s?/.test(trimmed)) {
            flushPara();
            const quoted = [];
            while (i < lines.length && /^>\s?/.test(lines[i].trim())) {
                quoted.push(lines[i].trim().replace(/^>\s?/, ''));
                i++;
            }
            out.push('<blockquote>' + inlineMarkdown(quoted.join(' ')) + '</blockquote>');
            continue;
        }

        const ulMatch = /^[-*+]\s+(.*)$/.exec(trimmed);
        if (ulMatch) {
            flushPara();
            out.push('<ul>');
            while (i < lines.length) {
                const li = /^[-*+]\s+(.*)$/.exec(lines[i].trim());
                if (!li) break;
                out.push('<li>' + inlineMarkdown(li[1]) + '</li>');
                i++;
            }
            out.push('</ul>');
            continue;
        }

        const olMatch = /^\d+[.)]\s+(.*)$/.exec(trimmed);
        if (olMatch) {
            flushPara();
            out.push('<ol>');
            while (i < lines.length) {
                const li = /^\d+[.)]\s+(.*)$/.exec(lines[i].trim());
                if (!li) break;
                out.push('<li>' + inlineMarkdown(li[1]) + '</li>');
                i++;
            }
            out.push('</ol>');
            continue;
        }

        if (trimmed.startsWith('|') && isTableSeparator(lines[i + 1] || '')) {
            const parseRow = (row) => row.trim().replace(/^\||\|$/g, '').split('|').map((c) => c.trim());
            const header = parseRow(lines[i]);
            const rows = [];
            let j = i + 2;
            while (j < lines.length && lines[j].trim().startsWith('|')) {
                rows.push(parseRow(lines[j]));
                j++;
            }
            flushPara();
            let table = '<table><thead><tr>';
            for (const cell of header) table += '<th>' + inlineMarkdown(cell) + '</th>';
            table += '</tr></thead><tbody>';
            for (const row of rows) {
                table += '<tr>';
                for (const cell of row) table += '<td>' + inlineMarkdown(cell) + '</td>';
                table += '</tr>';
            }
            table += '</tbody></table>';
            out.push(table);
            i = j;
            continue;
        }

        para.push(trimmed);
        i++;
    }
    flushPara();

    return restoreCodeBlocks(out.join('\n'));
}

function isTableSeparator(line) {
    return /^\|?[\s:|-]+\|?\s*$/.test(line) && line.includes('-');
}

function inlineMarkdown(text) {
    // Protect HTML we emit so it survives escaping below.
    const segs = [];
    const stash = (html) => {
        const idx = segs.length;
        segs.push(html);
        return '«MD' + idx + '»';
    };

    // Inline code (escaped at extraction).
    text = text.replace(/`([^`\n]+)`/g, (_m, code) => stash('<code>' + escapeHtml(code) + '</code>'));

    // Images (attributes escaped).
    text = text.replace(
        /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g,
        (_m, alt, url) => stash('<img src="' + escapeAttr(url) + '" alt="' + escapeAttr(alt) + '">')
    );

    // Links (href + label escaped).
    text = text.replace(
        /\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g,
        (_m, label, url) => stash('<a href="' + escapeAttr(url) + '" target="_blank" rel="noopener">' + escapeHtml(label) + '</a>')
    );

    // Escape any remaining raw HTML before applying inline styles.
    text = escapeHtml(text);

    // Bold + italic.
    text = text.replace(/\*\*\*([^*]+)\*\*\*/g, '<strong><em>$1</em></strong>');
    text = text.replace(/___([^_]+)___/g, '<strong><em>$1</em></strong>');
    // Bold.
    text = text.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
    text = text.replace(/__([^_]+)__/g, '<strong>$1</strong>');
    // Italic.
    text = text.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>');
    text = text.replace(/(^|[^_])\_([^_\n]+)\_/g, '$1<em>$2</em>');
    // Strikethrough.
    text = text.replace(/~~([^~]+)~~/g, '<del>$1</del>');

    // Restore protected segments.
    text = text.replace(/«MD(\d+)»/g, (_m, idx) => segs[+idx] || '');
    return text;
}

function escapeAttr(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/"/g, '&quot;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}