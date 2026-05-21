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
const previewPanel =
    document.getElementById('previewPanel');
const terminalPanel =
    document.getElementById('terminalPanel');
const scriptTerminal =
    document.getElementById('scriptTerminal');
const terminalMeta =
    document.getElementById('terminalMeta');
const terminalInputForm =
    document.getElementById('terminalInputForm');
const terminalInput =
    document.getElementById('terminalInput');
const terminalSecretInput =
    document.getElementById('terminalSecretInput');
const sendInputBtn =
    document.getElementById('sendInputBtn');

let selectedScript = null;
let activeSessionId = null;
let outputStream = null;
let terminalAnsiClasses = [];

function setStatus(message, type = 'info') {
    scriptStatus.textContent = message;
    scriptStatus.dataset.type = type;
}

function setTerminalInputEnabled(enabled) {
    terminalInput.disabled = !enabled;
    terminalSecretInput.disabled = !enabled;
    sendInputBtn.disabled = !enabled;
}

function showPreviewMode() {
    previewPanel.hidden = false;
    terminalPanel.hidden = true;
    terminalMeta.textContent = 'Idle';
    setTerminalInputEnabled(false);
}

function showTerminalMode() {
    previewPanel.hidden = true;
    terminalPanel.hidden = false;
}

function terminalAnsiClass(code) {
    const classes = {
        1: 'ansi-bold',
        2: 'ansi-dim',
        3: 'ansi-italic',
        4: 'ansi-underline',
        30: 'ansi-black',
        31: 'ansi-red',
        32: 'ansi-green',
        33: 'ansi-yellow',
        34: 'ansi-blue',
        35: 'ansi-magenta',
        36: 'ansi-cyan',
        37: 'ansi-white',
        90: 'ansi-bright-black',
        91: 'ansi-bright-red',
        92: 'ansi-bright-green',
        93: 'ansi-bright-yellow',
        94: 'ansi-bright-blue',
        95: 'ansi-bright-magenta',
        96: 'ansi-bright-cyan',
        97: 'ansi-bright-white'
    };

    return classes[code] || null;
}

function resetTerminalAnsi() {
    terminalAnsiClasses = [];
}

function updateTerminalAnsi(codes) {
    if (codes.length === 0) {
        resetTerminalAnsi();
        return;
    }

    codes.forEach(code => {
        if (code === 0) {
            resetTerminalAnsi();
            return;
        }

        if (code === 22) {
            terminalAnsiClasses = terminalAnsiClasses
                .filter(item => item !== 'ansi-bold' && item !== 'ansi-dim');
            return;
        }

        if (code === 23) {
            terminalAnsiClasses = terminalAnsiClasses
                .filter(item => item !== 'ansi-italic');
            return;
        }

        if (code === 24) {
            terminalAnsiClasses = terminalAnsiClasses
                .filter(item => item !== 'ansi-underline');
            return;
        }

        if (code === 39) {
            terminalAnsiClasses = terminalAnsiClasses
                .filter(item => !item.startsWith('ansi-') || ['ansi-bold', 'ansi-dim', 'ansi-italic', 'ansi-underline'].includes(item));
            return;
        }

        const className =
            terminalAnsiClass(code);

        if (!className) {
            return;
        }

        if (code >= 30) {
            terminalAnsiClasses = terminalAnsiClasses
                .filter(item => !item.startsWith('ansi-') || ['ansi-bold', 'ansi-dim', 'ansi-italic', 'ansi-underline'].includes(item));
        }

        if (!terminalAnsiClasses.includes(className)) {
            terminalAnsiClasses.push(className);
        }
    });
}

function renderTerminalAnsi(text) {
    const ansiPattern =
        /\x1b\[([0-9;]*)m/g;
    let html =
        '';
    let lastIndex =
        0;
    let match;

    while ((match = ansiPattern.exec(text)) !== null) {
        html += renderTerminalText(text.slice(lastIndex, match.index));

        const codes =
            match[1]
                .split(';')
                .filter(Boolean)
                .map(value => Number.parseInt(value, 10))
                .filter(Number.isFinite);

        updateTerminalAnsi(codes);
        lastIndex = ansiPattern.lastIndex;
    }

    html += renderTerminalText(text.slice(lastIndex));

    return html;
}

function renderTerminalText(text) {
    if (!text) {
        return '';
    }

    const escaped =
        escapeHTML(text);

    if (terminalAnsiClasses.length === 0) {
        return escaped;
    }

    return `<span class="${terminalAnsiClasses.join(' ')}">${escaped}</span>`;
}

function appendTerminal(text) {
    if (scriptTerminal.textContent === 'Terminal output will appear here.') {
        scriptTerminal.textContent = '';
    }

    scriptTerminal.insertAdjacentHTML('beforeend', renderTerminalAnsi(text));
    scriptTerminal.scrollTop = scriptTerminal.scrollHeight;
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

function folderNode(name, depth) {
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

        parts.slice(0, -1).forEach((part, index) => {
            key =
                key ? `${key}/${part}` : part;

            if (!folders.has(key)) {
                const node =
                    folderNode(part, index);

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
                    'The running Rock-OS server does not support the script dashboard yet. Start from Go source with Website/run-go-server.cmd or build a new release binary.'
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
    showPreviewMode();

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

    if (outputStream) {
        outputStream.close();
    }

    showTerminalMode();
    scriptTerminal.textContent = '';
    resetTerminalAnsi();
    terminalMeta.textContent = 'Starting';
    setTerminalInputEnabled(false);
    setStatus('Starting script...', 'info');

    try {
        const response =
            await fetch('/api/scripts/run', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    id: selectedScript.id
                })
            });

        if (!response.ok) {
            throw new Error(await response.text());
        }

        const result =
            await response.json();

        activeSessionId = result.sessionId;
        terminalMeta.textContent = 'Running';
        setTerminalInputEnabled(true);
        setStatus('Script is running. Type into the terminal input if the script asks a question.', 'info');

        outputStream =
            new EventSource('/api/scripts/output/' + encodeURIComponent(activeSessionId));

        outputStream.onmessage = event => {
            appendTerminal(JSON.parse(event.data));
        };

        outputStream.addEventListener('done', () => {
            terminalMeta.textContent = 'Finished';
            setTerminalInputEnabled(false);
            setStatus('Script finished.', 'success');
            outputStream.close();
        });

        outputStream.onerror = () => {
            terminalMeta.textContent = 'Disconnected';
            setTerminalInputEnabled(false);
            setStatus('Terminal stream disconnected.', 'error');
        };
    }
    catch (err) {
        terminalMeta.textContent = 'Failed';
        setTerminalInputEnabled(false);
        setStatus(err.message, 'error');
    }
}

async function sendTerminalInput(event) {
    event.preventDefault();

    if (!activeSessionId || terminalInput.disabled) {
        return;
    }

    const input =
        terminalInput.value;
    const secretInput =
        terminalSecretInput.checked;

    terminalInput.value = '';

    if (secretInput) {
        appendTerminal('[hidden input sent]\n');
    }
    else {
        appendTerminal(input + '\n');
    }

    const response =
        await fetch('/api/scripts/input/' + encodeURIComponent(activeSessionId), {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                input: input + '\n'
            })
        });

    if (!response.ok) {
        setStatus(await response.text(), 'error');
    }
}

runScriptBtn.addEventListener('click', runSelectedScript);
toggleAllScriptsBtn.addEventListener('click', () => {
    setAllScriptFoldersExpanded(!allScriptFoldersExpanded());
});
terminalInputForm.addEventListener('submit', sendTerminalInput);
loadScripts();
