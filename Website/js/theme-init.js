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
    }
    catch (err) {
        document.documentElement.dataset.theme = defaultTheme;
    }
}());
