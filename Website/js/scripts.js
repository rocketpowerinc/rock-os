const scriptList =
    document.getElementById('scriptList');
const scriptPreview =
    document.getElementById('scriptPreview');
const scriptStatus =
    document.getElementById('scriptStatus');
const scriptMeta =
    document.getElementById('scriptMeta');
const runScriptBtn =
    document.getElementById('runScriptBtn');
const toggleAllScriptsBtn =
    document.getElementById('toggleAllScriptsBtn');

let selectedScript = null;

function setStatus(message, type = 'info') {
    scriptStatus.textContent = message;
    scriptStatus.dataset.type = type;
}

function escapeHTML(value) {
    return value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}

function scriptLanguage(script) {
    const id =
        script.id.toLowerCase();

    if (id.endsWith('.ps1')) {
        return 'powershell';
    }

    if (id.endsWith('.cmd') || id.endsWith('.bat')) {
        return 'batch';
    }

    return 'shell';
}

function highlightScript(content, script) {
    const language =
        scriptLanguage(script);

    return content
        .split('\n')
        .map(line => highlightScriptLine(line, language))
        .join('\n');
}

function highlightScriptLine(line, language) {
    const trimmed =
        line.trimStart();

    if (
        trimmed.startsWith('#') ||
        trimmed.toLowerCase().startsWith('rem ') ||
        trimmed.startsWith('::')
    ) {
        return `<span class="script-token-comment">${escapeHTML(line)}</span>`;
    }

    let html =
        escapeHTML(line);

    if (language === 'powershell') {
        html = html
            .replace(/(\$[A-Za-z_][\w:]*)/g, '<span class="script-token-variable">$1</span>')
            .replace(/\b(param|function|if|else|elseif|foreach|for|while|switch|try|catch|finally|return|exit|Write-Host|Read-Host|New-Item|Test-Path)\b/g, '<span class="script-token-keyword">$1</span>');
    }
    else if (language === 'batch') {
        html = html
            .replace(/(%[A-Za-z_][\w]*%|![A-Za-z_][\w]*!)/g, '<span class="script-token-variable">$1</span>')
            .replace(/\b(@echo|echo|set|setlocal|if|else|for|do|exit|mkdir|call|pause)\b/gi, '<span class="script-token-keyword">$1</span>');
    }
    else {
        html = html
            .replace(/(\$[A-Za-z_][\w]*|\$\{[^}]+\})/g, '<span class="script-token-variable">$1</span>')
            .replace(/\b(if|then|else|elif|fi|case|esac|for|while|do|done|read|printf|mkdir|exit|set|export|sudo)\b/g, '<span class="script-token-keyword">$1</span>');
    }

    return html;
}

function renderScriptPreview(content, script) {
    scriptPreview.className =
        'language-' + scriptLanguage(script);
    scriptPreview.innerHTML =
        highlightScript(content, script);
}

function currentClientPlatform() {
    const platform =
        (
            navigator.userAgentData &&
            navigator.userAgentData.platform ?
                navigator.userAgentData.platform :
                navigator.platform
        ).toLowerCase();

    if (platform.includes('win')) {
        return 'Windows';
    }

    if (platform.includes('mac')) {
        return 'Mac';
    }

    if (platform.includes('linux')) {
        return 'Linux';
    }

    return 'Unknown';
}

function scriptFolderPlatform(script) {
    const topFolder =
        script.id.split('/')[0].toLowerCase();

    if (topFolder === 'windows') {
        return 'Windows';
    }

    if (topFolder === 'mac' || topFolder === 'macos') {
        return 'Mac';
    }

    if (topFolder === 'linux') {
        return 'Linux';
    }

    return 'Universal';
}

function scriptPlatformWarning(script) {
    const currentPlatform =
        currentClientPlatform();
    const scriptPlatform =
        scriptFolderPlatform(script);

    if (
        currentPlatform === 'Unknown' ||
        scriptPlatform === 'Universal' ||
        currentPlatform === scriptPlatform
    ) {
        return '';
    }

    return `This script is stored under ${scriptPlatform}, but this browser appears to be on ${currentPlatform}. It may not work here.`;
}

function scriptButton(script) {
    const button =
        document.createElement('button');

    button.type = 'button';
    button.className = 'script-tree-file';
    button.dataset.scriptId = script.id;

    const name =
        document.createElement('strong');
    name.textContent = script.name;

    button.append(name);
    button.addEventListener('click', () => selectScript(script));

    return button;
}

