async function loadDoc(path){

const response = await fetch(
path + '?nocache=' + Date.now()
);

if(!response.ok){
console.error('Failed:', path);
return;
}

const text = await response.text();

const md = window.markdownit({
html:true,
linkify:true,
typographer:true
});

document.getElementById('content').innerHTML =
md.render(text);

}

function buildTree(files) {
    const tree = {};
    files.forEach(file => {
        const parts = file.replace('markdown/', '').split('/');
        let current = tree;
        parts.forEach((part, index) => {
            if (!current[part]) {
                current[part] = index === parts.length - 1 ? { type: 'file', path: file } : { type: 'folder', children: {} };
            }
            if (current[part].type === 'folder') {
                current = current[part].children;
            }
        });
    });
    return tree;
}

function getExpandedFolders(container) {
    const expanded = new Set();
    const buttons = container.querySelectorAll('.collapse-list-btn');
    buttons.forEach(button => {
        if (button.innerText.startsWith('▼')) {
            expanded.add(button.dataset.path);
        }
    });
    return expanded;
}

function renderTree(tree, container, prefix = '', expanded = new Set()) {
    Object.keys(tree).sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase())).forEach(key => {
        const item = tree[key];
        if (item.type === 'folder') {
            const folderDiv = document.createElement('div');
            folderDiv.className = 'folder-item';

            const button = document.createElement('button');
            button.className = 'collapse-list-btn';
            button.dataset.path = prefix + key;
            const isExpanded = expanded.has(button.dataset.path);
            button.innerText = isExpanded ? '▼ ' + key : '▶ ' + key;
            button.onclick = () => {
                const childrenDiv = folderDiv.querySelector('.folder-children');
                if (childrenDiv.style.display === 'none') {
                    childrenDiv.style.display = 'block';
                    button.innerText = '▼ ' + key;
                } else {
                    childrenDiv.style.display = 'none';
                    button.innerText = '▶ ' + key;
                }
            };

            const childrenDiv = document.createElement('div');
            childrenDiv.className = 'folder-children';
            childrenDiv.style.display = isExpanded ? 'block' : 'none';
            childrenDiv.style.marginLeft = '20px';

            folderDiv.appendChild(button);
            folderDiv.appendChild(childrenDiv);
            container.appendChild(folderDiv);

            renderTree(item.children, childrenDiv, prefix + key + '/', expanded);
        } else {
            const link = document.createElement('a');
            link.className = 'doc-link';
            link.href = '#';
            const cleanName = key.replace('.md','');
            link.innerText = cleanName;
            link.onclick = () => { loadDoc(item.path); return false; };
            container.appendChild(link);
        }
    });
}

async function loadIndex(){

const response = await fetch(
'markdown-index.json?nocache=' + Date.now()
);

const files = await response.json();

const tree = buildTree(files);

const sidebarListContainer = document.getElementById('sidebarListContainer');
if (sidebarListContainer) {
    const expanded = getExpandedFolders(sidebarListContainer);
    sidebarListContainer.innerHTML = ''; // Clear existing
    renderTree(tree, sidebarListContainer, '', expanded);
}

}


loadIndex();

// Reload the index every 5 seconds to pick up new files
setInterval(loadIndex, 5000);
