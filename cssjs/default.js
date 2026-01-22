document.addEventListener('DOMContentLoaded', () => {
    // Update title if it's the default SQLITER
    if (document.title === 'SQLITER') {
        const urlPath = window.location.pathname.replace(/^\/+/, '');
        const shortPath = urlPath.length > 80 ? '...' + urlPath.slice(-77) : urlPath;
        document.title = shortPath || 'SQLITER';
    }

    const getCellValue = (tr, idx) => tr.children[idx].innerText || tr.children[idx].textContent;

    const comparer = (idx, asc) => (a, b) => ((v1, v2) =>
        v1 !== '' && v2 !== '' && !isNaN(v1) && !isNaN(v2) ? v1 - v2 : v1.toString().localeCompare(v2)
    )(getCellValue(asc ? a : b, idx), getCellValue(asc ? b : a, idx));

    // do the work...
    document.querySelectorAll('th').forEach(th => th.addEventListener('click', (() => {
        const table = th.closest('table');
        const tbody = table.querySelector('tbody');

        // Remove active sort classes from other headers
        table.querySelectorAll('th').forEach(header => {
            if (header !== th) header.classList.remove('sort-asc', 'sort-desc');
        });

        // Toggle sort direction
        const asc = !th.classList.contains('sort-asc');
        th.classList.toggle('sort-asc', asc);
        th.classList.toggle('sort-desc', !asc);

        Array.from(tbody.querySelectorAll('tr'))
            .sort(comparer(Array.from(th.parentNode.children).indexOf(th), asc))
            .forEach(tr => tbody.appendChild(tr));
    })));
});