function folderNode(name) {
    const details =
        document.createElement('details');
    details.className = 'script-tree-folder';
    details.open = false;

    const summary =
        document.createElement('summary');
    summary.textContent = name;

    const children =
        document.createElement('div');
    children.className = 'script-tree-children';

    details.append(summary, children);
    details.addEventListener('toggle', updateToggleAllScriptsButton);

    return {
        details,
        children
    };
}

function scriptFolders() {
    return Array.from(
        scriptList.querySelectorAll('.script-tree-folder')
    );
}

function allScriptFoldersExpanded() {
    const folders =
        scriptFolders();

    return folders.length > 0 &&
        folders.every(folder => folder.open);
}

function updateToggleAllScriptsButton() {
    if (!toggleAllScriptsBtn) {
        return;
    }

    const allExpanded =
        allScriptFoldersExpanded();

    toggleAllScriptsBtn.innerHTML =
        allExpanded ? '&#x229F;' : '&#x229E;';
    toggleAllScriptsBtn.setAttribute(
        'aria-label',
        allExpanded ? 'Fold all script folders' : 'Expand all script folders'
    );
    toggleAllScriptsBtn.title =
        allExpanded ? 'Fold all script folders' : 'Expand all script folders';
}

function setAllScriptFoldersExpanded(expanded) {
    scriptFolders()
        .forEach(folder => {
            folder.open = expanded;
        });

    updateToggleAllScriptsButton();
}

function renderScriptTree(scripts) {
    scriptList.textContent = '';

    const root =
        document.createElement('div');
    root.className = 'script-tree-root';

    const folders =
        new Map();

    scripts.forEach(script => {
        const parts =
            script.id.split('/');
        let parent =
            root;
        let key =
            '';

        parts.slice(0, -1).forEach(part => {
            key =
                key ? `${key}/${part}` : part;

            if (!folders.has(key)) {
                const node =
                    folderNode(part);

                folders.set(key, node.children);
                parent.appendChild(node.details);
            }

            parent =
                folders.get(key);
        });

        parent.appendChild(scriptButton(script));
    });

    scriptList.appendChild(root);
    updateToggleAllScriptsButton();
}

async function loadScripts() {
    try {
        const response =
            await fetch('/api/scripts');

        if (!response.ok) {
            if (response.status === 404) {
                throw new Error(
                    'The running Rock-OS server does not support the script dashboard yet. Start from Go source with the platform start-rock-os-from-source script in START-HERE, or build a new release binary.'
                );
            }

            throw new Error(await response.text());
        }

        const scripts =
            await response.json();

        if (scripts.length === 0) {
            scriptList.textContent = 'No scripts found in Website/scripts.';
            return;
        }

        renderScriptTree(scripts);
    }
    catch (err) {
        setStatus(err.message, 'error');
    }
}

async function selectScript(script) {
    selectedScript = script;
    runScriptBtn.disabled = !script.runnable;

    document.querySelectorAll('.script-tree-file')
        .forEach(item => {
            item.classList.toggle('active', item.dataset.scriptId === script.id);
        });

    const warning =
        scriptPlatformWarning(script);

    if (warning) {
        setStatus(warning, 'warn');
    }
    else {
        setStatus(
            script.runnable ?
                'Review the script, then run it when ready.' :
                'This script is visible for review but does not run on this operating system.',
            script.runnable ? 'info' : 'warn'
        );
    }

    scriptMeta.textContent =
        `${script.path} - ${script.platform}`;

    try {
        const response =
            await fetch('/api/scripts/content?id=' + encodeURIComponent(script.id));

        if (!response.ok) {
            throw new Error(await response.text());
        }

        const result =
            await response.json();

        renderScriptPreview(result.content, script);
    }
    catch (err) {
        setStatus(err.message, 'error');
    }
}

async function runSelectedScript() {
    if (!selectedScript || !selectedScript.runnable) {
        return;
    }

    setStatus('Opening script in your OS terminal...', 'info');

    try {
        const response =
            await fetch('/api/scripts/run', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Rock-OS-Requested': 'true'
                },
                body: JSON.stringify({
                    id: selectedScript.id
                })
            });

        if (!response.ok) {
            throw new Error(await response.text());
        }

        setStatus('Script opened in your OS terminal.', 'success');
    }
    catch (err) {
        setStatus(err.message, 'error');
    }
}

runScriptBtn.addEventListener('click', runSelectedScript);
toggleAllScriptsBtn.addEventListener('click', () => {
    setAllScriptFoldersExpanded(!allScriptFoldersExpanded());
});
loadScripts();
