const ids = {
    checked: document.getElementById('healthChecked'),
    ok: document.getElementById('healthOk'),
    broken: document.getElementById('healthBroken'),
    external: document.getElementById('healthExternal'),
    skipped: document.getElementById('healthSkipped'),
    status: document.getElementById('healthStatus'),
    list: document.getElementById('brokenLinksList'),
    refresh: document.getElementById('refreshLinkHealthBtn')
};

function escapeHtml(value) {
    return String(value ?? '')
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#039;');
}

function setText(element, value) {
    if (element) {
        element.textContent = value;
    }
}

function renderSummary(report) {
    setText(ids.checked, String(report?.checked ?? 0));
    setText(ids.ok, String(report?.ok ?? 0));
    setText(ids.broken, String(report?.broken ?? 0));
    setText(ids.external, String(report?.external ?? 0));
    setText(ids.skipped, String(report?.skipped ?? 0));
}

function renderBrokenLinks(report) {
    if (!ids.list) {
        return;
    }

    const brokenItems =
        Array.isArray(report?.items) ?
            report.items.filter(item => item.status === 'broken') :
            [];

    if (brokenItems.length === 0) {
        ids.list.innerHTML =
            '<p class="health-empty">No broken local links found.</p>';
        return;
    }

    ids.list.innerHTML =
        brokenItems.map(item => `
            <article class="health-item">
                <div class="health-item-main">
                    <h3>${escapeHtml(item.label || item.href)}</h3>
                    <p>${escapeHtml(item.reason || 'Broken local link')}</p>
                </div>
                <dl>
                    <div>
                        <dt>Source</dt>
                        <dd>${escapeHtml(item.source)}</dd>
                    </div>
                    <div>
                        <dt>Link</dt>
                        <dd>${escapeHtml(item.href)}</dd>
                    </div>
                    <div>
                        <dt>Target</dt>
                        <dd>${escapeHtml(item.target || 'Not resolved')}</dd>
                    </div>
                </dl>
            </article>
        `).join('');
}

async function loadLinkHealth() {
    setText(ids.status, 'Scanning...');
    if (ids.refresh) {
        ids.refresh.disabled = true;
    }

    try {
        const response =
            await fetch('/api/health/links?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error(`Link health scan failed with HTTP ${response.status}`);
        }

        const report =
            await response.json();

        renderSummary(report);
        renderBrokenLinks(report);

        const broken =
            Number(report?.broken || 0);

        setText(
            ids.status,
            broken > 0 ? `${broken} broken local link${broken === 1 ? '' : 's'}` : 'All local links clean'
        );
    }
    catch (err) {
        console.warn(err);
        renderSummary(null);
        setText(ids.status, 'Scan unavailable');
        if (ids.list) {
            ids.list.innerHTML =
                '<p class="health-empty">Could not run the link health scan. Start Rock-OS from the Go server and try again.</p>';
        }
    }
    finally {
        if (ids.refresh) {
            ids.refresh.disabled = false;
        }
    }
}

ids.refresh?.addEventListener('click', loadLinkHealth);
loadLinkHealth();
