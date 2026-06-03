const quoteElement =
    document.getElementById('dailyQuote');
const serverModeBanner =
    document.getElementById('serverModeBanner');
const serverModeTitle =
    document.getElementById('serverModeTitle');
const launchPointsGrid =
    document.getElementById('launchPointsGrid');
const sessionSelect =
    document.getElementById('sessionSelect');

function launchPointLink(point, options = {}) {
    const link =
        document.createElement('a');
    const title =
        document.createElement('strong');
    const description =
        document.createElement('small');
    const target =
        options.useCardPath && point.path ? point.path : point.href;
    const href =
        new URL(target, window.location.origin);

    if (href.protocol !== 'http:' && href.protocol !== 'https:') {
        return null;
    }

    link.className =
        'launch-link';
    link.href =
        href.href;
    title.textContent =
        point.title;
    description.textContent =
        point.description;

    if (href.origin !== window.location.origin) {
        link.target =
            '_blank';
        link.rel =
            'noopener noreferrer';
    }

    link.append(title, description);
    return link;
}

async function loadLockedLaunchPoints() {
    if (!launchPointsGrid) {
        return;
    }

    try {
        const response =
            await fetch('/api/launch-points?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load launch points');
        }

        const points =
            await response.json();
        const links =
            Array.isArray(points)
                ? points.map(point => launchPointLink(point, { useCardPath: true })).filter(Boolean)
                : [];

        launchPointsGrid.replaceChildren();
        if (links.length === 0) {
            launchPointsGrid.innerHTML =
                '<p class="launch-points-status">Add .md files under Website/launch-point-cards-locked to create locked-mode launch cards.</p>';
            return;
        }

        launchPointsGrid.append(...links);
    }
    catch (err) {
        console.warn(err);
        launchPointsGrid.innerHTML =
            '<p class="launch-points-status">Locked-mode launch points are unavailable. Restart Rock-OS with the latest release binary.</p>';
    }
}

function parseQuoteBullets(markdown) {

    return markdown
        .split(/\r?\n/)
        .map(line => line.trim())
        .filter(line => /^[-*]\s+/.test(line))
        .map(line => line.replace(/^[-*]\s+/, '').trim())
        .filter(Boolean);
}

function showRandomQuote(quotes) {

    if (!quoteElement || quotes.length === 0) {
        quoteElement?.closest('.quote-panel')?.classList.add('is-empty');
        return;
    }

    const quote =
        quotes[Math.floor(Math.random() * quotes.length)];

    quoteElement.textContent =
        quote.replace(/^["'\u2018-\u201D]+|["'\u2018-\u201D]+$/g, '');
}

async function loadQuote() {

    if (!quoteElement) {
        return;
    }

    try {
        const response =
            await fetch('quotes.md?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load quotes.md');
        }

        const markdown =
            await response.text();

        showRandomQuote(
            parseQuoteBullets(markdown)
        );
    }
    catch (err) {
        console.warn(err);
        quoteElement.closest('.quote-panel')?.classList.add('is-empty');
    }
}

let serverUptimeSeconds = null;
let lastFetchTime = null;
let uptimeInterval = null;
let serverLastSyncTimestamp = null;

function formatUptime(seconds) {
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;

    if (d > 0) {
        return `${d}d ${h}h ${m}m`;
    }
    if (h > 0) {
        return `${h}h ${m}m ${s}s`;
    }
    if (m > 0) {
        return `${m}m ${s}s`;
    }
    return `${s}s`;
}

function formatRelativeTime(unixTimestamp) {
    if (!unixTimestamp || unixTimestamp <= 0) {
        return '—';
    }
    const nowSecs = Math.floor(Date.now() / 1000);
    const diff = nowSecs - unixTimestamp;

    if (diff < 0) {
        return 'Just now';
    }
    if (diff < 60) {
        return 'Just now';
    }

    const minutes = Math.floor(diff / 60);
    if (minutes < 60) {
        return `${minutes}m ago`;
    }

    const hours = Math.floor(minutes / 60);
    if (hours < 24) {
        return `${hours}h ago`;
    }

    const days = Math.floor(hours / 24);
    return `${days}d ago`;
}

function updateLastSyncDisplay() {
    const element = document.getElementById('serverLastSync');
    if (!element) return;
    if (serverLastSyncTimestamp === null || serverLastSyncTimestamp <= 0) {
        element.textContent = '—';
        return;
    }
    element.textContent = formatRelativeTime(serverLastSyncTimestamp);
}

function startUptimeTicker() {
    if (uptimeInterval) {
        clearInterval(uptimeInterval);
    }
    const element = document.getElementById('serverUptime');
    if (!element) return;

    function tick() {
        if (serverUptimeSeconds === null || lastFetchTime === null) {
            element.textContent = '—';
            return;
        }
        const elapsed = Math.floor((Date.now() - lastFetchTime) / 1000);
        element.textContent = formatUptime(serverUptimeSeconds + elapsed);
        updateLastSyncDisplay();
    }

    tick();
    uptimeInterval = setInterval(tick, 1000);
}

function renderServerMode(status) {

    if (serverModeBanner && serverModeTitle) {
        const mode =
            status?.mode === 'lan' ? 'lan' :
                status?.mode === 'local' ? 'local' :
                    'unknown';

        serverModeBanner.dataset.mode =
            mode;

        if (mode === 'lan') {
            serverModeTitle.textContent = 'LAN';
        } else if (mode === 'local') {
            serverModeTitle.textContent = 'Host';
        } else {
            serverModeTitle.textContent = 'Unknown';
        }
    }

    const cryptStatus = status?.gitCrypt || 'unknown';
    if (cryptStatus !== 'unlocked') {
        loadLockedLaunchPoints();
    }
    const cryptElement = document.getElementById('encryptedFolderStatus');
    if (cryptElement) {
        if (cryptStatus === 'unlocked') {
            cryptElement.textContent = 'Unlocked';
            cryptElement.style.color = 'var(--success)';
        } else if (cryptStatus === 'locked') {
            cryptElement.textContent = 'Locked';
            cryptElement.style.color = '#ef4444';
        } else {
            cryptElement.textContent = 'Missing';
            cryptElement.style.color = 'var(--text-muted)';
        }
    }

    const countElement = document.getElementById('wikiFilesCount');
    if (countElement) {
        countElement.textContent =
            typeof status?.wikiCount === 'number' ? `${status.wikiCount} Files` : '—';
    }

    const scriptsCountElement = document.getElementById('scriptsFilesCount');
    if (scriptsCountElement) {
        scriptsCountElement.textContent =
            typeof status?.scriptsCount === 'number' ? `${status.scriptsCount} Files` : '—';
    }

    const commitElement = document.getElementById('serverCommit');
    if (commitElement) {
        const commit =
            typeof status?.commit === 'string' ? status.commit.trim() : '';

        if (/^[0-9a-f]{40}$/i.test(commit)) {
            commitElement.textContent = commit.slice(0, 7);
            commitElement.href =
                `https://github.com/rocketpowerinc/rock-os/commit/${commit}`;
            commitElement.title =
                `Rock-OS is running commit ${commit}.`;
        } else {
            commitElement.textContent = 'Unavailable';
            commitElement.removeAttribute('href');
            commitElement.title =
                'Commit metadata is unavailable when Rock-OS is not running from a Git clone.';
        }
    }

    const uptimeElement = document.getElementById('serverUptime');
    if (uptimeElement) {
        if (status && typeof status.uptime === 'number') {
            serverUptimeSeconds = status.uptime;
            lastFetchTime = Date.now();
            serverLastSyncTimestamp = typeof status.lastSync === 'number' ? status.lastSync : null;
            startUptimeTicker();
        } else {
            serverUptimeSeconds = null;
            lastFetchTime = null;
            serverLastSyncTimestamp = null;
            uptimeElement.textContent = '—';
            updateLastSyncDisplay();
        }
    }
}

async function loadServerMode() {

    if (!serverModeBanner) {
        return;
    }

    try {
        const response =
            await fetch('/api/server/status?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load server status');
        }

        renderServerMode(
            await response.json()
        );
    }
    catch (err) {
        console.warn(err);
        renderServerMode(null);
    }
}

