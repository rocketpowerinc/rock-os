function calloutMeta(type) {

    const normalizedType =
        type.toLowerCase();

    const labels = {
        note: 'Note',
        info: 'Info',
        tip: 'Tip',
        success: 'Success',
        warning: 'Warning',
        danger: 'Danger',
        error: 'Error',
        question: 'Question'
    };

    return {
        type: normalizedType,
        label: labels[normalizedType] || type
    };
}

export function enhanceCallouts(container) {

    container.querySelectorAll('blockquote')
        .forEach(blockquote => {

            const firstParagraph =
                blockquote.querySelector('p');

            if (!firstParagraph) {
                return;
            }

            const match =
                firstParagraph.textContent
                    .trimStart()
                    .match(/^\[!(\w+)\]/);

            if (!match) {
                return;
            }

            const meta =
                calloutMeta(match[1]);

            firstParagraph.innerHTML =
                firstParagraph.innerHTML.replace(
                    /^\s*\[!\w+\]\s*/i,
                    ''
                );

            const title =
                document.createElement('div');

            title.className = 'callout-title';
            title.innerText = meta.label;

            blockquote.classList.add(
                'callout',
                `callout-${meta.type}`
            );

            blockquote.insertBefore(
                title,
                blockquote.firstChild
            );

            if (!firstParagraph.textContent.trim()) {
                firstParagraph.remove();
            }
        });
}

