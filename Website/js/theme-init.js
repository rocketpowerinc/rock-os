(function () {

    const allowedThemes = [
        'steel',
        'rugged',
        'cyberpunk',
        'blue-grass'
    ];

    const defaultTheme = 'steel';

    try {
        const savedTheme = localStorage.getItem('rock-os-theme');

        document.documentElement.dataset.theme =
            allowedThemes.includes(savedTheme)
            ? savedTheme
            : defaultTheme;

        const pagePath =
            window.location.pathname.toLowerCase();
        const params =
            new URLSearchParams(window.location.search);
        const profile =
            String(params.get('profile') || '').toLowerCase();
        const profilePath =
            `${pagePath}/${profile}`.replaceAll('%2f', '/');

        if (profilePath.includes('public/family/profiles/boys')) {
            document.documentElement.classList.add('kid-profile-page', 'kid-profile-boys');
        }
        else if (profilePath.includes('public/family/profiles/girls')) {
            document.documentElement.classList.add('kid-profile-page', 'kid-profile-girls');
        }
    }
    catch (err) {
        document.documentElement.dataset.theme = defaultTheme;
    }
}());