async function loadLinkHealth() {
    const element =
        document.getElementById('linkHealthStatus');

    if (!element) {
        return;
    }

    try {
        const response =
            await fetch('/api/health/links?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load link health');
        }

        const report =
            await response.json();

        const broken =
            Number(report?.broken || 0);
        const checked =
            Number(report?.checked || 0);

        if (broken > 0) {
            element.textContent =
                `${broken} Broken`;
            element.style.color =
                '#ef4444';
            element.title =
                `${checked} links checked. Open /api/health/links for details.`;
        } else {
            element.textContent =
                'OK';
            element.style.color =
                'var(--success)';
            element.title =
                `${checked} links checked.`;
        }
    }
    catch (err) {
        console.warn(err);
        element.textContent =
            'Unavailable';
        element.style.color =
            'var(--text-muted)';
    }
}

function renderSessionSelect(config) {
    if (!sessionSelect) {
        return;
    }

    const sessions =
        Array.isArray(config?.sessions) ? config.sessions : [];

    sessionSelect.replaceChildren();

    if (sessions.length === 0) {
        const option =
            document.createElement('option');
        option.value =
            '';
        option.textContent =
            'Sessions unavailable';
        sessionSelect.append(option);
        sessionSelect.disabled =
            true;
        return;
    }

    for (const session of sessions) {
        const name =
            String(session?.name || '').trim();
        if (!name) {
            continue;
        }

        const option =
            document.createElement('option');
        option.value =
            name;
        option.textContent =
            name;
        if (session.description) {
            option.title =
                session.description;
        }
        sessionSelect.append(option);
    }

    sessionSelect.value =
        config?.active || sessions[0]?.name || '';
    sessionSelect.disabled =
        false;
    sessionSelect.title =
        'Choose the active Rock-OS dashboard session.';
}

async function loadSessions() {
    if (!sessionSelect) {
        return;
    }

    try {
        const response =
            await fetch('/api/sessions?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load dashboard sessions');
        }

        renderSessionSelect(
            await response.json()
        );
    }
    catch (err) {
        console.warn(err);
        renderSessionSelect(null);
    }
}

async function updateActiveSession(active) {
    const response =
        await fetch('/api/sessions', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Rock-OS-Requested': 'true'
            },
            body: JSON.stringify({ active })
        });

    if (!response.ok) {
        throw new Error(
            (await response.text()).trim() ||
            `Session update failed with HTTP ${response.status}`
        );
    }

    return response.json();
}

if (sessionSelect) {
    sessionSelect.disabled =
        true;
    sessionSelect.addEventListener('change', async () => {
        const nextSession =
            sessionSelect.value;
        if (!nextSession) {
            return;
        }

        sessionSelect.disabled =
            true;

        try {
            renderSessionSelect(
                await updateActiveSession(nextSession)
            );
        }
        catch (err) {
            console.warn(err);
            window.alert(`Rock-OS could not change the dashboard session.\n\n${err.message}`);
            await loadSessions();
        }
    });
}

loadQuote();
loadServerMode();
loadLinkHealth();
loadSessions();
