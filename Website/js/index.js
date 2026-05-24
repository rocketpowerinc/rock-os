const quoteElement =
    document.getElementById('dailyQuote');
const serverModeBanner =
    document.getElementById('serverModeBanner');
const serverModeTitle =
    document.getElementById('serverModeTitle');

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

loadQuote();
loadServerMode();
