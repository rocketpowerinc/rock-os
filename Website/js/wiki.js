const expandedFolders = new Set();

let lastIndexText = '';

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

async function loadDoc(path) {

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

    document.getElementById('content').innerHTML =
        md.render(text);
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

        const response = await fetch(
            'markdown-index.json?nocache=' +
            Date.now()
        );

        if (!response.ok) {

            console.error(
                'Failed to load markdown-index.json'
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
    }
}

loadIndex();

setInterval(loadIndex, 5000);
