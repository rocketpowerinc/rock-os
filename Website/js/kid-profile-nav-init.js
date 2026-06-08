(function () {
    const sections = [
        { key: 'dashboards', label: 'Dashboards' },
        { key: 'bookmarks', label: 'Bookmarks' },
        { key: 'cheatsheets', label: 'Cheatsheets' },
        { key: 'dotfiles', label: 'Dotfiles' },
        { key: 'bootstraps', label: 'Bootstraps' },
        { key: 'scripts', label: 'Scripts' },
        { key: 'wiki', label: 'Wiki' }
    ];

    function encodeProfile(profile) {
        return String(profile || '')
            .split('/')
            .filter(Boolean)
            .map(part => encodeURIComponent(part))
            .join('/');
    }

    function currentProfile() {
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
        const sessionsIndex =
            parts.findIndex((part, index) =>
                part === 'Sessions' &&
                parts[index - 1] === 'ENCRYPTED'
            );
        if (sessionsIndex < 0) {
            return '';
        }

        const stopSections =
            new Set(sections.map(section => section.key));
        const profileParts = [];
        for (const part of parts.slice(sessionsIndex + 1)) {
            if (stopSections.has(part) || part === 'index.html') {
                break;
            }
            profileParts.push(part);
        }
        return profileParts.join('/');
    }

    function kidTheme(profile) {
        const normalized =
            String(profile || '').toLowerCase();
        if (normalized === 'family/profiles/boys') {
            return 'boys';
        }
        if (normalized === 'family/profiles/girls') {
            return 'girls';
        }
        return '';
    }

    function signatureFor(links) {
        return links
            .map(link => `${link.key}:${link.href}:${link.active ? '1' : '0'}`)
            .join('|');
    }

    const profile =
        currentProfile();
    const theme =
        kidTheme(profile);

    if (!profile) {
        return;
    }

    if (theme) {
        document.documentElement.classList.add('kid-profile-page', `kid-profile-${theme}`);
        document.body?.classList.add('kid-profile-page', `kid-profile-${theme}`);
    }

    const encodedProfile =
        encodeProfile(profile);
    const profilePath =
        `/ENCRYPTED/Sessions/${encodedProfile}/`;
    const currentPath =
        window.location.pathname.toLowerCase();
    const profileDashboardPath =
        `/encrypted/sessions/${encodedProfile.toLowerCase()}/dashboards/`;
    const links = [
        {
            key: 'overview',
            label: 'Hub',
            href: profilePath,
            active: currentPath.includes('/encrypted/sessions/') && !currentPath.includes('/dashboards/')
        },
        ...sections.map(section => ({
            ...section,
            href: section.key === 'dashboards'
                ? `/dashboards.html?profile=${encodeURIComponent(profile)}`
                : `/${section.key}.html?profile=${encodeURIComponent(profile)}`,
            active: section.key === 'dashboards'
                ? currentPath.endsWith('/dashboards.html') || currentPath.includes(profileDashboardPath)
                : currentPath.endsWith(`/${section.key}.html`)
        }))
    ];
    const signature =
        signatureFor(links);

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
        document.querySelector('.navbar')?.insertAdjacentElement('afterend', nav);
    }
    if (!nav || nav.dataset.navSignature === signature) {
        return;
    }

    nav.dataset.navSignature =
        signature;
    nav.replaceChildren(
        ...links.map(link => {
            const anchor =
                document.createElement('a');
            anchor.href =
                link.href;
            anchor.textContent =
                link.label;
            anchor.classList.toggle('is-active', link.active);
            if (link.active) {
                anchor.setAttribute('aria-current', 'page');
            }
            return anchor;
        })
    );
}());
