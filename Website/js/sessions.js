const sessionSelect =
    document.getElementById('sessionSelect');

function renderSessionSelect(config) {
    if (!sessionSelect) {
        return;
    }

    const sessions =
        Array.isArray(config?.sessions) ? config.sessions : [];

    sessionSelect.replaceChildren();

    if (sessions.length === 0) {
        const option =
            document.createElement('option');
        option.value =
            '';
        option.textContent =
            'Sessions unavailable';
        sessionSelect.append(option);
        sessionSelect.disabled =
            true;
        return;
    }

    for (const session of sessions) {
        const name =
            String(session?.name || '').trim();
        if (!name) {
            continue;
        }

        const option =
            document.createElement('option');
        option.value =
            name;
        option.textContent =
            name;
        if (session.description) {
            option.title =
                session.description;
        }
        sessionSelect.append(option);
    }

    sessionSelect.value =
        config?.active || sessions[0]?.name || '';
    sessionSelect.disabled =
        false;
    sessionSelect.title =
        'Choose the active Rock-OS dashboard session.';
}

async function loadSessions() {
    if (!sessionSelect) {
        return;
    }

    try {
        const response =
            await fetch('/api/sessions?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load dashboard sessions');
        }

        renderSessionSelect(
            await response.json()
        );
    }
    catch (err) {
        console.warn(err);
        renderSessionSelect(null);
    }
}

async function updateActiveSession(active) {
    const response =
        await fetch('/api/sessions', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Rock-OS-Requested': 'true'
            },
            body: JSON.stringify({ active })
        });

    if (!response.ok) {
        throw new Error(
            (await response.text()).trim() ||
            `Session update failed with HTTP ${response.status}`
        );
    }

    return response.json();
}

function sessionChangeNeedsReload() {
    const path =
        window.location.pathname.toLowerCase();

    return path.endsWith('/dashboards.html') ||
        path.includes('/encrypted/dashboards/');
}

if (sessionSelect) {
    sessionSelect.disabled =
        true;
    sessionSelect.addEventListener('change', async () => {
        const nextSession =
            sessionSelect.value;
        if (!nextSession) {
            return;
        }

        sessionSelect.disabled =
            true;

        try {
            renderSessionSelect(
                await updateActiveSession(nextSession)
            );
            if (sessionChangeNeedsReload()) {
                window.location.reload();
            }
        }
        catch (err) {
            console.warn(err);
            window.alert(`Rock-OS could not change the dashboard session.\n\n${err.message}`);
            await loadSessions();
        }
    });
}

loadSessions();
