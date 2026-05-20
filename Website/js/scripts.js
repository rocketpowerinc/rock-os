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

function appendTerminal(text) {
    if (scriptTerminal.textContent === 'Terminal output will appear here.') {
        scriptTerminal.textContent = '';
    }

    scriptTerminal.textContent += text;
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
    details.open = depth === 0;

    const summary =
        document.createElement('summary');
    summary.textContent = name;

    const children =
        document.createElement('div');
    children.className = 'script-tree-children';

    details.append(summary, children);

    return {
        details,
        children
    };
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

    setStatus(
        script.runnable ?
            'Review the script, then run it when ready.' :
            'This script is visible for review but does not run on this operating system.',
        script.runnable ? 'info' : 'warn'
    );

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
terminalInputForm.addEventListener('submit', sendTerminalInput);
loadScripts();
