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
const scriptSearchInput =
    document.getElementById('scriptSearchInput');
const scriptSearchStatus =
    document.getElementById('scriptSearchStatus');

let selectedScript = null;
let allScripts = [];
let scriptRunInProgress = false;
const launchedScriptIds = new Set();
let scriptSearchQuery = '';
let scriptSearchResults = [];
let scriptSearchLoading = false;
let scriptSearchDebounceTimer = null;
let scriptSearchRequestId = 0;

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

function escapeRegExp(value) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function highlightSearchQuery(text, query) {
    const trimmedQuery =
        query.trim();

    if (!trimmedQuery) {
        return escapeHTML(text);
    }

    const matcher =
        new RegExp(escapeRegExp(trimmedQuery), 'gi');

    let html =
        '';
    let lastIndex =
        0;

    text.replace(matcher, (match, offset) => {
        html += escapeHTML(
            text.slice(lastIndex, offset)
        );
        html += `<mark class="search-match">${escapeHTML(match)}</mark>`;
        lastIndex =
            offset + match.length;

        return match;
    });

    html += escapeHTML(
        text.slice(lastIndex)
    );

    return html;
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

function scriptFromSearchResult(result) {
    return {
        id: result.id,
        name: result.name,
        path: result.path,
        runnable: result.runnable,
        platform: result.platform
    };
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

    if (!scripts.length) {
        scriptList.textContent =
            allScripts.length
                ? 'No scripts match your search.'
                : 'No scripts found in Website/scripts.';
        updateToggleAllScriptsButton();
        return;
    }

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

function scheduleScriptSearch() {
    const query =
        scriptSearchInput ?
            scriptSearchInput.value.trim() :
            '';

    scriptSearchQuery =
        query;

    if (!query) {
        scriptSearchLoading =
            false;
        scriptSearchResults =
            [];
        if (scriptSearchStatus) {
            scriptSearchStatus.textContent =
                '';
        }
        renderScriptTree(allScripts);
        return;
    }

    scriptSearchLoading =
        true;
    scriptSearchResults =
        [];
    renderScriptSearchResults();

    const requestId =
        ++scriptSearchRequestId;

    clearTimeout(scriptSearchDebounceTimer);
    scriptSearchDebounceTimer =
        setTimeout(() => {
            runScriptSearch(query, requestId);
        }, 250);
}

async function runScriptSearch(query, requestId) {
    try {
        const response =
            await fetch(
                `/api/scripts/search?q=${encodeURIComponent(query)}&nocache=${Date.now()}`
            );

        if (!response.ok) {
            throw new Error(
                `Search failed with HTTP ${response.status}`
            );
        }

        const payload =
            await response.json();

        if (
            requestId !== scriptSearchRequestId ||
            query !== scriptSearchQuery
        ) {
            return;
        }

        scriptSearchResults =
            Array.isArray(payload.results)
                ? payload.results.filter(result =>
                    result &&
                    typeof result.id === 'string'
                )
                : [];
        scriptSearchLoading =
            false;
        renderScriptSearchResults();
    }
    catch (err) {
        console.warn('Server script search failed:', err);

        if (
            requestId !== scriptSearchRequestId ||
            query !== scriptSearchQuery
        ) {
            return;
        }

        scriptSearchResults =
            fallbackScriptSearch(query);
        scriptSearchLoading =
            false;
        renderScriptSearchResults();
    }
}

function fallbackScriptSearch(query) {
    const normalizedQuery =
        query.toLowerCase();

    return allScripts
        .filter(script =>
            script.id.toLowerCase().includes(normalizedQuery) ||
            script.name.toLowerCase().includes(normalizedQuery) ||
            script.path.toLowerCase().includes(normalizedQuery)
        )
        .map(script => ({
            ...script,
            snippet: ''
        }));
}

function renderScriptSearchResults() {
    scriptList.textContent =
        '';

    const query =
        scriptSearchQuery.trim();

    if (!query) {
        renderScriptTree(allScripts);
        return;
    }

    if (scriptSearchStatus) {
        scriptSearchStatus.textContent =
            scriptSearchLoading
                ? 'Searching...'
                : `${scriptSearchResults.length} result${scriptSearchResults.length === 1 ? '' : 's'}`;
    }

    if (!scriptSearchResults.length && !scriptSearchLoading) {
        scriptList.textContent =
            'No scripts match your search.';
        updateToggleAllScriptsButton();
        return;
    }

    scriptSearchResults.forEach(result => {
        const script =
            scriptFromSearchResult(result);
        const button =
            scriptButton(script);

        button.classList.toggle(
            'active',
            selectedScript &&
            selectedScript.id === script.id
        );
        button.innerHTML =
            `${escapeHTML(result.name)}<span class="search-result-meta">${escapeHTML(result.path)}</span>`;

        scriptList.appendChild(button);

        if (result.snippet) {
            const snippet =
                document.createElement('div');

            snippet.className =
                'search-result-snippet';
            snippet.innerHTML =
                highlightSearchQuery(
                    result.snippet,
                    query
                );

            scriptList.appendChild(snippet);
        }
    });

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

        allScripts =
            scripts;

        scheduleScriptSearch();
    }
    catch (err) {
        setStatus(err.message, 'error');
    }
}

async function selectScript(script) {
    selectedScript = script;
    runScriptBtn.disabled =
        !script.runnable ||
        launchedScriptIds.has(script.id);
    runScriptBtn.classList.toggle(
        'script-run-used',
        launchedScriptIds.has(script.id)
    );

    document.querySelectorAll('.script-tree-file')
        .forEach(item => {
            item.classList.toggle('active', item.dataset.scriptId === script.id);
        });

    const warning =
        scriptPlatformWarning(script);

    if (warning) {
        setStatus(warning, 'warn');
    }
    else if (launchedScriptIds.has(script.id)) {
        setStatus(
            'This script has already been launched from this page session.',
            'success'
        );
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
    if (
        scriptRunInProgress ||
        !selectedScript ||
        !selectedScript.runnable
    ) {
        return;
    }

    scriptRunInProgress =
        true;
    runScriptBtn.disabled =
        true;
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

        launchedScriptIds.add(selectedScript.id);
        runScriptBtn.classList.add('script-run-used');
        setStatus(
            'Script opened in your OS terminal. Reload this page to run another copy of this script.',
            'warn'
        );
    }
    catch (err) {
        setStatus(err.message, 'error');
    }
    finally {
        scriptRunInProgress =
            false;

        if (selectedScript) {
            runScriptBtn.disabled =
                !selectedScript.runnable ||
                launchedScriptIds.has(selectedScript.id);
            runScriptBtn.classList.toggle(
                'script-run-used',
                launchedScriptIds.has(selectedScript.id)
            );
        }
    }
}

runScriptBtn.addEventListener('click', runSelectedScript);
toggleAllScriptsBtn.addEventListener('click', () => {
    setAllScriptFoldersExpanded(!allScriptFoldersExpanded());
});

if (scriptSearchInput) {
    scriptSearchInput.addEventListener('input', scheduleScriptSearch);
}

loadScripts();
