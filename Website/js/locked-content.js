function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

export function renderLockedContent(contentEl, label = 'Encrypted Content') {
    if (!contentEl) {
        return;
    }

    contentEl.classList.add('fullwidth');
    contentEl.innerHTML = `
        <section class="profiles-locked-panel" aria-live="polite">
            <div class="profiles-lock-badge">Locked</div>
            <h1>${escapeHtml(label)} Locked</h1>
            <p>Profile workspaces and dashboards are locked with git-crypt.</p>
            <p class="profiles-lock-hint">Unlock the repository to open this content:</p>
            <pre><code>START-HERE\\Windows\\unlock-git-crypt.cmd</code></pre>
            <button id="lockedContentRefreshBtn" class="command-button primary" type="button">Refresh</button>
        </section>
    `;

    document.getElementById('lockedContentRefreshBtn')
        ?.addEventListener('click', () => window.location.reload());
}

