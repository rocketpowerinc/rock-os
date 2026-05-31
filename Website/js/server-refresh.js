export async function pullLatestRockOS() {
    const response =
        await fetch('/api/server/refresh', {
            method: 'POST',
            headers: {
                'X-Rock-OS-Requested': 'true'
            }
        });

    if (!response.ok) {
        throw new Error(
            (await response.text()).trim() ||
            `Live update failed with HTTP ${response.status}`
        );
    }

    return response.json();
}

export async function pullLatestRockOSAndReload() {
    const result =
        await pullLatestRockOS();

    if (result?.updated) {
        window.location.reload();
        return true;
    }

    return false;
}

export function warnLiveUpdateFailed(err) {
    console.warn('Rock-OS live update failed:', err);
    window.alert(
        `Rock-OS could not pull the latest GitHub changes. Refreshing local files instead.\n\n${err.message}`
    );
}
