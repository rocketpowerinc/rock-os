import { createMarkdownTabApp } from './wiki/markdown-tab.js';

async function profilesAreLocked() {

    try {
        const response =
            await fetch('/api/server/status?nocache=' + Date.now());

        if (!response.ok) {
            return true;
        }

        const status =
            await response.json();

        return status?.gitCrypt !== 'unlocked';
    }
    catch {
        return true;
    }
}

function renderLockedProfiles() {

    const sidebar =
        document.getElementById('sidebar');
    const resizer =
        document.getElementById('sidebarResizer');
    const expandButton =
        document.getElementById('expandSidebarBtn');
    const toc =
        document.getElementById('wikiToc');
    const content =
        document.getElementById('content');

    if (sidebar) {
        sidebar.style.display = 'none';
    }
    if (resizer) {
        resizer.style.display = 'none';
    }
    if (expandButton) {
        expandButton.style.display = 'none';
    }
    if (toc) {
        toc.innerHTML = '';
    }
    if (content) {
        content.classList.add('fullwidth');
        content.innerHTML = `
            <section class="profiles-locked-panel" aria-live="polite">
                <div class="profiles-lock-badge">Locked</div>
                <h1>Profiles Locked</h1>
                <p>Encrypted profile notes are locked with git-crypt. Unlock the repository to view Rocket, Kids, Prepper, and any future profiles.</p>
                <div class="profiles-lock-fake-button" aria-hidden="true">Profiles Locked</div>
                <pre><code>START-HERE\\Windows\\unlock-git-crypt.cmd</code></pre>
            </section>
        `;
    }
}

async function startProfiles() {

    if (await profilesAreLocked()) {
        renderLockedProfiles();
        return;
    }

    createMarkdownTabApp({
        key: 'profiles',
        label: 'Profiles',
        emptyLabel: 'profile files',
        searchStatusId: 'profilesSearchStatus',
        searchInputId: 'profilesSearchInput',
        refreshButtonId: 'refreshProfilesBtn',
        indexUrl: 'profiles-index.json',
        docApiUrl: '/api/profiles/doc',
        searchApiUrl: '/api/profiles/search',
        pathPrefix: 'profiles',
        directOpenPageName: 'profiles.html'
    });
}

startProfiles();
