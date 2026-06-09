import { pullLatestRockOSAndReload, warnLiveUpdateFailed } from './server-refresh.js';
import { renderLockedContent } from './locked-content.js';
import {
    currentProfileWorkspaceName,
    renderMissingProfileContext,
    renderProfileWorkspaceNav
} from './profile-workspace.js';

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
const refreshScriptsBtn =
    document.getElementById('refreshScriptsBtn');
const scriptSearchInput =
    document.getElementById('scriptSearchInput');
const scriptSearchStatus =
    document.getElementById('scriptSearchStatus');
const activeProfile =
    currentProfileWorkspaceName();
const pinnedScriptsStorageKey =
    `rock-os-${activeProfile || 'profile'}-pinned-scripts`;

let selectedScript = null;
let allScripts = [];
let scriptRunInProgress = false;
const launchedScriptIds = new Set();
let pinnedScriptIds = new Set();
let scriptSearchQuery = '';
let scriptSearchResults = [];
let scriptSearchLoading = false;
let scriptSearchDebounceTimer = null;
let scriptSearchRequestId = 0;

loadPinnedScripts();

if (activeProfile) {
    renderProfileWorkspaceNav(activeProfile);
    document.title =
        `Rock-OS ${activeProfile} Scripts`;
    const tabLabel =
        'Profile Based Scripts';
    const sidebarHeading =
        document.querySelector('.sidebar-header h3');
    const pageHeading =
        document.querySelector('.script-workbench-header h1');
    if (sidebarHeading) {
        sidebarHeading.textContent =
            tabLabel;
    }
    if (pageHeading) {
        pageHeading.textContent =
            tabLabel;
    }
}

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

function loadPinnedScripts() {
    try {
        const saved =
            JSON.parse(
                localStorage.getItem(pinnedScriptsStorageKey) || '[]'
            );

        pinnedScriptIds =
            new Set(
                Array.isArray(saved)
                    ? saved.filter(id => typeof id === 'string')
                    : []
            );
    }
    catch (err) {
        console.warn('Could not load pinned scripts:', err);
        pinnedScriptIds =
            new Set();
    }
}

function savePinnedScripts() {
    try {
        localStorage.setItem(
            pinnedScriptsStorageKey,
            JSON.stringify(Array.from(pinnedScriptIds))
        );
    }
    catch (err) {
        console.warn('Could not save pinned scripts:', err);
    }
}

function isScriptPinned(id) {
    return pinnedScriptIds.has(id);
}

function toggleScriptPin(id) {
    if (isScriptPinned(id)) {
        pinnedScriptIds.delete(id);
    }
    else {
        pinnedScriptIds.add(id);
    }

    savePinnedScripts();
    renderCurrentScriptList();
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

    const pinButton =
        document.createElement('span');
    pinButton.className =
        'pin-toggle';
    pinButton.classList.toggle(
        'active',
        isScriptPinned(script.id)
    );
    pinButton.setAttribute(
        'role',
        'button'
    );
    pinButton.setAttribute(
        'aria-label',
        isScriptPinned(script.id) ? 'Unpin script' : 'Pin script'
    );
    pinButton.title =
        isScriptPinned(script.id) ? 'Unpin script' : 'Pin script';
    pinButton.textContent =
        '★';

    pinButton.addEventListener('click', event => {
        event.preventDefault();
        event.stopPropagation();
        toggleScriptPin(script.id);
    });
    pinButton.addEventListener('keydown', event => {
        if (event.key !== 'Enter' && event.key !== ' ') {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        toggleScriptPin(script.id);
    });
    pinButton.tabIndex =
        0;

    button.append(name, pinButton);
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

    if (folderParts.includes('mac') || folderParts.includes('macos')) {
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

function folderNode(name, folderPath) {
    const details =
        document.createElement('details');
    details.className = 'script-tree-folder';
    details.open = false;

    const summary =
        document.createElement('summary');

    const icon = iconElement(folderIconName(folderPath));
    const label = document.createElement('span');
    label.className = 'folder-label';
    label.textContent = name;

    summary.append(icon, label);

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
                : `No scripts found in the ${activeProfile} profile workspace.`;
        updateToggleAllScriptsButton();
        return;
    }

    renderPinnedScripts(scripts);

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
                    folderNode(part, key);

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

function renderCurrentScriptList() {
    if (scriptSearchQuery.trim()) {
        if (scriptSearchResults.length || !scriptSearchLoading) {
            renderScriptSearchResults();
        }
        return;
    }

    renderScriptTree(allScripts);
}

function pinnedScripts(scripts) {
    return scripts
        .filter(script => isScriptPinned(script.id))
        .sort((first, second) =>
            first.name.localeCompare(second.name, undefined, {
                sensitivity: 'base'
            })
        );
}

function renderPinnedScripts(scripts) {
    const pinned =
        pinnedScripts(scripts);

    if (!pinned.length) {
        return;
    }

    const section =
        document.createElement('section');
    section.className =
        'pinned-scripts';
    section.setAttribute('aria-label', 'Pinned scripts');

    const title =
        document.createElement('div');
    title.className =
        'pinned-scripts-title';
    title.textContent =
        'Favorites';

    section.appendChild(title);

    pinned.forEach(script => {
        const button =
            scriptButton(script);

        button.classList.add('pinned-script-link');
        button.classList.toggle(
            'active',
            selectedScript &&
            selectedScript.id === script.id
        );

        section.appendChild(button);
    });

    scriptList.appendChild(section);
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
                `/api/scripts/search?profile=${encodeURIComponent(activeProfile)}&q=${encodeURIComponent(query)}&nocache=${Date.now()}`
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
        const meta =
            document.createElement('span');
        meta.className =
            'search-result-meta';
        meta.textContent =
            result.path;
        button.appendChild(meta);

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

function renderScriptsLocked() {
    const sidebar = document.getElementById('sidebar');
    const resizer = document.getElementById('sidebarResizer');
    const expandButton = document.getElementById('expandSidebarBtn');
    const runButton = document.getElementById('runScriptBtn');

    if (sidebar) sidebar.style.display = 'none';
    if (resizer) resizer.style.display = 'none';
    if (expandButton) expandButton.style.display = 'none';
    if (runButton) runButton.style.display = 'none';

    const stage = document.querySelector('.script-stage');
    if (stage) {
        renderLockedContent(stage, 'Scripts');
    }
}

async function loadScripts() {
    if (!activeProfile) {
        renderMissingProfileContext('Scripts');
        return;
    }

    try {
        const response =
            await fetch(`/api/scripts?profile=${encodeURIComponent(activeProfile)}`);

        if (!response.ok) {
            if (response.status === 423) {
                renderScriptsLocked();
                return;
            }

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
            await fetch(
                `/api/scripts/content?profile=${encodeURIComponent(activeProfile)}&id=${encodeURIComponent(script.id)}`
            );

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
            await fetch(`/api/scripts/run?profile=${encodeURIComponent(activeProfile)}`, {
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

if (refreshScriptsBtn) {
    refreshScriptsBtn.addEventListener('click', async () => {
        refreshScriptsBtn.disabled = true;
        refreshScriptsBtn.classList.add('is-refreshing');
        try {
            if (await pullLatestRockOSAndReload()) {
                return;
            }

            await loadScripts();
        } catch (err) {
            warnLiveUpdateFailed(err);
            await loadScripts();
        } finally {
            refreshScriptsBtn.disabled = false;
            refreshScriptsBtn.classList.remove('is-refreshing');
        }
    });
}

if (scriptSearchInput) {
    scriptSearchInput.addEventListener('input', scheduleScriptSearch);
}

loadScripts();
