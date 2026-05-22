import { escapeHtml } from './utils.js';

function getToc() {

    return document.getElementById('wikiToc');
}

export function clearToc() {

    const toc =
        getToc();

    if (toc) {
        toc.innerHTML = '';
    }
}

function slugifyHeading(value) {

    const slug =
        value
            .toLowerCase()
            .trim()
            .replace(/&/g, ' and ')
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .replace(/^-|-$/g, '');

    return slug || 'section';
}

function highlightHeading(heading) {

    heading.classList.remove('heading-jump-highlight');

    window.setTimeout(() => {

        heading.classList.add('heading-jump-highlight');
    }, 40);

    window.setTimeout(() => {

        heading.classList.remove('heading-jump-highlight');
    }, 1800);
}

export function buildTableOfContents(container) {

    const toc =
        getToc();

    if (!toc) {
        return;
    }

    const headings =
        Array.from(
            container.querySelectorAll('h2, h3, h4')
        )
            .filter(heading =>
                heading.textContent.trim()
            );

    if (headings.length < 2) {
        toc.innerHTML = '';
        return;
    }

    const usedIds =
        new Map();

    headings.forEach(heading => {

        const baseId =
            heading.id ||
            slugifyHeading(heading.textContent);

        const count =
            usedIds.get(baseId) || 0;

        usedIds.set(baseId, count + 1);

        heading.id =
            count === 0
            ? baseId
            : `${baseId}-${count + 1}`;
    });

    const links =
        headings
            .map(heading => {

                const level =
                    Number(
                        heading.tagName.replace('H', '')
                    );

                const label =
                    escapeHtml(
                        heading.textContent.trim()
                    );

                return `
                    <a class="wiki-toc-link level-${level}" href="#${heading.id}">
                        ${label}
                    </a>
                `;
            })
            .join('');

    toc.innerHTML = `
        <p class="wiki-toc-title">On This Page</p>
        <nav class="wiki-toc-list">
            ${links}
        </nav>
    `;

    toc.querySelectorAll('a[href^="#"]')
        .forEach(link => {

            link.onclick = event => {

                const target =
                    document.getElementById(
                        link.getAttribute('href').slice(1)
                    );

                if (!target) {
                    return;
                }

                event.preventDefault();

                container.scrollTo({
                    top:
                        target.offsetTop -
                        container.offsetTop -
                        24,
                    behavior: 'smooth'
                });

                highlightHeading(target);

                const url =
                    new URL(window.location.href);

                url.hash =
                    target.id;

                window.history.replaceState(
                    {},
                    '',
                    url
                );
            };
        });
}

export function scrollToCurrentHash() {

    if (!window.location.hash) {
        return;
    }

    const target =
        document.getElementById(
            decodeURIComponent(
                window.location.hash.slice(1)
            )
        );

    if (!target) {
        return;
    }

    requestAnimationFrame(() => {
        const content =
            document.getElementById('content');

        if (!content) {
            return;
        }

        content.scrollTo({
            top:
                target.offsetTop -
                content.offsetTop -
                24,
            behavior: 'smooth'
        });
    });
}

