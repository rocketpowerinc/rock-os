import { normalizeDocPath } from './utils.js';

const tabIndexCache = new Map();
const workspaceSectionNames =
    new Set(['wiki', 'bootstraps', 'cheatsheets', 'dotfiles', 'bookmarks']);

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
            if (file && typeof file === 'object' && typeof file.path === 'string') {
                return file.path.trim();
            }
            return '';
        })
        .filter(path => path.toLowerCase().endsWith('.md'));
}

function encodePathSegments(path) {
    return String(path || '')
        .split('/')
        .filter(Boolean)
        .map(part => encodeURIComponent(part))
        .join('/');
}

function profileWorkspacePathInfo(path) {
    const parts =
        String(path || '').split('/');

    if (
        parts.length >= 6 &&
        parts[0] === 'ENCRYPTED' &&
        parts[1] === 'dashboards' &&
        parts[2] === 'Profiles' &&
        workspaceSectionNames.has(parts[4])
    ) {
        return {
            profile: parts[3],
            section: parts[4]
        };
    }

    return null;
}

function workspaceTab(profile, section) {
    return `workspace|${profile}|${section}`;
}

function dashboardTab(dashboard) {
    return `dashboard|${dashboard}`;
}

async function loadTabIndex(tab) {
    let indexUrl = '';
    const parts =
        String(tab || '').split('|');

    if (parts[0] === 'workspace' && parts.length === 3) {
        indexUrl =
            `/${parts[2]}-index.json?profile=${encodeURIComponent(parts[1])}`;
    } else if (parts[0] === 'dashboard' && parts.length === 2) {
        indexUrl =
            `/dashboards-index.json?profile=${encodeURIComponent(parts[1])}`;
    }

    if (!indexUrl) {
        return [];
    }

    if (!tabIndexCache.has(tab)) {
        const separator =
            indexUrl.includes('?') ? '&' : '?';

        tabIndexCache.set(
            tab,
            fetch(`${indexUrl}${separator}nocache=${Date.now()}`)
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

export function resolveMarkdownLink(href, currentDocPath) {
    if (!href || href.startsWith('#') || /^[a-z][a-z0-9+.-]*:/i.test(href)) {
        return '';
    }

    const pathOnly =
        href.split('#')[0]
            .split('?')[0];

    if (!pathOnly.toLowerCase().endsWith('.md')) {
        return '';
    }

    if (pathOnly.startsWith('/') || /^ENCRYPTED\/dashboards\//.test(pathOnly)) {
        return normalizeDocPath(pathOnly);
    }

    const currentWorkspace =
        profileWorkspacePathInfo(currentDocPath);
    const sectionMatch =
        pathOnly.match(/^(wiki|bootstraps|cheatsheets|dotfiles|bookmarks)\/(.+)$/);

    if (currentWorkspace && sectionMatch) {
        return normalizeDocPath(
            `ENCRYPTED/dashboards/Profiles/${currentWorkspace.profile}/${sectionMatch[1]}/${sectionMatch[2]}`
        );
    }

    const currentFolder =
        currentDocPath
            .split('/')
            .slice(0, -1)
            .join('/');

    return normalizeDocPath(`${currentFolder}/${pathOnly}`);
}

export function wikiDocHref(path) {
    const workspace =
        profileWorkspacePathInfo(path);
    let targetPage =
        '/dashboards.html';

    if (workspace) {
        targetPage =
            `/${workspace.section}.html`;
    } else if (path.startsWith('ENCRYPTED/dashboards/')) {
        const parts =
            path.split('/');
        const dashboard =
            parts.length > 3 ? `${parts[2]}/${parts[3]}` : '';

        targetPage =
            dashboard
                ? `/ENCRYPTED/dashboards/${encodePathSegments(dashboard)}/`
                : '/dashboards.html';
    }

    const url =
        new URL(targetPage, window.location.origin);

    if (workspace) {
        url.searchParams.set('profile', workspace.profile);
    }
    url.searchParams.set('doc', path);

    return `${url.pathname}${url.search}`;
}

function getTabForPath(path) {
    const workspace =
        profileWorkspacePathInfo(path);
    if (workspace) {
        return workspaceTab(workspace.profile, workspace.section);
    }

    if (path.startsWith('ENCRYPTED/dashboards/')) {
        const parts =
            path.split('/');
        const dashboard =
            parts.length > 3 ? `${parts[2]}/${parts[3]}` : '';
        return dashboardTab(dashboard);
    }

    return '';
}

function getCurrentTab() {
    const rawPath =
        window.location.pathname;
    const path =
        rawPath.toLowerCase();
    const params =
        new URLSearchParams(window.location.search);
    const profile =
        params.get('profile') || '';

    for (const section of workspaceSectionNames) {
        if (path.endsWith(`/${section}.html`) && profile) {
            return workspaceTab(profile, section);
        }
    }

    if (path.includes('dashboards.html') || path.endsWith('/dashboards')) {
        const dashboard =
            params.get('profile') || params.get('dashboard') || '';
        return dashboardTab(dashboard);
    }

    if (path.includes('/dashboards/')) {
        const parts =
            rawPath.split('/').filter(Boolean);
        const rootIndex =
            parts.findIndex(part => part.toLowerCase() === 'dashboards');

        if (rootIndex >= 0) {
            const dashboardParts =
                parts
                    .slice(rootIndex + 1)
                    .filter(part => part && part.toLowerCase() !== 'index.html')
                    .slice(0, 2)
                    .map(part => decodeURIComponent(part));

            if (dashboardParts.length === 2) {
                return dashboardTab(dashboardParts.join('/'));
            }
        }
    }

    return '';
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

            link.href =
                wikiDocHref(docPath);
            link.dataset.path =
                docPath;

            const isCrossTab =
                getTabForPath(docPath) !== getCurrentTab();
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
                if (docExists) {
                    loadDoc(docPath);
                }
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

export function markdownLinksInText(text, sourceDocPath) {
    const links =
        new Set();
    const inlineLinkPattern =
        /(!)?\[[^\]]*]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;

    let match;
    while ((match = inlineLinkPattern.exec(text)) !== null) {
        if (match[1]) {
            continue;
        }

        const docPath =
            resolveMarkdownLink(match[2], sourceDocPath);
        if (docPath) {
            links.add(docPath);
        }
    }

    return links;
}
