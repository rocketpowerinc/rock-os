(function() {
    const sidebar = document.getElementById('sidebar');
    const resizer = document.getElementById('sidebarResizer');
    const layout = document.querySelector('.layout') || document.querySelector('.scripts-layout');

    if (!sidebar || !resizer || !layout) {
        return;
    }

    const storageKey = 'rock-os-sidebar-width';

    // Load saved width from localStorage
    const savedWidth = localStorage.getItem(storageKey);
    if (savedWidth) {
        const widthNum = parseInt(savedWidth, 10);
        if (widthNum >= 180 && widthNum <= 600) {
            sidebar.style.width = widthNum + 'px';
            sidebar.style.minWidth = widthNum + 'px';
            sidebar.style.maxWidth = widthNum + 'px';
        }
    }

    let startX = 0;
    let startWidth = 0;

    resizer.addEventListener('mousedown', initDrag);

    function initDrag(e) {
        e.preventDefault();
        startX = e.clientX;
        startWidth = sidebar.getBoundingClientRect().width;
        
        document.body.classList.add('is-resize-dragging');
        resizer.classList.add('is-dragging');

        document.addEventListener('mousemove', doDrag);
        document.addEventListener('mouseup', stopDrag);
    }

    function doDrag(e) {
        const newWidth = startWidth + (e.clientX - startX);
        if (newWidth >= 180 && newWidth <= 600) {
            sidebar.style.width = newWidth + 'px';
            sidebar.style.minWidth = newWidth + 'px';
            sidebar.style.maxWidth = newWidth + 'px';
        }
    }

    function stopDrag() {
        document.body.classList.remove('is-resize-dragging');
        resizer.classList.remove('is-dragging');
        
        // Save new width to localStorage
        const currentWidth = sidebar.getBoundingClientRect().width;
        localStorage.setItem(storageKey, currentWidth);

        document.removeEventListener('mousemove', doDrag);
        document.removeEventListener('mouseup', stopDrag);
    }
})();
