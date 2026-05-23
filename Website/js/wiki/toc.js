import { escapeHtml } from './utils.js';

let tocObserver = null;
let tocScrollCleanup = null;

function getToc() {

    return document.getElementById('wikiToc');
}

export function clearToc() {

    stopTocScrollSpy();

    const toc =
        getToc();

    if (toc) {
        toc.innerHTML = '';
    }
}

function stopTocScrollSpy() {

    if (tocObserver) {
        tocObserver.disconnect();
        tocObserver = null;
    }

    if (tocScrollCleanup) {
        tocScrollCleanup();
        tocScrollCleanup = null;
    }
}

function setActiveTocLink(id) {

    const toc =
        getToc();

    if (!toc || !id) {
        return;
    }

    toc.querySelectorAll('.wiki-toc-link')
        .forEach(link => {

            link.classList.toggle(
                'active',
                link.getAttribute('href') === `#${id}`
            );
        });
}

function scrolledToBottom(container) {

    return Math.ceil(
        container.scrollTop + container.clientHeight
    ) >= container.scrollHeight - 1;
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

function observeTocScroll(container, headings) {

    stopTocScrollSpy();

    if (!headings.length) {
        return;
    }

    setActiveTocLink(headings[0].id);

    const setLastHeadingIfAtBottom = () => {

        if (!scrolledToBottom(container)) {
            return false;
        }

        setActiveTocLink(
            headings[headings.length - 1].id
        );

        return true;
    };

    if ('IntersectionObserver' in window) {

        const visibleHeadings =
            new Map();

        tocObserver =
            new IntersectionObserver(entries => {

                if (setLastHeadingIfAtBottom()) {
                    return;
                }

                entries.forEach(entry => {

                    if (entry.isIntersecting) {
                        visibleHeadings.set(
                            entry.target.id,
                            entry.boundingClientRect.top
                        );
                    } else {
                        visibleHeadings.delete(entry.target.id);
                    }
                });

                if (!visibleHeadings.size) {
                    return;
                }

                const [activeId] =
                    Array.from(visibleHeadings.entries())
                        .sort((a, b) => a[1] - b[1])[0];

                setActiveTocLink(activeId);
            }, {
                root: container,
                rootMargin: '-12% 0px -72% 0px',
                threshold: [0, 1]
            });

        headings.forEach(heading =>
            tocObserver.observe(heading)
        );

        let animationFrame = null;
        const onScroll = () => {

            if (animationFrame) {
                window.cancelAnimationFrame(animationFrame);
            }

            animationFrame =
                window.requestAnimationFrame(() => {
                    animationFrame = null;
                    setLastHeadingIfAtBottom();
                });
        };

        container.addEventListener('scroll', onScroll, {
            passive: true
        });
        tocScrollCleanup = () => {
            container.removeEventListener('scroll', onScroll);
            if (animationFrame) {
                window.cancelAnimationFrame(animationFrame);
            }
        };

        return;
    }

    let scrollTimer = null;

    const updateActiveHeading = () => {

        if (setLastHeadingIfAtBottom()) {
            return;
        }

        const containerTop =
            container.getBoundingClientRect().top;

        const activeHeading =
            headings
                .slice()
                .reverse()
                .find(heading =>
                    heading.getBoundingClientRect().top -
                    containerTop <= 96
                ) ||
            headings[0];

        setActiveTocLink(activeHeading.id);
    };

    const onScroll = () => {

        window.clearTimeout(scrollTimer);
        scrollTimer =
            window.setTimeout(updateActiveHeading, 80);
    };

    container.addEventListener('scroll', onScroll, {
        passive: true
    });
    tocScrollCleanup = () => {
        container.removeEventListener('scroll', onScroll);
        window.clearTimeout(scrollTimer);
    };
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
                setActiveTocLink(target.id);

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

    observeTocScroll(container, headings);
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

