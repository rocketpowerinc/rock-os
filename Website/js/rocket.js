import { renderBreadcrumbs } from './wiki/breadcrumbs.js';
import { enhanceCallouts } from './wiki/callouts.js';
import { enhanceCodeBlocks, enhanceInlineCode } from './wiki/code-blocks.js';
import { enhanceExternalLinks, enhanceWikiLinks, markdownLinksInText, wikiDocHref } from './wiki/links.js';
import { buildTableOfContents, clearToc, scrollToCurrentHash } from './wiki/toc.js';
import { escapeHtml, fileTitle, formatEditedDate } from './wiki/utils.js';

const expandedFolders = new Set();
const stateStorageKey = 'rock-os-rocket-state';
const pinnedDocsStorageKey = 'rock-os-rocket-pinned-docs';

let lastIndexText = '';
let activeDocPath = '';
let indexLoadInProgress = false;
let allMarkdownFiles = [];
let pinnedDocPaths = new Set();
let searchQuery = '';
let searchResults = [];
let searchLoading = false;
let searchDebounceTimer = null;
let searchRequestId = 0;
let initialUrlDocLoaded = false;

loadSavedState();
loadPinnedDocs();

function getSidebar() {

    return document.getElementById(
        'sidebarListContainer'
    );
}

function getSearchStatus() {

    return document.getElementById(
        'rocketSearchStatus'
    );
}

function normalizeFiles(parsed) {

    const files =
        Array.isArray(parsed)
        ? parsed
        : [parsed];

    return files.map(file => {

        if (typeof file === 'string') {
            return file;
        }

        if (file && typeof file === 'object' && typeof file.path === 'string') {
            return file.path;
        }

        return '';
    }).filter(path =>
        path !== ''
    );
}

function isDocPinned(path) {

    return pinnedDocPaths.has(path);
}

function toggleDocPin(path) {

    if (pinnedDocPaths.has(path)) {
        pinnedDocPaths.delete(path);
    } else {
        pinnedDocPaths.add(path);
    }

    savePinnedDocs();
    rerenderSidebar();
}

function savePinnedDocs() {

    try {
        localStorage.setItem(
            pinnedDocsStorageKey,
            JSON.stringify(
                Array.from(pinnedDocPaths)
            )
        );
    }
    catch (err) {
        console.error(
            'Error saving pinned docs:',
            err
        );
    }
}

function loadPinnedDocs() {

    try {
        const saved =
            localStorage.getItem(
                pinnedDocsStorageKey
            );

        if (saved) {

            const parsed =
                JSON.parse(saved);

            pinnedDocPaths =
                new Set(
                    normalizeFiles(parsed)
                );
        }
    }
    catch (err) {
        console.error(
            'Error loading pinned docs:',
            err
        );
    }
}

function saveState() {

    try {

        const state = {
            expanded: Array.from(expandedFolders),
            active: activeDocPath
        };

        localStorage.setItem(
            stateStorageKey,
            JSON.stringify(state)
        );
    }
    catch (err) {
        console.error(
            'Error saving sidebar state:',
            err
        );
    }
}

function loadSavedState() {

    try {
        const saved =
            localStorage.getItem(
                stateStorageKey
            );

        if (saved) {

            const state =
                JSON.parse(saved);

            if (Array.isArray(state.expanded)) {

                state.expanded.forEach(path =>
                    expandedFolders.add(path)
                );
            }

            if (typeof state.active === 'string') {
                activeDocPath = state.active;
            }
        }
    }
    catch (err) {
        console.error(
            'Error loading sidebar state:',
            err
        );
    }
}

function renderEmptyState(container) {

    container.innerHTML = `
        <div class="empty-state">
            <p>No rocket files found.</p>
        </div>
    `;
}

function pinnedMarkdownFiles() {

    return allMarkdownFiles.filter(file =>
        isDocPinned(file)
    );
}

