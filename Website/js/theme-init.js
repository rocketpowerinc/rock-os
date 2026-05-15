(function () {

    const allowedThemes = [
        'rock',
        'papers',
        'scissors'
    ];

    const legacyThemes = {
        terminal: 'rock',
        'warm-paper': 'papers',
        'blue-glass': 'scissors'
    };

    const defaultTheme =
        'rock';

    try {

        const savedTheme =
            localStorage.getItem('rock-os-theme');

        document.documentElement.dataset.theme =
            allowedThemes.includes(savedTheme)
            ? savedTheme
            : legacyThemes[savedTheme] || defaultTheme;
    }
    catch (err) {

        document.documentElement.dataset.theme =
            defaultTheme;
    }
}());
