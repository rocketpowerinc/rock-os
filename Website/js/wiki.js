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

    const md = window.markdownit({
        html: true,
        linkify: true,
        breaks: true,
        typographer: true
    });

    const content =
        document.getElementById('content');

    content.innerHTML =
        md.render(text);

    enhanceCodeBlocks(content);
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
                    'aria-expanded',
                    String(isExpanded)
                );

                button.innerText =
                    (isExpanded ? '▼ ' : '▶ ') + key;

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

                        button.innerText =
                            '▶ ' + key;

                        button.setAttribute(
                            'aria-expanded',
                            'false'
                        );

                    } else {

                        childrenDiv.style.display =
                            'block';

                        expandedFolders.add(
                            folderPath
                        );

                        saveState();

                        button.innerText =
                            '▼ ' + key;

                        button.setAttribute(
                            'aria-expanded',
                            'true'
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
