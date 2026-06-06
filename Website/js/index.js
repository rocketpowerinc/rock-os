const quoteElement =
    document.getElementById('dailyQuote');
const serverModeBanner =
    document.getElementById('serverModeBanner');
const serverModeTitle =
    document.getElementById('serverModeTitle');
const launchPointsGrid =
    document.getElementById('launchPointsGrid');
const launchPointsSection =
    launchPointsGrid?.closest('.quick-start');

function hideProfileLaunchPoints() {
    if (launchPointsSection) {
        launchPointsSection.hidden =
            true;
    }
}

function profileFromDashboardPath(path) {
    const parts =
        String(path || '').split('/');

    if (
        parts.length >= 5 &&
        parts[0] === 'ENCRYPTED' &&
        parts[1] === 'dashboards' &&
        parts[2] === 'Profiles'
    ) {
        return parts[3];
    }

    return '';
}

function profileCard(profile) {
    const link =
        document.createElement('a');
    const icon =
        document.createElement('div');
    const info =
        document.createElement('div');
    const title =
        document.createElement('span');

    link.className =
        'profiles-card';
    link.dataset.profile =
        profile;
    link.href =
        `/ENCRYPTED/dashboards/Profiles/${encodeURIComponent(profile)}/`;

    icon.className =
        'profile-card-icon';
    info.className =
        'profiles-card-info';
    title.textContent =
        profile;

    info.append(title);
    link.append(icon, info);
    return link;
}

function profileOrderForSession(sessionName) {
    const normalized =
        String(sessionName || '').trim().toLowerCase();

    if (normalized === 'rocket') {
        return ['Rocket', 'Admin', 'Family', 'Kids', 'Prepper'];
    }
    if (normalized === 'admin') {
        return ['Admin', 'Prepper', 'Family', 'Kids'];
    }

    return ['Rocket', 'Family', 'Kids', 'Admin'];
}

function sortProfilesForSession(profiles, sessionName) {
    const profileOrder =
        profileOrderForSession(sessionName);
    const profileRank = profile => {
        if (!profileOrder.includes('Prepper') && profile === 'Prepper') {
            return profileOrder.length + 1;
        }

        const rank =
            profileOrder.indexOf(profile);

        return rank === -1
            ? profileOrder.length
            : rank;
    };

    profiles.sort((first, second) => {
        const rankCompare =
            profileRank(first) - profileRank(second);

        if (rankCompare !== 0) {
            return rankCompare;
        }
        return first.localeCompare(second, undefined, { sensitivity: 'base' });
    });
}

async function loadProfileLaunchPoints() {
    if (!launchPointsGrid) {
        return;
    }

    try {
        const [response, sessionsResponse] =
            await Promise.all([
                fetch('/dashboards-index.json?nocache=' + Date.now()),
                fetch('/api/sessions?nocache=' + Date.now())
            ]);

        if (!response.ok) {
            throw new Error('Could not load profile launch points');
        }

        const files =
            await response.json();
        const sessions =
            sessionsResponse.ok ? await sessionsResponse.json() : null;
        const profiles =
            Array.from(
                new Set(
                    (Array.isArray(files) ? files : [])
                        .map(file => profileFromDashboardPath(file?.path || file))
                        .filter(Boolean)
                )
            );

        sortProfilesForSession(profiles, sessions?.active);

        launchPointsGrid.replaceChildren(...profiles.map(profileCard));
        if (launchPointsSection) {
            launchPointsSection.hidden =
                profiles.length === 0;
        }
    }
    catch (err) {
        console.warn(err);
        hideProfileLaunchPoints();
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
        element.title =
            'Rock-OS has not recorded a successful Git sync yet.';
        return;
    }
    const relative =
        formatRelativeTime(serverLastSyncTimestamp);
    element.textContent =
        relative;
    element.title =
        `Last successful Git sync was ${relative}.`;
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
            element.title =
                'Server uptime is unavailable.';
            return;
        }
        const elapsed = Math.floor((Date.now() - lastFetchTime) / 1000);
        const uptime =
            formatUptime(serverUptimeSeconds + elapsed);
        element.textContent =
            uptime;
        element.title =
            `Rock-OS has been running for ${uptime}.`;
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
            serverModeBanner.title =
                'LAN mode: Rock-OS is reachable from trusted devices on your local network.';
        } else if (mode === 'local') {
            serverModeTitle.textContent = 'Host';
            serverModeBanner.title =
                'Host mode: Rock-OS is only reachable from this computer.';
        } else {
            serverModeTitle.textContent = 'Unknown';
            serverModeBanner.title =
                'Server mode is unavailable.';
        }
    }

    const cryptStatus = status?.gitCrypt || 'unknown';
    if (cryptStatus === 'unlocked') {
        loadProfileLaunchPoints();
    } else {
        hideProfileLaunchPoints();
    }
    const cryptElement = document.getElementById('encryptedFolderStatus');
    if (cryptElement) {
        if (cryptStatus === 'unlocked') {
            cryptElement.textContent = 'Unlocked';
            cryptElement.style.color = 'var(--success)';
            cryptElement.title =
                'Encrypted Rock-OS content is unlocked and available.';
        } else if (cryptStatus === 'locked') {
            cryptElement.textContent = 'Locked';
            cryptElement.style.color = '#ef4444';
            cryptElement.title =
                'Encrypted Rock-OS content is locked with git-crypt.';
        } else {
            cryptElement.textContent = 'Missing';
            cryptElement.style.color = 'var(--text-muted)';
            cryptElement.title =
                'Encrypted Rock-OS content status is unavailable.';
        }
    }

    const countElement = document.getElementById('wikiFilesCount');
    if (countElement) {
        countElement.textContent =
            typeof status?.wikiCount === 'number' ? `${status.wikiCount} Files` : '—';
        countElement.title =
            typeof status?.wikiCount === 'number'
                ? `${status.wikiCount} markdown documents are currently indexed.`
                : 'Wiki document count is unavailable.';
    }

    const scriptsCountElement = document.getElementById('scriptsFilesCount');
    if (scriptsCountElement) {
        scriptsCountElement.textContent =
            typeof status?.scriptsCount === 'number' ? `${status.scriptsCount} Files` : '—';
        scriptsCountElement.title =
            typeof status?.scriptsCount === 'number'
                ? `${status.scriptsCount} .cmd, .bat, .sh, and .ps1 scripts are currently available.`
                : 'Script count is unavailable.';
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
            uptimeElement.title =
                'Server uptime is unavailable.';
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
                checked === 0 ? '0 links exist.' : `${checked} links checked.`;
        }
    }
    catch (err) {
        console.warn(err);
        element.textContent =
            'Unavailable';
        element.style.color =
            'var(--text-muted)';
        element.title =
            'Link health is unavailable.';
    }
}

loadQuote();
loadServerMode();
loadLinkHealth();
