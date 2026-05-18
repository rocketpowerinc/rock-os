const quoteElement =
    document.getElementById('dailyQuote');

function parseQuoteBullets(markdown) {

    return markdown
        .split(/\r?\n/)
        .map(line => line.trim())
        .filter(line => /^[-*]\s+/.test(line))
        .map(line => line.replace(/^[-*]\s+/, '').trim())
        .filter(Boolean);
}

function showRandomQuote(quotes) {

    if (!quoteElement || quotes.length === 0) {
        quoteElement?.closest('.quote-panel')?.classList.add('is-empty');
        return;
    }

    const quote =
        quotes[Math.floor(Math.random() * quotes.length)];

    quoteElement.textContent =
        quote.replace(/^["'\u2018-\u201D]+|["'\u2018-\u201D]+$/g, '');
}

async function loadQuote() {

    if (!quoteElement) {
        return;
    }

    try {
        const response =
            await fetch('quotes.md?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error('Could not load quotes.md');
        }

        const markdown =
            await response.text();

        showRandomQuote(
            parseQuoteBullets(markdown)
        );
    }
    catch (err) {
        console.warn(err);
        quoteElement.closest('.quote-panel')?.classList.add('is-empty');
    }
}

loadQuote();
