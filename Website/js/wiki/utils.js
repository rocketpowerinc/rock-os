export function fileTitle(path) {

    const parts =
        path.split('/');

    return parts[parts.length - 1]
        .replace(/\.md$/i, '');
}

export function escapeHtml(value) {

    return value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

export function getSnippet(text, query) {

    const normalizedQuery =
        query.toLowerCase();

    const line =
        text
            .split(/\r?\n/)
            .find(item =>
                item.toLowerCase()
                    .includes(normalizedQuery)
            );

    if (!line) {
        return '';
    }

    const trimmed =
        line.trim();

    if (trimmed.length <= 120) {
        return trimmed;
    }

    const matchIndex =
        trimmed.toLowerCase()
            .indexOf(normalizedQuery);

    const start =
        Math.max(0, matchIndex - 45);

    return `${start > 0 ? '...' : ''}${trimmed.slice(
        start,
        start + 120
    )}...`;
}

export async function copyText(text) {

    if (navigator.clipboard && window.isSecureContext) {

        await navigator.clipboard.writeText(text);
        return;
    }

    const textarea =
        document.createElement('textarea');

    textarea.value = text;
    textarea.setAttribute('readonly', '');
    textarea.style.position = 'fixed';
    textarea.style.left = '-9999px';

    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    textarea.remove();
}

export function normalizeDocPath(path) {

    const segments = [];

    decodeURIComponent(path)
        .replace(/\\/g, '/')
        .replace(/^\/+/, '')
        .split('/')
        .forEach(segment => {

            if (!segment || segment === '.') {
                return;
            }

            if (segment === '..') {
                segments.pop();
                return;
            }

            segments.push(segment);
        });

    return segments.join('/');
}

export function formatEditedDate(value) {

    if (!value) {
        return '';
    }

    const date =
        new Date(value);

    if (Number.isNaN(date.getTime())) {
        return '';
    }

    return date.toLocaleString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: 'numeric',
        minute: '2-digit'
    });
}

