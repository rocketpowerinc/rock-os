(function () {

    const allowedThemes = [
        'steel',
        'papers',
        'rock',
        'scissors'
    ];

    const legacyThemes = {
        terminal: 'rock',
        'warm-paper': 'papers',
        'blue-glass': 'scissors',
        cyberpunk: 'rock',
        rugged: 'papers',
        'blue-grass': 'scissors',
        bluegrass: 'scissors'
    };

    const defaultTheme =
        'steel';

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
