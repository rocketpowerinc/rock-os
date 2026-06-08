import { createMarkdownTabApp } from './wiki/markdown-tab.js';

const workspaceSections = [
    { key: 'dashboards', label: 'Dashboards' },
    { key: 'bookmarks', label: 'Bookmarks' },
    { key: 'cheatsheets', label: 'Cheatsheets' },
    { key: 'dotfiles', label: 'Dotfiles' },
    { key: 'bootstraps', label: 'Bootstraps' },
    { key: 'scripts', label: 'Scripts' },
    { key: 'wiki', label: 'Wiki' }
];

function encodePathSegment(value) {
    return encodeURIComponent(String(value || '').trim());
}

function encodeProfilePath(profile) {
    return String(profile || '')
        .split('/')
        .filter(Boolean)
        .map(encodePathSegment)
        .join('/');
}

export function currentProfileWorkspaceName() {
    const params =
        new URLSearchParams(window.location.search);
    const queryProfile =
        String(params.get('profile') || '').trim();

    if (queryProfile) {
        return queryProfile;
    }

    const parts =
        window.location.pathname
            .split('/')
            .filter(Boolean)
            .map(part => decodeURIComponent(part));
    const profilesIndex =
        parts.findIndex((part, index) =>
            part === 'Sessions' &&
            parts[index - 1] === 'ENCRYPTED'
        );

    if (profilesIndex < 0) {
        return '';
    }

    const workspaceSectionKeys =
        new Set(workspaceSections.map(section => section.key));
    const profileParts = [];
    for (const part of parts.slice(profilesIndex + 1)) {
        if (workspaceSectionKeys.has(part) || part === 'index.html') {
            break;
        }
        profileParts.push(part);
    }

    return profileParts.join('/');
}

export function renderProfileWorkspaceNav(profile = currentProfileWorkspaceName()) {
    if (!profile) {
        return;
    }

    let nav =
        document.getElementById('profileWorkspaceNav');
    if (!nav) {
        nav =
            document.createElement('nav');
        nav.id =
            'profileWorkspaceNav';
        nav.className =
            'profile-workspace-nav';
        nav.setAttribute('aria-label', `${profile} profile workspace`);

        const navbar =
            document.querySelector('.navbar');
        navbar?.insertAdjacentElement('afterend', nav);
    }
    if (!nav) {
        return;
    }

    const profilePath =
        `/ENCRYPTED/Sessions/${encodeProfilePath(profile)}/`;
    const currentPath =
        window.location.pathname.toLowerCase();
    const profileDashboardPath =
        `/encrypted/sessions/${encodeProfilePath(profile).toLowerCase()}/dashboards/`;
    const links = [
        {
            key: 'overview',
            label: 'Hub',
            href: profilePath
        },
        ...workspaceSections.map(section => ({
            ...section,
            href: section.key === 'dashboards'
                ? `/dashboards.html?profile=${encodeURIComponent(profile)}`
                : `/${section.key}.html?profile=${encodeURIComponent(profile)}`
        }))
    ];

    nav.replaceChildren(
        ...links.map(link => {
            const anchor =
                document.createElement('a');
            const active =
                link.key === 'overview'
                    ? currentPath.includes('/encrypted/sessions/') && !currentPath.includes('/dashboards/')
                    : link.key === 'dashboards'
                        ? currentPath.endsWith('/dashboards.html') || currentPath.includes(profileDashboardPath)
                        : currentPath.endsWith(`/${link.key}.html`);

            anchor.href =
                link.href;
            anchor.textContent =
                link.label;
            anchor.classList.toggle('is-active', active);
            if (active) {
                anchor.setAttribute('aria-current', 'page');
            }
            return anchor;
        })
    );
}

export function renderMissingProfileContext(label) {
    const sidebar =
        document.getElementById('sidebar');
    const resizer =
        document.getElementById('sidebarResizer');
    const expandButton =
        document.getElementById('expandSidebarBtn');
    const content =
        document.getElementById('content');

    if (sidebar) sidebar.style.display = 'none';
    if (resizer) resizer.style.display = 'none';
    if (expandButton) expandButton.style.display = 'none';
    if (content) {
        content.classList.add('fullwidth');
        content.innerHTML = `
            <section class="profiles-locked-panel">
                <div class="profiles-lock-badge">Profile Required</div>
                <h1>${label}</h1>
                <p>Open this section from a profile workspace.</p>
                <a class="command-button primary" href="/index.html">Open Home</a>
            </section>
        `;
    }
}

export function startProfileMarkdownSection(config) {
    const profile =
        currentProfileWorkspaceName();
    if (!profile) {
        renderMissingProfileContext(config.label);
        return;
    }

    renderProfileWorkspaceNav(profile);

    const heading =
        document.querySelector('.sidebar-header h3');
    const contentHeading =
        document.querySelector('#content h1');
    if (heading) {
        heading.textContent =
            `${profile} ${config.label}`;
    }
    if (contentHeading) {
        contentHeading.textContent =
            `${profile} ${config.label}`;
    }
    document.title =
        `Rock-OS ${profile} ${config.label}`;

    const encodedProfile =
        encodeURIComponent(profile);
    const section =
        config.key;

    createMarkdownTabApp({
        key: `${profile}-${section}`,
        label: config.label,
        emptyLabel: config.emptyLabel,
        searchStatusId: config.searchStatusId,
        searchInputId: config.searchInputId,
        refreshButtonId: config.refreshButtonId,
        indexUrl: `/${section}-index.json?profile=${encodedProfile}`,
        docApiUrl: `/api/${section}/doc?profile=${encodedProfile}`,
        searchApiUrl: `/api/${section}/search?profile=${encodedProfile}`,
        pathPrefix: `ENCRYPTED/Sessions/${profile}/${section}`,
        directOpenPageName: `${section}.html`
    });
}
