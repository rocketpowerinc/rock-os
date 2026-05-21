const quoteElement =
    document.getElementById('dailyQuote');
const serverModeBanner =
    document.getElementById('serverModeBanner');
const serverModeTitle =
    document.getElementById('serverModeTitle');
const serverModeText =
    document.getElementById('serverModeText');

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

    if (!serverModeBanner || !serverModeTitle || !serverModeText) {
        return;
    }

    const mode =
        status?.mode === 'lan' ? 'lan' :
            status?.mode === 'local' ? 'local' :
                'unknown';

    serverModeBanner.dataset.mode =
        mode;

    if (mode === 'lan') {
        serverModeTitle.textContent =
            'LAN Mode';
        serverModeText.textContent =
            'Other trusted devices on this local network can reach Rock-OS. Avoid LAN mode on public or untrusted Wi-Fi.';
        return;
    }

    if (mode === 'local') {
        serverModeTitle.textContent =
            'Local Mode';
        serverModeText.textContent =
            'Only this computer can reach Rock-OS. Use LAN mode only when you intentionally want trusted home devices to connect.';
        return;
    }

    serverModeTitle.textContent =
        'Mode Unavailable';
    serverModeText.textContent =
        'Start Rock-OS from the Go server to display whether it is running local-only or on your LAN.';
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
