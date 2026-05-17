const expandedFolders = new Set();
const stateStorageKey = 'rock-os-wiki-state';

let lastIndexText = '';
let activeDocPath = '';
let indexLoadInProgress = false;
let allMarkdownFiles = [];
let searchQuery = '';
let initialUrlDocLoaded = false;

const markdownContentCache = new Map();
const markdownContentLoads = new Map();

loadSavedState();

function getSidebar() {

    return document.getElementById(
        'sidebarListContainer'
    );
}

function getSearchStatus() {

    return document.getElementById(
        'wikiSearchStatus'
    );
}

function normalizeFiles(parsed) {

    const files =
        Array.isArray(parsed)
        ? parsed
        : [parsed];

    return files
        .filter(file =>
            typeof file === 'string' &&
            file.trim().toLowerCase().endsWith('.md')
        )
        .map(file =>
            file.trim()
        )
        .sort((a, b) =>
            a.toLowerCase()
                .localeCompare(b.toLowerCase())
        );
}

function fileTitle(path) {

    const parts =
        path.split('/');

    return parts[parts.length - 1]
        .replace(/\.md$/i, '');
}

function escapeHtml(value) {

    return value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function getSnippet(text, query) {

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

function getDocFromUrl() {

    const params =
        new URLSearchParams(window.location.search);

    const doc =
        params.get('doc');

    if (!doc || !doc.toLowerCase().endsWith('.md')) {
        return '';
    }

    return doc;
}

function updateDocUrl(path, replace = false) {

    const url =
        new URL(window.location.href);

    url.searchParams.set('doc', path);

    if (replace) {
        window.history.replaceState({}, '', url);
    } else {
        window.history.pushState({}, '', url);
    }
}

function loadSavedState() {

    try {

        const saved =
            JSON.parse(
                localStorage.getItem(stateStorageKey) || '{}'
            );

        if (Array.isArray(saved.expandedFolders)) {

            saved.expandedFolders
                .filter(folderPath =>
                    typeof folderPath === 'string'
                )
                .forEach(folderPath =>
                    expandedFolders.add(folderPath)
                );
        }
    }
    catch (err) {

        console.warn('Could not load wiki state:', err);
    }
}

function saveState() {

    try {

        localStorage.setItem(
            stateStorageKey,
            JSON.stringify({
                expandedFolders: Array.from(
                    expandedFolders
                )
            })
        );
    }
    catch (err) {

        console.warn('Could not save wiki state:', err);
    }
}

function clearExpandedFolders() {

    expandedFolders.clear();
    saveState();
}

function renderEmptyState(container) {

    const empty =
        document.createElement('div');

    empty.className = 'wiki-empty-state';

    empty.innerText =
        'No markdown files found.';

    container.appendChild(empty);
}

function renderWelcomeState() {

    activeDocPath = '';
    clearExpandedFolders();
    updateActiveDocLinks();

    const content =
        document.getElementById('content');

    if (!content) {
        return;
    }

    content.innerHTML = `
        <h1>Rock-OS Wiki</h1>
        <p>Select a markdown document.</p>
    `;
}

async function loadMarkdownText(path) {

    if (markdownContentCache.has(path)) {
        return markdownContentCache.get(path);
    }

    if (markdownContentLoads.has(path)) {
        return markdownContentLoads.get(path);
    }

    const pending =
        fetch(path + '?nocache=' + Date.now())
            .then(response => {

                if (!response.ok) {
                    return '';
                }

                return response.text();
            })
            .then(text => {

                markdownContentCache.set(path, text);
                markdownContentLoads.delete(path);
                return text;
            })
            .catch(err => {

                console.warn('Search index failed:', path, err);
                markdownContentLoads.delete(path);
                markdownContentCache.set(path, '');
                return '';
            });

    markdownContentLoads.set(path, pending);

    return pending;
}

function warmSearchIndex(files) {

    Promise.all(
        files.map(file =>
            loadMarkdownText(file)
        )
    ).then(() => {

        if (searchQuery) {
            renderSearchResults();
        }
    });
}

function renderWikiError(title, message, details = []) {

    const sidebar =
        getSidebar();

    if (sidebar) {

        sidebar.innerHTML = '';

        const empty =
            document.createElement('div');

        empty.className = 'wiki-empty-state';
        empty.innerText = 'Wiki server is not available.';

        sidebar.appendChild(empty);
    }

    const content =
        document.getElementById('content');

    if (!content) {
        return;
    }

    const detailItems =
        details
            .map(detail => `<li>${detail}</li>`)
            .join('');

    content.innerHTML = `
        <div class="wiki-error-panel">
            <p class="wiki-error-kicker">Wiki offline</p>
            <h1>${title}</h1>
            <p>${message}</p>
            ${detailItems ? `<ul>${detailItems}</ul>` : ''}
            <pre><code>cd Website
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
            .replace(/^markdown\//, '')
            .split('/');

    parts.pop();

    return parts.map((_, index) =>
        parts.slice(0, index + 1).join('/')
    );
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

async function copyText(text) {

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

function codeLanguage(code) {

    const languageClass =
        Array.from(code.classList)
            .find(className =>
                className.startsWith('language-')
            );

    if (!languageClass) {
        return 'text';
    }

    const language =
        languageClass.replace('language-', '').toLowerCase();

    const aliases = {
        ps1: 'powershell',
        pwsh: 'powershell',
        shell: 'bash',
        sh: 'bash',
        zsh: 'bash'
    };

    return aliases[language] || language;
}

function languageLabel(language) {

    const labels = {
        bash: 'Bash',
        powershell: 'PowerShell',
        text: 'Text'
    };

    return labels[language] || language;
}

function highlightCode(code, rawText, language) {

    if (!window.hljs) {
        return;
    }

    try {

        if (language !== 'text' && window.hljs.getLanguage(language)) {

            code.innerHTML =
                window.hljs.highlight(rawText, {
                    language
                }).value;
        } else {

            code.innerHTML =
                window.hljs.highlightAuto(rawText).value;
        }

        code.classList.add('hljs');
    }
    catch (err) {

        console.warn('Code highlighting failed:', err);
        code.innerText = rawText;
    }
}

function createLineNumbers(rawText) {

    const lineCount =
        Math.max(
            1,
            rawText.replace(/\n$/, '').split('\n').length
        );

    const gutter =
        document.createElement('div');

    gutter.className = 'code-line-numbers';
    gutter.setAttribute('aria-hidden', 'true');

    for (let index = 1; index <= lineCount; index += 1) {

        const line =
            document.createElement('span');

        line.innerText = String(index);
        gutter.appendChild(line);
    }

    return gutter;
}

function calloutMeta(type) {

    const normalizedType =
        type.toLowerCase();

    const labels = {
        note: 'Note',
        info: 'Info',
        tip: 'Tip',
        success: 'Success',
        warning: 'Warning',
        danger: 'Danger',
        error: 'Error',
        question: 'Question'
    };

    return {
        type: normalizedType,
        label: labels[normalizedType] || type
    };
}

function enhanceCallouts(container) {

    container.querySelectorAll('blockquote')
        .forEach(blockquote => {

            const firstParagraph =
                blockquote.querySelector('p');

            if (!firstParagraph) {
                return;
            }

            const match =
                firstParagraph.textContent
                    .trimStart()
                    .match(/^\[!(\w+)\]/);

            if (!match) {
                return;
            }

            const meta =
                calloutMeta(match[1]);

            firstParagraph.innerHTML =
                firstParagraph.innerHTML.replace(
                    /^\s*\[!\w+\]\s*/i,
                    ''
                );

            const title =
                document.createElement('div');

            title.className = 'callout-title';
            title.innerText = meta.label;

            blockquote.classList.add(
                'callout',
                `callout-${meta.type}`
            );

            blockquote.insertBefore(
                title,
                blockquote.firstChild
            );

            if (!firstParagraph.textContent.trim()) {
                firstParagraph.remove();
            }
        });
}

function enhanceCodeBlocks(container) {

    container.querySelectorAll('pre > code')
        .forEach(code => {

            const pre = code.parentElement;

            if (!pre || pre.parentElement.classList.contains(
                'code-block'
            )) {
                return;
            }

            const wrapper =
                document.createElement('div');

            wrapper.className = 'code-block';

            const rawText =
                code.textContent;

            const language =
                codeLanguage(code);

            highlightCode(code, rawText, language);

            const header =
                document.createElement('div');

            header.className = 'code-block-header';

            const label =
                document.createElement('span');

            label.className = 'code-language-label';
            label.innerText = languageLabel(language);

            const button =
                document.createElement('button');

            button.className = 'copy-code-btn';
            button.type = 'button';
            button.innerText = 'Copy';

            button.onclick = async () => {

                try {

                    await copyText(rawText);

                    button.innerText = 'Copied';

                    setTimeout(() => {
                        button.innerText = 'Copy';
                    }, 1600);
                }
                catch (err) {

                    console.error('Copy failed:', err);
                    button.innerText = 'Error';

                    setTimeout(() => {
                        button.innerText = 'Copy';
                    }, 1600);
                }
            };

            const body =
                document.createElement('div');

            body.className = 'code-block-body';

            pre.parentNode.insertBefore(wrapper, pre);
            header.appendChild(label);
            header.appendChild(button);
            body.appendChild(createLineNumbers(rawText));
            body.appendChild(pre);
            wrapper.appendChild(header);
            wrapper.appendChild(body);
        });
}

function normalizeDocPath(path) {

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

function resolveMarkdownLink(
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

    if (pathOnly.startsWith('markdown/')) {
        return normalizeDocPath(pathOnly);
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

function wikiDocHref(path) {

    const url =
        new URL('wiki.html', window.location.href);

    url.searchParams.set('doc', path);

    return `${url.pathname}${url.search}`;
}

function enhanceWikiLinks(
    container,
    currentDocPath
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

function formatEditedDate(value) {

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

async function loadDoc(path, options = {}) {

    const shouldUpdateUrl =
        options.updateUrl !== false;

    rememberActiveDoc(path);
    updateActiveDocLinks();

    if (shouldUpdateUrl) {
        updateDocUrl(path, options.replaceUrl === true);
    }

    const response = await fetch(
        path + '?nocache=' + Date.now()
    );

    if (!response.ok) {

        console.error('Failed:', path);
        return;
    }

    const text = await response.text();

    const lastEdited =
        formatEditedDate(
            response.headers.get('Last-Modified')
        );

    const md = window.markdownit({
        html: true,
        linkify: true,
        breaks: true,
        typographer: true
    });

    const content =
        document.getElementById('content');

    content.innerHTML =
        `${lastEdited
            ? `<div class="doc-meta">Last edited ${escapeHtml(lastEdited)}</div>`
            : ''}${md.render(text)}`;

    enhanceCodeBlocks(content);
    enhanceCallouts(content);
    enhanceWikiLinks(content, path);
}

function buildTree(files) {

    const tree = {};

    files.forEach(file => {

        const parts = file
            .replace('markdown/', '')
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

    if (folderPath.includes('/')) {
        return 'folder';
    }

    const rootFolder =
        folderPath.split('/')[0].toLowerCase();

    if (rootFolder === 'windows') {
        return 'windows';
    }

    if (rootFolder === 'mac') {
        return 'mac';
    }

    if (rootFolder === 'linux') {
        return 'linux';
    }

    if (rootFolder === 'android') {
        return 'android';
    }

    if (rootFolder === 'ios') {
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

                const link =
                    document.createElement('a');

                link.className = 'doc-link';

                link.href = '#';
                link.dataset.path = item.path;

                if (item.path === activeDocPath) {
                    link.classList.add('active');
                }

                link.innerText =
                    key.replace('.md', '');

                link.onclick = () => {

                    loadDoc(item.path);

                    return false;
                };

                container.appendChild(link);
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
        searchQuery.trim().toLowerCase();

    sidebar.innerHTML = '';

    if (!query) {

        if (status) {
            status.innerText = '';
        }

        renderTree(
            buildTree(allMarkdownFiles),
            sidebar,
            ''
        );

        return;
    }

    const results =
        allMarkdownFiles
            .map(file => {

                const title =
                    fileTitle(file);

                const searchablePath =
                    file.toLowerCase();

                const markdownText =
                    markdownContentCache.get(file) || '';

                const searchableText =
                    markdownText.toLowerCase();

                const nameMatch =
                    searchablePath.includes(query) ||
                    title.toLowerCase().includes(query);

                const contentMatch =
                    searchableText.includes(query);

                if (!nameMatch && !contentMatch) {
                    return null;
                }

                return {
                    file,
                    title,
                    snippet: contentMatch
                        ? getSnippet(markdownText, query)
                        : ''
                };
            })
            .filter(Boolean);

    if (status) {

        const loadingCount =
            allMarkdownFiles.filter(file =>
                !markdownContentCache.has(file)
            ).length;

        status.innerText =
            `${results.length} result${results.length === 1 ? '' : 's'}` +
            (loadingCount ? `, indexing ${loadingCount}` : '');
    }

    if (!results.length) {

        renderEmptyState(sidebar);
        return;
    }

    results.forEach(result => {

        const link =
            document.createElement('a');

        link.className = 'doc-link';
        link.href = '#';
        link.dataset.path = result.file;

        if (result.file === activeDocPath) {
            link.classList.add('active');
        }

        link.innerHTML =
            `${escapeHtml(result.title)}<span class="search-result-meta">${escapeHtml(result.file)}</span>`;

        link.onclick = () => {

            loadDoc(result.file);
            return false;
        };

        sidebar.appendChild(link);

        if (result.snippet) {

            const snippet =
                document.createElement('div');

            snippet.className = 'search-result-snippet';
            snippet.innerText = result.snippet;

            sidebar.appendChild(snippet);
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
            'Markdown document not found',
            `The URL points to ${doc}, but that file is not in markdown-index.json.`,
            [
                'Check that the file exists under the markdown folder.',
                'Click the refresh button after adding new markdown files.'
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
                'Start the Rock-OS Wiki server',
                'This page was opened directly from the filesystem, so the browser cannot safely load the markdown index or markdown files.',
                [
                    'Open a terminal in the Website folder.',
                    'Run the Go server command below.',
                    'Use the http:// address printed by the server instead of opening wiki.html directly.'
                ]
            );

            return;
        }

        const response = await fetch(
            'markdown-index.json?nocache=' +
            Date.now()
        );

        if (!response.ok) {

            renderWikiError(
                'Could not load the wiki index',
                'The page loaded, but markdown-index.json was not available from the server.',
                [
                    'Make sure the Go server is running from the Website folder.',
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

        warmSearchIndex(files);

        const nextIndexText =
            JSON.stringify(files);

        if (nextIndexText === lastIndexText) {
            return;
        }

        lastIndexText = nextIndexText;

        const tree =
            buildTree(files);

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
            renderSearchResults();
        } else {
            renderTree(
                tree,
                sidebar,
                ''
            );
        }

        await openDocFromUrlIfNeeded();
    }
    catch (err) {

        console.error(
            'loadIndex error:',
            err
        );

        renderWikiError(
            'Could not connect to the wiki server',
            'The browser could not load the markdown index. The server may not be running, or the page may have been opened from the wrong place.',
            [
                'Start the Go server from the Website folder.',
                'Open the http:// address printed by the server.'
            ]
        );
    }
    finally {

        indexLoadInProgress = false;
    }
}

async function refreshWiki() {

    lastIndexText = '';
    markdownContentCache.clear();
    markdownContentLoads.clear();

    await loadIndex();

    if (activeDocPath) {
        await loadDoc(activeDocPath);
    }
}

const refreshWikiBtn =
    document.getElementById('refreshWikiBtn');

const wikiSearchInput =
    document.getElementById('wikiSearchInput');

if (refreshWikiBtn) {

    refreshWikiBtn.addEventListener('click', async () => {

        refreshWikiBtn.disabled = true;
        refreshWikiBtn.classList.add('is-refreshing');

        try {

            await refreshWiki();
        }
        finally {

            refreshWikiBtn.disabled = false;
            refreshWikiBtn.classList.remove('is-refreshing');
        }
    });
}

if (wikiSearchInput) {

    wikiSearchInput.addEventListener('input', () => {

        searchQuery =
            wikiSearchInput.value;

        if (searchQuery) {
            warmSearchIndex(allMarkdownFiles);
        }

        renderSearchResults();
    });
}

window.addEventListener('popstate', () => {

    const doc =
        getDocFromUrl();

    if (doc && allMarkdownFiles.includes(doc)) {

        loadDoc(doc, {
            updateUrl: false
        });

        return;
    }

    renderWelcomeState();
});

loadIndex();
