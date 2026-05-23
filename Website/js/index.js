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
            cryptElement.style.color = 'var(--accent)';
        } else {
            cryptElement.textContent = 'Missing';
            cryptElement.style.color = 'var(--text-muted)';
        }
    }

    const countElement = document.getElementById('markdownFilesCount');
    if (countElement) {
        countElement.textContent =
            typeof status?.markdownCount === 'number' ? `${status.markdownCount} Files` : '—';
    }

    const scriptsCountElement = document.getElementById('scriptsFilesCount');
    if (scriptsCountElement) {
        scriptsCountElement.textContent =
            typeof status?.scriptsCount === 'number' ? `${status.scriptsCount} Files` : '—';
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
