// Shared "locked landing" shown while git-crypt content is locked.
//
// Instead of a generic "content locked" message, each page shows its matching
// card from Website/launch-point-locked. Those markdown files are never
// encrypted and are exposed (parsed) through /api/launch-points; each card's
// link points at a page (e.g. ../dashboards.html), so we match a card to the
// current page by that link's filename. If the matching card can't be loaded
// for any reason, we fall back to the original generic locked message so the
// page never ends up blank.

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function hrefBasename(href) {
    try {
        const parts = new URL(href, window.location.origin)
            .pathname
            .split('/')
            .filter(Boolean);
        return (parts.pop() || '').toLowerCase();
    } catch {
        return '';
    }
}

async function findLaunchPoint(pageFileName) {
    const target = String(pageFileName || '').toLowerCase();
    if (!target) {
        return null;
    }

    try {
        const response = await fetch('/api/launch-points?nocache=' + Date.now());
        if (!response.ok) {
            return null;
        }

        const points = await response.json();
        if (!Array.isArray(points)) {
            return null;
        }

        return points.find(point => point && hrefBasename(point.href) === target) || null;
    } catch {
        return null;
    }
}

function lockedPanelHTML(title, body) {
    return `
        <section class="profiles-locked-panel" aria-live="polite">
            <div class="profiles-lock-badge">Locked</div>
            <h1>${escapeHtml(title)}</h1>
            <p>${escapeHtml(body)}</p>
            <p class="profiles-lock-hint">Unlock the repository to open this content:</p>
            <pre><code>START-HERE\\Windows\\unlock-git-crypt.cmd</code></pre>
            <button id="lockedLandingRefreshBtn" class="command-button primary" type="button">Refresh</button>
        </section>
    `;
}

// Renders the locked landing into contentEl. pageFileName is the page this
// content belongs to (e.g. "dashboards.html", "wiki.html"), used to pick the
// matching launch-point-locked card.
export async function renderLockedLanding(contentEl, pageFileName) {
    if (!contentEl) {
        return;
    }

    const point = await findLaunchPoint(pageFileName);

    const title = point ? point.title : 'Encrypted Content Locked';
    const body = point
        ? point.description
        : 'Dashboards, menu content, and scripts are locked with git-crypt. Unlock the repository to use Rock-OS content.';

    contentEl.classList.add('fullwidth');
    contentEl.innerHTML = lockedPanelHTML(title, body);

    const refreshButton = document.getElementById('lockedLandingRefreshBtn');
    if (refreshButton) {
        refreshButton.addEventListener('click', () => window.location.reload());
    }
}
