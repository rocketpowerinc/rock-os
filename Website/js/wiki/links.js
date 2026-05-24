import { normalizeDocPath } from './utils.js';

const tabIndexCache = new Map();

const tabIndexes = {
    wiki: 'wiki-index.json',
    guides: 'guides-index.json',
    cheatsheets: 'cheatsheets-index.json',
    dotfiles: 'dotfiles-index.json',
    bookmarks: 'bookmarks-index.json',
    rocket: 'rocket-index.json'
};

function normalizeIndexFiles(payload) {

    const files =
        Array.isArray(payload)
        ? payload
        : [payload];

    return files
        .map(file => {

            if (typeof file === 'string') {
                return file.trim();
            }

            if (
                file &&
                typeof file === 'object' &&
                typeof file.path === 'string'
            ) {
                return file.path.trim();
            }

            return '';
        })
        .filter(path => path.toLowerCase().endsWith('.md'));
}

async function loadTabIndex(tab) {

    const indexUrl =
        tabIndexes[tab];

    if (!indexUrl) {
        return [];
    }

    if (!tabIndexCache.has(tab)) {

        tabIndexCache.set(
            tab,
            fetch(`${indexUrl}?nocache=${Date.now()}`)
                .then(response => response.ok ? response.json() : [])
                .then(normalizeIndexFiles)
                .catch(() => [])
        );
    }

    return tabIndexCache.get(tab);
}

function markBrokenLink(link, docPath) {

    link.classList.add('broken-wiki-link');
    link.title = `Missing wiki page: ${docPath}`;
    link.setAttribute(
        'aria-label',
        `${link.textContent.trim()} missing wiki page`
    );
}

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

    if (/^menu\/(wiki|guides|cheatsheets|dotfiles|bookmarks|rocket)\//.test(pathOnly)) {
        return normalizeDocPath(pathOnly);
    }

    if (/^(wiki|guides|cheatsheets|dotfiles|bookmarks|rocket)\//.test(pathOnly)) {
        return normalizeDocPath(`menu/${pathOnly}`);
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
    if (path.startsWith('menu/rocket/')) {
        targetPage = 'rocket.html';
    } else if (path.startsWith('menu/guides/')) {
        targetPage = 'guides.html';
    } else if (path.startsWith('menu/cheatsheets/')) {
        targetPage = 'cheatsheets.html';
    } else if (path.startsWith('menu/dotfiles/')) {
        targetPage = 'dotfiles.html';
    } else if (path.startsWith('menu/bookmarks/')) {
        targetPage = 'bookmarks.html';
    }

    const url =
        new URL(targetPage, window.location.href);

    url.searchParams.set('doc', path);

    return `${url.pathname}${url.search}`;
}


function getTabForPath(path) {
    if (path.startsWith('menu/wiki/')) return 'wiki';
    if (path.startsWith('menu/rocket/')) return 'rocket';
    if (path.startsWith('menu/guides/')) return 'guides';
    if (path.startsWith('menu/cheatsheets/')) return 'cheatsheets';
    if (path.startsWith('menu/dotfiles/')) return 'dotfiles';
    if (path.startsWith('menu/bookmarks/')) return 'bookmarks';
    return '';
}

function getCurrentTab() {
    const path = window.location.pathname.toLowerCase();
    if (path.includes('wiki.html') || path.endsWith('/wiki')) return 'wiki';
    if (path.includes('rocket.html') || path.endsWith('/rocket')) return 'rocket';
    if (path.includes('guides.html') || path.endsWith('/guides')) return 'guides';
    if (path.includes('cheatsheets.html') || path.endsWith('/cheatsheets')) return 'cheatsheets';
    if (path.includes('dotfiles.html') || path.endsWith('/dotfiles')) return 'dotfiles';
    if (path.includes('bookmarks.html') || path.endsWith('/bookmarks')) return 'bookmarks';
    return 'wiki';
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

            const isCrossTab = getTabForPath(docPath) !== getCurrentTab();
            if (isCrossTab) {
                loadTabIndex(getTabForPath(docPath))
                    .then(files => {

                        if (!files.includes(docPath)) {
                            markBrokenLink(link, docPath);
                            link.onclick = event => {
                                event.preventDefault();
                            };
                        }
                    });

                return;
            }

            if (!docExists) {

                markBrokenLink(link, docPath);
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