function createSidebarDocLink(path, label, className = 'doc-link') {

    const link =
        document.createElement('a');

    link.className = className;
    link.href = '#';
    link.dataset.path = path;

    if (path === activeDocPath) {
        link.classList.add('active');
    }

    const labelText =
        document.createElement('span');
    labelText.className =
        'doc-link-label';
    labelText.innerText =
        label;

    const pinButton =
        document.createElement('span');
    pinButton.className =
        'pin-toggle';
    pinButton.classList.toggle(
        'active',
        isDocPinned(path)
    );
    pinButton.setAttribute(
        'role',
        'button'
    );
    pinButton.setAttribute(
        'aria-label',
        isDocPinned(path) ? 'Unpin doc' : 'Pin doc'
    );
    pinButton.title =
        isDocPinned(path) ? 'Unpin doc' : 'Pin doc';
    pinButton.innerText =
        '★';

    pinButton.onclick = event => {

        event.preventDefault();
        event.stopPropagation();
        toggleDocPin(path);
    };
    pinButton.onkeydown = event => {

        if (event.key !== 'Enter' && event.key !== ' ') {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        toggleDocPin(path);
    };
    pinButton.tabIndex =
        0;

    link.append(labelText, pinButton);

    link.onclick = () => {

        loadDoc(path);
        const searchInput = document.getElementById('rocketSearchInput');
        if (searchInput) {
            searchInput.value = '';
            searchQuery = '';
            searchResults = [];
            rerenderSidebar();
        }
    };

    return link;
}

function renderWikiError(
    title,
    message,
    details = []
) {

    const content =
        document.getElementById('content');

    if (!content) {
        return;
    }

    const detailItems =
        details.map(d =>
            `<li>${escapeHtml(d)}</li>`
        ).join('');

    content.innerHTML = `
        <div class="wiki-error-panel">
            <h1>${escapeHtml(title)}</h1>
            <p>${escapeHtml(message)}</p>
            ${detailItems ? `<ul>${detailItems}</ul>` : ''}
            <pre><code>cd cmd/rock-os-wiki
go run .</code></pre>
        </div>
    `;
}

function isDirectFileOpen() {

    return window.location.protocol === 'file:';
}

function folderPathsForDoc(path) {

    const parts =
        path
            .replace(/^menu\/rocket\//, '')
            .split('/');

    parts.pop();

    return parts.map((_, index) =>
        parts.slice(0, index + 1).join('/')
    );
}

function allFolderPaths(files) {

    const paths =
        new Set();

    files.forEach(file => {

        const parts =
            file
                .replace(/^menu\/rocket\//, '')
                .split('/');

        parts.pop();

        parts.forEach((_, index) => {
            paths.add(
                parts.slice(0, index + 1).join('/')
            );
        });
    });

    return Array.from(paths);
}

function rerenderSidebar() {

    const sidebar =
        getSidebar();

    if (!sidebar) {
        return;
    }

    sidebar.innerHTML = '';

    if (!allMarkdownFiles.length) {
        renderEmptyState(sidebar);
        return;
    }

    if (searchQuery) {
        renderSearchResults();
        return;
    }

    renderNormalSidebar(sidebar);

    updateActiveDocLinks();
    updateToggleAllFoldersButton();
}

function renderNormalSidebar(sidebar) {

    const pinned =
        pinnedMarkdownFiles();

    if (pinned.length > 0) {

        const pinnedHeader =
            document.createElement('div');

        pinnedHeader.className =
            'sidebar-section-header';
        pinnedHeader.innerText =
            'Pinned';

        const pinnedList =
            document.createElement('div');

        pinnedList.className =
            'pinned-list';

        pinned.forEach(path => {

            pinnedList.appendChild(
                createSidebarDocLink(
                    path,
                    fileTitle(path),
                    'doc-link pinned-doc-link'
                )
            );
        });

        sidebar.appendChild(pinnedHeader);
        sidebar.appendChild(pinnedList);
    }

    const tree =
        buildTree(
            allMarkdownFiles
        );

    renderTree(
        tree,
        sidebar
    );
}

function clearExpandedFolders() {

    expandedFolders.clear();
    saveState();
    rerenderSidebar();
}

function setAllFoldersExpanded(expanded) {

    expandedFolders.clear();

    if (expanded) {

        allFolderPaths(allMarkdownFiles)
            .forEach(folderPath =>
                expandedFolders.add(folderPath)
            );
    }

    saveState();
    rerenderSidebar();
    updateToggleAllFoldersButton();
}

function areAllFoldersExpanded() {

    const folderPaths =
        allFolderPaths(allMarkdownFiles);

    return folderPaths.length > 0 &&
        folderPaths.every(folderPath =>
            expandedFolders.has(folderPath)
        );
}

function updateToggleAllFoldersButton() {

    const toggleAllFoldersBtn =
        document.getElementById('toggleAllFoldersBtn');

    if (!toggleAllFoldersBtn) {
        return;
    }

    const allExpanded =
        areAllFoldersExpanded();

    toggleAllFoldersBtn.innerHTML =
        allExpanded ? '&#x229F;' : '&#x229E;';
    toggleAllFoldersBtn.setAttribute(
        'aria-label',
        allExpanded ? 'Fold all folders' : 'Expand all folders'
    );
    toggleAllFoldersBtn.title =
        allExpanded ? 'Fold all folders' : 'Expand all folders';
}

function rememberActiveDoc(path) {

    activeDocPath = path;

    folderPathsForDoc(path)
        .forEach(folderPath =>
            expandedFolders.add(folderPath)
        );

    saveState();
}

function updateActiveDocLinks() {

    const sidebar =
        getSidebar();

    if (!sidebar) {
        return;
    }

    sidebar.querySelectorAll('.doc-link')
        .forEach(link => {

            link.classList.toggle(
                'active',
                link.dataset.path === activeDocPath
            );
        });
}

function syncExpandedFoldersFromDom() {

    const sidebar =
        getSidebar();

    if (!sidebar) {
        return;
    }

    sidebar.querySelectorAll('[data-folder-path]')
        .forEach(folder => {

            const children =
                Array.from(folder.children)
                    .find(child =>
                        child.classList.contains(
                            'folder-children'
                        )
                    );

            if (!children) {
                return;
            }

            const folderPath =
                folder.dataset.folderPath;

            if (children.style.display === 'none') {
                expandedFolders.delete(folderPath);
            } else {
                expandedFolders.add(folderPath);
            }
        });

    saveState();
}

async function findBacklinks(targetDocPath) {

    const backlinks = [];

    await Promise.all(
        allMarkdownFiles
            .filter(file =>
                file !== targetDocPath
            )
            .map(async file => {

                const text =
                    await loadMarkdownText(file);

                if (!text) {
                    return;
                }

                const links =
                    markdownLinksInText(
                        text,
                        file
                    );

                if (links.has(targetDocPath)) {
                    backlinks.push(file);
                }
            })
    );

    return backlinks.sort((a, b) =>
        a.toLowerCase()
            .localeCompare(b.toLowerCase())
    );
}

function renderBacklinks(
    container,
    backlinks
) {

    const existing =
        container.querySelector('.wiki-backlinks');

    if (existing) {
        existing.remove();
    }

    if (!backlinks.length) {
        return;
    }

    const section =
        document.createElement('section');

    section.className = 'wiki-backlinks';
    section.innerHTML = `
        <h2>Referenced by</h2>
        <div class="wiki-backlink-list"></div>
    `;

    const list =
        section.querySelector('.wiki-backlink-list');

    backlinks.forEach(path => {

        const link =
            document.createElement('a');

        link.className = 'wiki-backlink';
        link.href = wikiDocHref(path);
        link.dataset.path = path;
        link.innerHTML = `
            <span>${escapeHtml(fileTitle(path))}</span>
            <small>${escapeHtml(path)}</small>
        `;

        link.onclick = event => {

            event.preventDefault();
            loadDoc(path);
        };

        list.appendChild(link);
    });

    container.appendChild(section);
}

async function enhanceBacklinks(
    container,
    targetDocPath
) {

    const backlinks =
        await findBacklinks(targetDocPath);

    if (activeDocPath !== targetDocPath) {
        return;
    }

    renderBacklinks(
        container,
        backlinks
    );
}

async function loadDoc(path, options = {}) {

    const shouldUpdateUrl =
        options.updateUrl !== false;

    rememberActiveDoc(path);
    updateActiveDocLinks();

    if (shouldUpdateUrl) {
        updateDocUrl(path, options.replaceUrl === true);
    }

    const response = await fetch(
        `/api/rocket/doc?path=${encodeURIComponent(path)}&nocache=${Date.now()}`
    );

    if (!response.ok) {

        console.error(
            'Error loading document:',
            response.statusText
        );

        return;
    }

    const doc =
        await response.json();

    if (activeDocPath !== path) {
        return;
    }

    const content =
        document.getElementById('content');

    if (!content) {
        return;
    }

    clearToc();

    const lastEdited =
        formatEditedDate(doc.lastEdited);

    content.innerHTML =
        `${renderBreadcrumbs(path)}${lastEdited
            ? `<div class="doc-meta">Last edited ${escapeHtml(lastEdited)}</div>`
            : ''}${doc.html || ''}`;

    enhanceCodeBlocks(content);
    enhanceInlineCode(content);
    enhanceCallouts(content);
    enhanceWikiLinks(content, path, { allMarkdownFiles, loadDoc });
    enhanceExternalLinks(content);
    highlightDocText(content, searchQuery);
    buildTableOfContents(content);
    enhanceBacklinks(content, path);
    content.scrollTop = 0;
    scrollToCurrentHash();
}

function buildTree(files) {

    const tree = {};

    files.forEach(file => {

        const parts = file
            .replace('menu/rocket/', '')
            .split('/');

        let current = tree;

        parts.forEach((part, index) => {

            if (!current[part]) {

                current[part] =
                    index === parts.length - 1
                    ? {
                        type: 'file',
                        path: file
                    }
                    : {
                        type: 'folder',
                        children: {}
                    };
            }

            if (current[part].type === 'folder') {

                current = current[part].children;
            }
        });
    });

    return tree;
}

function folderIconName(folderPath) {

    const normalizedPath =
        folderPath.toLowerCase();

    const folderParts =
        folderPath
            .split('/')
            .map(part =>
                part.toLowerCase()
            );

    if (normalizedPath === 'private') {
        return 'lock';
    }

    if (normalizedPath === 'tech/universal') {
        return 'globe';
    }

    if (normalizedPath === 'tech') {
        return 'laptop';
    }

    if (folderParts.includes('windows')) {
        return 'windows';
    }

    if (folderParts.includes('mac')) {
        return 'mac';
    }

    if (folderParts.includes('linux')) {
        return 'linux';
    }

    if (folderParts.includes('android')) {
        return 'android';
    }

    if (folderParts.includes('ios')) {
        return 'ios';
    }

    return 'folder';
}

function iconElement(name) {

    const svg =
        document.createElementNS(
            'http://www.w3.org/2000/svg',
            'svg'
        );

    svg.setAttribute('viewBox', '0 0 24 24');
    svg.setAttribute('aria-hidden', 'true');
    svg.classList.add('tree-folder-icon');

    const addPath = (d) => {

        const path =
            document.createElementNS(
                'http://www.w3.org/2000/svg',
                'path'
            );

        path.setAttribute('d', d);
        path.setAttribute('fill', 'currentColor');
        svg.appendChild(path);
    };

    const addCircle = (cx, cy, r) => {

        const circle =
            document.createElementNS(
                'http://www.w3.org/2000/svg',
                'circle'
            );

        circle.setAttribute('cx', cx);
        circle.setAttribute('cy', cy);
        circle.setAttribute('r', r);
        circle.setAttribute('fill', 'currentColor');
        svg.appendChild(circle);
    };

    switch (name) {
    case 'windows':
        addPath('M3 4.5l7.8-1.1v8.1H3V4.5zM12.1 3.2L21 2v9.5h-8.9V3.2zM3 12.7h7.8v8.1L3 19.7v-7zM12.1 12.7H21V22l-8.9-1.2v-8.1z');
        break;
    case 'mac':
        addPath('M16.4 2.4c.1 1.4-.5 2.7-1.3 3.7-.9 1-2.2 1.7-3.5 1.6-.2-1.3.5-2.7 1.3-3.6.9-1 2.4-1.8 3.5-1.7z');
        addPath('M19.6 17.1c-.5 1.1-.8 1.6-1.5 2.6-1 1.5-2.4 3.3-4.1 3.3-1.5 0-1.9-1-3.9-1s-2.5 1-3.9 1c-1.7 0-3-1.6-4-3.1-2.8-4.2-3.1-9.1-1.4-11.7 1.2-1.8 3.1-2.9 4.9-2.9 1.9 0 3 1 4.5 1s2.4-1 4.6-1c1.6 0 3.4.9 4.6 2.4-4 2.2-3.4 7.9.2 9.4z');
        break;
    case 'linux':
        addPath('M12 2.3c-2.3 0-4 2.2-4 5.2 0 1.4.3 2.4.8 3.3-1.8 1.1-3.1 3.5-3.1 6.2 0 3 2.7 4.7 6.3 4.7s6.3-1.7 6.3-4.7c0-2.7-1.3-5.1-3.1-6.2.5-.9.8-1.9.8-3.3 0-3-1.7-5.2-4-5.2zm-2.2 14.4c.8.8 3.6.8 4.4 0 .3 1.2-.5 2.2-2.2 2.2s-2.5-1-2.2-2.2z');
        addCircle('10.2', '7.2', '.8');
        addCircle('13.8', '7.2', '.8');
        break;
    case 'android':
        addPath('M7.2 8.3h9.6c1.2 0 2.2 1 2.2 2.2v6.6c0 1.2-1 2.2-2.2 2.2h-.5v2.2c0 .7-.5 1.2-1.2 1.2s-1.2-.5-1.2-1.2v-2.2h-3.8v2.2c0 .7-.5 1.2-1.2 1.2s-1.2-.5-1.2-1.2v-2.2h-.5C6 19.3 5 18.3 5 17.1v-6.6c0-1.2 1-2.2 2.2-2.2z');
        addPath('M8.1 4.2L6.7 2.1 5.6 2.8l1.5 2.3C6.3 5.8 5.8 6.7 5.6 7.6h12.8c-.2-.9-.7-1.8-1.5-2.5l1.5-2.3-1.1-.7-1.4 2.1C14.8 3.6 13.5 3.3 12 3.3s-2.8.3-3.9.9z');
        addPath('M3.1 10.3c.7 0 1.2.5 1.2 1.2v4.5c0 .7-.5 1.2-1.2 1.2s-1.2-.5-1.2-1.2v-4.5c0-.7.5-1.2 1.2-1.2zM20.9 10.3c.7 0 1.2.5 1.2 1.2v4.5c0 .7-.5 1.2-1.2 1.2s-1.2-.5-1.2-1.2v-4.5c0-.7.5-1.2 1.2-1.2z');
        addCircle('9.5', '6.1', '.55');
        addCircle('14.5', '6.1', '.55');
        break;
    case 'ios':
        addPath('M17.2 2.2c.1 1.3-.5 2.6-1.3 3.5-.9 1-2.1 1.6-3.4 1.5-.1-1.2.5-2.5 1.3-3.4.9-1 2.3-1.7 3.4-1.6z');
        addPath('M20.1 16.6c-.5 1.1-.8 1.6-1.4 2.5-1 1.4-2.3 3.1-3.9 3.1-1.4 0-1.8-.9-3.7-.9s-2.3.9-3.7.9c-1.6 0-2.9-1.5-3.8-2.9-2.7-4-2.9-8.6-1.3-11.1 1.1-1.7 2.9-2.7 4.6-2.7 1.8 0 2.9 1 4.3 1s2.3-1 4.3-1c1.6 0 3.2.8 4.4 2.3-3.8 2-3.2 7.5.2 8.8z');
        break;
    case 'lock':
        addPath('M7 10V7.6C7 4.5 9.1 2.5 12 2.5s5 2 5 5.1V10h1.2c.8 0 1.3.6 1.3 1.3v8.4c0 .8-.6 1.3-1.3 1.3H5.8c-.8 0-1.3-.6-1.3-1.3v-8.4c0-.8.6-1.3 1.3-1.3H7zm2.4 0h5.2V7.6c0-1.7-1-2.8-2.6-2.8S9.4 5.9 9.4 7.6V10zm1.5 4.7c0 .5.3.9.7 1.1v1.7c0 .3.2.5.5.5s.5-.2.5-.5v-1.7c.4-.2.7-.6.7-1.1 0-.7-.5-1.2-1.2-1.2s-1.2.5-1.2 1.2z');
        break;
    case 'laptop':
        addPath('M5 5.2C5 4.5 5.5 4 6.2 4h11.6c.7 0 1.2.5 1.2 1.2v8.6H5V5.2zm2 1.1v5.5h10V6.3H7z');
        addPath('M3.2 16h17.6l1.2 2.5c.2.5-.1 1-.7 1H2.7c-.6 0-.9-.6-.7-1L3.2 16zm6.3 1.2l-.4 1h5.8l-.4-1h-5z');
        break;
    case 'globe':
        addPath('M12 2.5a9.5 9.5 0 1 0 0 19 9.5 9.5 0 0 0 0-19zm-1.1 2.4c-.7.9-1.2 2.1-1.5 3.6H6.6c.9-1.7 2.4-3 4.3-3.6zm-5.1 5.5h3.3c-.1.5-.1 1.1-.1 1.6s0 1.1.1 1.6H5.8c-.1-.5-.2-1.1-.2-1.6s.1-1.1.2-1.6zm.8 5.1h2.8c.3 1.5.8 2.7 1.5 3.6-1.9-.6-3.4-1.9-4.3-3.6zm5.4 3.3c-.4-.6-.8-1.7-1.1-3.3h2.2c-.3 1.6-.7 2.7-1.1 3.3zm1.5-5.2h-3c-.1-.5-.1-1.1-.1-1.6s0-1.1.1-1.6h3c.1.5.1 1.1.1 1.6s0 1.1-.1 1.6zm-.4-5.1h-2.2c.3-1.6.7-2.7 1.1-3.3.4.6.8 1.7 1.1 3.3zm.9 10.6c.7-.9 1.2-2.1 1.5-3.6h2.8c-.9 1.7-2.4 3-4.3 3.6zm4.2-5.5h-3.3c.1-.5.1-1.1.1-1.6s0-1.1-.1-1.6h3.3c.1.5.2 1.1.2 1.6s-.1 1.1-.2 1.6zm-2.7-5.1c-.3-1.5-.8-2.7-1.5-3.6 1.9.6 3.4 1.9 4.3 3.6h-2.8z');
        break;
    default:
        addPath('M3 6.5C3 5.7 3.7 5 4.5 5h5l2 2h8c.8 0 1.5.7 1.5 1.5v8c0 .8-.7 1.5-1.5 1.5h-15C3.7 18 3 17.3 3 16.5v-10z');
        break;
    }

    return svg;
}

function setFolderButtonState(
    button,
    isExpanded
) {

    button.classList.toggle(
        'is-expanded',
        isExpanded
    );

    button.setAttribute(
        'aria-expanded',
        String(isExpanded)
    );
}

function renderTree(
    tree,
    container,
    prefix = ''
) {

    Object.keys(tree)
        .sort((a, b) =>
            a.toLowerCase()
             .localeCompare(b.toLowerCase())
        )
        .forEach(key => {

            const item = tree[key];

            if (item.type === 'folder') {

                const folderPath = prefix + key;

                const folderDiv =
                    document.createElement('div');

                folderDiv.className = 'folder-item';
                folderDiv.dataset.folderPath = folderPath;

                const button =
                    document.createElement('button');

                button.className =
                    'collapse-list-btn';
                button.type = 'button';

                const isExpanded =
                    expandedFolders.has(folderPath);

                button.setAttribute(
                    'aria-label',
                    `${isExpanded ? 'Collapse' : 'Expand'} ${key}`
                );

                button.appendChild(
                    iconElement(
                        folderIconName(folderPath)
                    )
                );

                const label =
                    document.createElement('span');

                label.className =
                    'folder-label';

                label.innerText =
                    key;

                button.appendChild(label);

                setFolderButtonState(
                    button,
                    isExpanded
                );

                const childrenDiv =
                    document.createElement('div');

                childrenDiv.className =
                    'folder-children';

                childrenDiv.style.display =
                    isExpanded ? 'block' : 'none';

                childrenDiv.style.marginLeft =
                    '20px';

                button.onclick = () => {

                    const expanded =
                        childrenDiv.style.display !== 'none';

                    if (expanded) {

                        childrenDiv.style.display =
                            'none';

                        expandedFolders.delete(
                            folderPath
                        );

                        saveState();

                        setFolderButtonState(
                            button,
                            false
                        );

                        button.setAttribute(
                            'aria-label',
                            `Expand ${key}`
                        );
                        updateToggleAllFoldersButton();

                    } else {

                        childrenDiv.style.display =
                            'block';

                        expandedFolders.add(
                            folderPath
                        );

                        saveState();

                        setFolderButtonState(
                            button,
                            true
                        );

                        button.setAttribute(
                            'aria-label',
                            `Collapse ${key}`
                        );
                        updateToggleAllFoldersButton();
                    }
                };

                folderDiv.appendChild(button);

                folderDiv.appendChild(childrenDiv);

                container.appendChild(folderDiv);

                renderTree(
                    item.children,
                    childrenDiv,
                    folderPath + '/'
                );

            } else {

                container.appendChild(
                    createSidebarDocLink(
                        item.path,
                        key.replace('.md', '')
                    )
                );
            }
        });
}

function renderSearchResults() {

    const sidebar =
        getSidebar();

    if (!sidebar) {
        return;
    }

    const status =
        getSearchStatus();

    const query =
        searchQuery.trim();

    sidebar.innerHTML = '';

    if (!query) {

        if (status) {
            status.innerText = '';
        }

        renderNormalSidebar(sidebar);

        return;
    }

    const results =
        searchResults;

    if (status) {
        status.innerText =
            searchLoading
            ? 'Searching...'
            : `${results.length} results`;
    }

    if (results.length === 0) {
        renderEmptyState(sidebar);
        return;
    }

    results.forEach(result => {

        const link =
            createSidebarDocLink(
                result.path,
                result.title,
                'doc-link search-result-link'
            );

        sidebar.appendChild(link);

        if (result.snippet) {

            const snippet =
                document.createElement('div');

            snippet.className = 'search-result-snippet';
            snippet.innerHTML =
                result.snippet;

            sidebar.appendChild(snippet);
        }
    });
}

function updateDocUrl(path, replace = false) {

    const url =
        new URL(window.location.href);

    url.searchParams.set('doc', path);

    url.hash = '';

    if (replace) {
        window.history.replaceState(null, '', url.toString());
    } else {
        window.history.pushState(null, '', url.toString());
    }
}

function getDocFromUrl() {

    const params =
        new URLSearchParams(
            window.location.search
        );

    return params.get('doc');
}

async function loadMarkdownText(path) {

    if (markdownContentCache.has(path)) {
        return markdownContentCache.get(path);
    }

    if (markdownContentLoads.has(path)) {
        return markdownContentLoads.get(path);
    }

    const pending =
        (async () => {

            try {
                const response = await fetch(
                    `/api/rocket/doc?path=${encodeURIComponent(path)}&nocache=${Date.now()}`
                );

                if (!response.ok) {
                    return '';
                }

                const data =
                    await response.json();

                return data.html || '';
            }
            catch (err) {
                console.error(
                    'Error loading doc text:',
                    err
                );
                return '';
            }
        })();

    const pendingWrapper =
        (async () => {
            const text =
                await pending;

            markdownContentCache.set(path, text);
            markdownContentLoads.delete(path);
            return text;
        })()
        .catch(() => {
            markdownContentLoads.delete(path);
            markdownContentCache.set(path, '');
            return '';
        });

    markdownContentLoads.set(path, pendingWrapper);

    return pendingWrapper;
}

function debounceSearch(callback, delay) {

    return (...args) => {

        if (searchDebounceTimer) {
            clearTimeout(
                searchDebounceTimer
            );
        }

        searchDebounceTimer =
            setTimeout(() => {
                callback(...args);
            }, delay);
    };
}

async function performSearch(query) {

    const requestId =
        ++searchRequestId;

    searchLoading = true;
    rerenderSidebar();

    try {
        const response = await fetch(
            `/api/rocket/search?q=${encodeURIComponent(query)}&nocache=${Date.now()}`
        );

        if (requestId !== searchRequestId) {
            return;
        }

        if (!response.ok) {
            throw new Error(
                response.statusText
            );
        }

        const data =
            await response.json();

        searchResults =
            data.results || [];
    }
    catch (err) {

        if (requestId === searchRequestId) {
            console.error(
                'Search error:',
                err
            );
            searchResults = [];
        }
    }
    finally {

        if (requestId === searchRequestId) {
            searchLoading = false;
            rerenderSidebar();
        }
    }
}

const scheduleSearch =
    debounceSearch(
        performSearch,
        250
    );

function initSearchInput() {

    const searchInput =
        document.getElementById('rocketSearchInput');

    if (!searchInput) {
        return;
    }

    searchInput.addEventListener('input', () => {

        searchQuery =
            searchInput.value;

        if (searchQuery.trim()) {
            scheduleSearch(
                searchQuery
            );
        } else {
            searchResults = [];
            searchLoading = false;
            rerenderSidebar();
        }
    });

    searchInput.addEventListener('keydown', (e) => {

        if (e.key === 'Escape') {
            searchInput.value = '';
            searchQuery = '';
            searchResults = [];
            rerenderSidebar();
            searchInput.blur();
        }
    });
}

async function openDocFromUrlIfNeeded() {

    if (initialUrlDocLoaded) {
        return;
    }

    const doc =
        getDocFromUrl();

    if (!doc) {
        initialUrlDocLoaded = true;
        clearExpandedFolders();
        renderSearchResults();
        return;
    }

    initialUrlDocLoaded = true;

    if (!allMarkdownFiles.includes(doc)) {

        renderWikiError(
            'Rocket document not found',
            `The URL points to ${doc}, but that file is not in rocket-index.json.`,
            [
                'Check that the file exists under the rocket folder.',
                'Click the refresh button after adding new rocket files.'
            ]
        );

        return;
    }

    await loadDoc(doc, {
        updateUrl: false
    });
}

async function loadIndex() {

    if (indexLoadInProgress) {
        return;
    }

    indexLoadInProgress = true;

    try {

        if (isDirectFileOpen()) {

            renderWikiError(
                'Start the Rock-OS Rocket server',
                'This page was opened directly from the filesystem, so the browser cannot safely load the rocket index or rocket files.',
                [
                    'Open a terminal in the repo root or Website folder.',
                    'Run the Go server command below.',
                    'Use the http:// address printed by the server instead of opening rocket.html directly.'
                ]
            );

            return;
        }

        const response = await fetch(
            'rocket-index.json?nocache=' +
            Date.now()
        );

        if (!response.ok) {

            renderWikiError(
                'Could not load the rocket index',
                'The page loaded, but rocket-index.json was not available from the server.',
                [
                    'Make sure the Go server is running from the repo root or Website folder.',
                    `The server returned HTTP ${response.status}.`
                ]
            );

            return;
        }

        const rawText =
            await response.text();

        const parsed =
            rawText.trim()
            ? JSON.parse(rawText)
            : [];

        const files =
            normalizeFiles(parsed);

        allMarkdownFiles = files;

        const urlDoc =
            getDocFromUrl();

        if (urlDoc && files.includes(urlDoc)) {
            rememberActiveDoc(urlDoc);
        }

        const nextIndexText =
            rawText.trim();

        if (nextIndexText === lastIndexText) {
            return;
        }

        lastIndexText = nextIndexText;

        const sidebar =
            getSidebar();

        if (!sidebar) {
            return;
        }

        syncExpandedFoldersFromDom();

        if (activeDocPath) {
            rememberActiveDoc(activeDocPath);
        }

        sidebar.innerHTML = '';

        if (!files.length) {

            renderEmptyState(sidebar);
            return;
        }

        if (searchQuery) {
            scheduleSearch();
        } else {
            renderNormalSidebar(sidebar);
        }

        await openDocFromUrlIfNeeded();
    }
    catch (err) {

        console.error(
            'loadIndex error:',
            err
        );

        renderWikiError(
            'Could not connect to the rocket server',
                'The browser could not load the rocket index. The server may not be running, or the page may have been opened from the wrong place.',
                [
                'Start the Go server from the repo root or Website folder.',
                'Open the http:// address printed by the server.'
            ]
        );
    }
    finally {

        indexLoadInProgress = false;
    }
}

function clearCacheAndReload() {

    markdownContentCache.clear();
    markdownContentLoads.clear();
    lastIndexText = '';
    loadIndex();
}

function initRefreshButton() {

    const btn =
        document.getElementById(
            'refreshRocketBtn'
        );

    if (btn) {

        btn.onclick = () => {

            const originalText =
                btn.innerHTML;

            btn.innerHTML =
                '&#x21BB;';

            btn.classList.add(
                'spinning'
            );

            clearCacheAndReload();

            setTimeout(() => {
                btn.classList.remove(
                    'spinning'
                );
                btn.innerHTML =
                    originalText;
            }, 600);
        };
    }
}

function initSidebarResizer() {
    // Sidebar resizing is loaded via sidebar-resizer.js
}

function initNavigationListener() {

    window.addEventListener('popstate', () => {

        const doc =
            getDocFromUrl();

        if (doc && allMarkdownFiles.includes(doc)) {
            loadDoc(doc, {
                updateUrl: false
            });
        } else if (!doc) {
            clearExpandedFolders();
            const content = document.getElementById('content');
            if (content) {
                content.innerHTML = `
                    <h1>Rocket</h1>
                    <p>Select a rocket document.</p>
                `;
            }
        }
    });
}

function initToggleAllFoldersButton() {

    const btn =
        document.getElementById(
            'toggleAllFoldersBtn'
        );

    if (btn) {

        btn.onclick = () => {

            const allExpanded =
                areAllFoldersExpanded();

            setAllFoldersExpanded(
                !allExpanded
            );
        };
    }
}

document.addEventListener('DOMContentLoaded', () => {

    initSearchInput();
    initRefreshButton();
    initSidebarResizer();
    initNavigationListener();
    initToggleAllFoldersButton();
    loadIndex();
});
