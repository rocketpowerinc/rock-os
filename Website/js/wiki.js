const expandedFolders = new Set();

let lastIndexText = '';
let activeDocPath = '';

function getSidebar() {

    return document.getElementById(
        'sidebarListContainer'
    );
}

function normalizeFiles(parsed) {

    const files =
        Array.isArray(parsed)
        ? parsed
        : [parsed];

    return files.filter(file =>
        typeof file === 'string' &&
        file.trim().toLowerCase().endsWith('.md')
    );
}

function renderEmptyState(container) {

    const empty =
        document.createElement('div');

    empty.className = 'wiki-empty-state';

    empty.innerText =
        'No markdown files found.';

    container.appendChild(empty);
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

            const button =
                document.createElement('button');

            button.className = 'copy-code-btn';
            button.type = 'button';
            button.innerText = 'Copy';

            button.onclick = async () => {

                try {

                    await copyText(code.innerText);

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

            pre.parentNode.insertBefore(wrapper, pre);
            wrapper.appendChild(button);
            wrapper.appendChild(pre);
        });
}

async function loadDoc(path) {

    rememberActiveDoc(path);

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

                const button =
                    document.createElement('button');

                button.className =
                    'collapse-list-btn';

                const isExpanded =
                    expandedFolders.has(folderPath);

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

                        button.innerText =
                            '▶ ' + key;

                    } else {

                        childrenDiv.style.display =
                            'block';

                        expandedFolders.add(
                            folderPath
                        );

                        button.innerText =
                            '▼ ' + key;
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

async function loadIndex() {

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

        sidebar.innerHTML = '';

        if (!files.length) {

            renderEmptyState(sidebar);
            return;
        }

        renderTree(
            tree,
            sidebar,
            ''
        );
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
}

loadIndex();

setInterval(loadIndex, 5000);
