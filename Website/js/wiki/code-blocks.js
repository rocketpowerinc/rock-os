import { copyText } from './utils.js';

function codeLanguage(code) {

    const languageClass =
        Array.from(code.classList)
            .find(className =>
                className.startsWith('language-')
            );

    if (!languageClass) {
        return 'text';
    }

    const language =
        languageClass.replace('language-', '').toLowerCase();

    const aliases = {
        ps1: 'powershell',
        pwsh: 'powershell',
        shell: 'bash',
        sh: 'bash',
        zsh: 'bash'
    };

    return aliases[language] || language;
}

function languageLabel(language) {

    const labels = {
        bash: 'Bash',
        powershell: 'PowerShell',
        text: 'Text'
    };

    return labels[language] || language;
}

function highlightCode(code, rawText, language) {

    if (!window.hljs) {
        return;
    }

    try {

        if (language !== 'text' && window.hljs.getLanguage(language)) {

            code.innerHTML =
                window.hljs.highlight(rawText, {
                    language
                }).value;
        } else {

            code.innerHTML =
                window.hljs.highlightAuto(rawText).value;
        }

        code.classList.add('hljs');
    }
    catch (err) {

        console.warn('Code highlighting failed:', err);
        code.innerText = rawText;
    }
}

function createLineNumbers(rawText) {

    const lineCount =
        Math.max(
            1,
            rawText.replace(/\n$/, '').split('\n').length
        );

    const gutter =
        document.createElement('div');

    gutter.className = 'code-line-numbers';
    gutter.setAttribute('aria-hidden', 'true');

    for (let index = 1; index <= lineCount; index += 1) {

        const line =
            document.createElement('span');

        line.innerText = String(index);
        gutter.appendChild(line);
    }

    return gutter;
}

export function enhanceCodeBlocks(container) {

    container.querySelectorAll('pre > code')
        .forEach(code => {

            const pre = code.parentElement;

            if (!pre || pre.parentElement.classList.contains(
                'code-block'
            )) {
                return;
            }

            const wrapper =
                document.createElement('div');

            wrapper.className = 'code-block';

            const rawText =
                code.textContent;

            const language =
                codeLanguage(code);

            highlightCode(code, rawText, language);

            const header =
                document.createElement('div');

            header.className = 'code-block-header';

            const label =
                document.createElement('span');

            label.className = 'code-language-label';
            label.innerText = languageLabel(language);

            const button =
                document.createElement('button');

            button.className = 'copy-code-btn';
            button.type = 'button';
            button.innerText = 'Copy';

            button.onclick = async () => {

                try {

                    await copyText(rawText);

                    button.innerText = 'Copied';

                    setTimeout(() => {
                        button.innerText = 'Copy';
                    }, 1600);
                }
                catch (err) {

                    console.error('Copy failed:', err);
                    button.innerText = 'Error';

                    setTimeout(() => {
                        button.innerText = 'Copy';
                    }, 1600);
                }
            };

            const body =
                document.createElement('div');

            body.className = 'code-block-body';

            pre.parentNode.insertBefore(wrapper, pre);
            header.appendChild(label);
            header.appendChild(button);
            body.appendChild(createLineNumbers(rawText));
            body.appendChild(pre);
            wrapper.appendChild(header);
            wrapper.appendChild(body);
        });
}

export function enhanceInlineCode(container) {

    container.querySelectorAll(':not(pre) > code')
        .forEach(code => {

            if (code.dataset.copyReady === 'true') {
                return;
            }

            const rawText =
                code.textContent;

            if (!rawText.trim()) {
                return;
            }

            code.dataset.copyReady = 'true';
            code.setAttribute('role', 'button');
            code.setAttribute('tabindex', '0');
            code.setAttribute('title', 'Copy inline code');
            code.setAttribute('aria-label', `Copy ${rawText}`);
            code.classList.add('inline-copy-code');

            const copyInlineCode = async () => {

                try {

                    await copyText(rawText);

                    code.classList.add('is-copied');
                    code.setAttribute('title', 'Copied');

                    setTimeout(() => {
                        code.classList.remove('is-copied');
                        code.setAttribute('title', 'Copy inline code');
                    }, 1200);
                }
                catch (err) {

                    console.error('Inline copy failed:', err);

                    code.classList.add('is-copy-error');
                    code.setAttribute('title', 'Copy failed');

                    setTimeout(() => {
                        code.classList.remove('is-copy-error');
                        code.setAttribute('title', 'Copy inline code');
                    }, 1200);
                }
            };

            code.addEventListener('click', copyInlineCode);

            code.addEventListener('keydown', event => {

                if (event.key !== 'Enter' && event.key !== ' ') {
                    return;
                }

                event.preventDefault();
                copyInlineCode();
            });
        });
}

