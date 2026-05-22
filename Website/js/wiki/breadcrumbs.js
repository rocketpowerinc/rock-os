import { escapeHtml } from './utils.js';

export function renderBreadcrumbs(path) {

    const parts =
        path.split('/');

    const crumbs =
        parts.map((part, index) => {

            const isFile =
                index === parts.length - 1;

            const label =
                isFile
                ? part.replace(/\.md$/i, '')
                : part;

            if (isFile) {
                return `
                    <span class="wiki-breadcrumb-current">${escapeHtml(label)}</span>
                `;
            }

            return `
                <span>${escapeHtml(label)}</span>
            `;
        })
        .join('<span class="wiki-breadcrumb-separator">/</span>');

    return `
        <nav class="wiki-breadcrumbs" aria-label="Breadcrumb">
            ${crumbs}
        </nav>
    `;
}

