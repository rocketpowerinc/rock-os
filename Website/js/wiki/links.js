import { normalizeDocPath } from './utils.js';

export function resolveMarkdownLink(
    href,
    currentDocPath
) {

    if (!href || href.startsWith('#')) {
        return '';
    }

    if (/^[a-z][a-z0-9+.-]*:/i.test(href)) {
        return '';
    }

    const pathOnly =
        href.split('#')[0]
            .split('?')[0];

    if (!pathOnly.toLowerCase().endsWith('.md')) {
        return '';
    }

    if (pathOnly.startsWith('/')) {
        return normalizeDocPath(pathOnly);
    }

    if (pathOnly.startsWith('tabs/wiki/')) {
        return normalizeDocPath(pathOnly);
    }

    if (pathOnly.startsWith('wiki/')) {
        return normalizeDocPath(`tabs/${pathOnly}`);
    }

    const currentFolder =
        currentDocPath
            .split('/')
            .slice(0, -1)
            .join('/');

    return normalizeDocPath(
        `${currentFolder}/${pathOnly}`
    );
}

export function wikiDocHref(path) {

    let targetPage = 'wiki.html';
    if (path.startsWith('tabs/rocket/')) {
        targetPage = 'rocket.html';
    } else if (path.startsWith('tabs/bootstraps/')) {
        targetPage = 'bootstraps.html';
    }

    const url =
        new URL(targetPage, window.location.href);

    url.searchParams.set('doc', path);

    return `${url.pathname}${url.search}`;
}


export function enhanceWikiLinks(
    container,
    currentDocPath,
    { allMarkdownFiles, loadDoc }
) {

    container.querySelectorAll('a[href]')
        .forEach(link => {

            const docPath =
                resolveMarkdownLink(
                    link.getAttribute('href'),
                    currentDocPath
                );

            if (!docPath) {
                return;
            }

            const docExists =
                allMarkdownFiles.includes(docPath);

            link.href = wikiDocHref(docPath);
            link.dataset.path = docPath;

            if (!docExists) {

                link.classList.add('broken-wiki-link');
                link.title = `Missing wiki page: ${docPath}`;
                link.setAttribute(
                    'aria-label',
                    `${link.textContent.trim()} missing wiki page`
                );
            }

            link.onclick = event => {

                event.preventDefault();

                if (!docExists) {
                    return;
                }

                loadDoc(docPath);
            };
        });
}

export function enhanceExternalLinks(container) {

    container.querySelectorAll('a[href]')
        .forEach(link => {

            const rawHref =
                link.getAttribute('href');

            if (!rawHref) {
                return;
            }

            let url;

            try {
                url =
                    new URL(rawHref, window.location.href);
            }
            catch {
                return;
            }

            if (
                (url.protocol !== 'http:' && url.protocol !== 'https:') ||
                url.host === window.location.host
            ) {
                return;
            }

            link.target =
                '_blank';
            link.rel =
                'noopener noreferrer';
        });
}

export function markdownLinksInText(
    text,
    sourceDocPath
) {

    const links = new Set();
    const inlineLinkPattern =
        /(!)?\[[^\]]*]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;

    let match;

    while ((match = inlineLinkPattern.exec(text)) !== null) {

        if (match[1]) {
            continue;
        }

        const docPath =
            resolveMarkdownLink(
                match[2],
                sourceDocPath
            );

        if (docPath) {
            links.add(docPath);
        }
    }

    return links;
}

