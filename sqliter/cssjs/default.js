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

    // --- Row CRUD Logic ---
    const isEditable = document.querySelector('meta[name="sqliter-editable"][content="true"]');
    if (isEditable) {
        console.log("SQLITER: Editable mode enabled");
        enableRowCRUD();
    }

    const EDIT_TRIGGER_MODE = 'dblclick'; // Options: 'dblclick', 'manual'

    function enableRowCRUD() {
        const table = document.querySelector('table');
        if (!table) return;

        // Edit Mode Toggle
        const pencilHeader = document.querySelector('.row-id-header');
        if (pencilHeader) {
            pencilHeader.style.cursor = 'pointer';
            pencilHeader.addEventListener('click', toggleEditMode);
        }

        // Add "Add Row" button to Edit Bar
        const editBarCell = document.querySelector('#edit-bar-row th');
        const addBtn = document.createElement('button');
        addBtn.id = 'addRowBtn';
        addBtn.innerText = 'Add Row';
        addBtn.style.display = 'none'; // Hidden by default, shown in edit mode

        if (editBarCell) {
            editBarCell.appendChild(addBtn);
        } else {
            // Fallback if template update didn't work for some reason (shouldn't happen)
            document.body.insertBefore(addBtn, table);
        }

        addBtn.addEventListener('click', handleCreate);

        // Make cells editable based on trigger
        table.querySelectorAll('tbody td').forEach(td => {
            td.dataset.original = td.innerText; // Store original value

            // Ignore Row ID column
            if (td.classList.contains('row-id')) return;

            // Handle save on blur
            td.addEventListener('blur', function () {
                const newValue = this.innerText;
                const originalValue = this.dataset.original;

                if (newValue !== originalValue) {
                    handleUpdate(this, newValue);
                }
            });

            td.addEventListener('keydown', function (e) {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    this.blur();
                }
            });
        });

        // Add context menu for deletion
        table.querySelectorAll('tbody tr').forEach(tr => {
            tr.addEventListener('contextmenu', function (e) {
                e.preventDefault();
                // Only allow delete in edit mode? Or always?
                // User requirement implies "Edit Mode" toggles editability.
                // Let's restrict delete to Edit Mode for safety/consistency.
                if (document.body.classList.contains('edit-mode')) {
                    if (confirm('Delete this row?')) {
                        handleDelete(this);
                    }
                }
            });
        });
    }

    function toggleEditMode() {
        document.body.classList.toggle('edit-mode');
        const isEditing = document.body.classList.contains('edit-mode');
        const table = document.querySelector('table');

        // Toggle contentEditable on valid cells
        table.querySelectorAll('tbody td').forEach(td => {
            if (td.classList.contains('row-id')) return;
            td.contentEditable = isEditing;
        });

        // Toggle Add Row button visibility
        const addBtn = document.getElementById('addRowBtn');
        if (addBtn) {
            addBtn.style.display = isEditing ? 'inline-block' : 'none';
        }
    }

    function makeEditable(td) {
        td.contentEditable = true;
        td.focus();
        // Select all text
        const range = document.createRange();
        range.selectNodeContents(td);
        const sel = window.getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
    }

    function getRowData(tr) {
        const headers = Array.from(tr.closest('table').querySelectorAll('thead th')).map(th => th.innerText);
        const cells = Array.from(tr.children);
        const data = {};
        headers.forEach((h, i) => {
            if (cells[i]) data[h] = cells[i].dataset.original || cells[i].innerText; // Use original values for identification
        });
        return data;
    }

    function getKeyData(tr) {
        // Identify row for WHERE clause.
        // Since we don't know the PK, we use ALL columns with their ORIGINAL values.
        return getRowData(tr);
    }

    function handleUpdate(td, newValue) {
        const tr = td.parentElement;
        const headers = Array.from(tr.closest('table').querySelectorAll('thead th')).map(th => th.innerText);
        const cellIndex = Array.from(tr.children).indexOf(td);
        const columnName = headers[cellIndex];

        const where = getKeyData(tr);
        const data = { [columnName]: newValue };

        // Remove the column being updated from the WHERE clause if it was part of the key
        // Actually, for the WHERE clause we want the OLD value of this column.
        where[columnName] = td.dataset.original;

        sendCRUD('update', { data, where })
            .then(() => {
                td.dataset.original = newValue;
                td.classList.add('updated-success');
                setTimeout(() => td.classList.remove('updated-success'), 1000);
            })
            .catch(err => {
                console.error("Update failed", err);
                td.innerText = td.dataset.original; // Revert
                td.classList.add('updated-error');
                setTimeout(() => td.classList.remove('updated-error'), 1000);
            });
    }

    function handleDelete(tr) {
        const where = getKeyData(tr);
        sendCRUD('delete', { where })
            .then(() => {
                tr.remove();
            })
            .catch(err => console.error("Delete failed", err));
    }

    function handleCreate() {
        // Create a new empty row
        const table = document.querySelector('table');
        const headers = Array.from(table.querySelectorAll('thead th')).map(th => th.innerText);
        const data = {};

        // Use empty string for all columns initially (or prompt user?)
        // User said "No extra forms... table becomes the form". 
        // So we insert a blank row and let them edit it? 
        // But we need to INSERT it into DB. 
        // SQLite will fail on NOT NULL constraints if we insert empty.
        // A better UX might be: Insert a row in UI, user fills it, then "Save" or auto-save on fill?
        // Let's try: Insert row with default values (empty strings), try to save. If fail, alert.
        headers.forEach(h => data[h] = "");

        sendCRUD('create', { data })
            .then(() => {
                location.reload(); // Reload to get the new row properly rendered/ID'd from server (if auto-increment)
            })
            .catch(err => {
                console.error("Create failed", err);
                // No alert, just log
            });

        // Alternative: Just reload which might not show anything if create failed.
    }

    function sendCRUD(action, payload) {
        return fetch(window.location.href, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ action, ...payload })
        }).then(response => {
            if (!response.ok) {
                return response.text().then(text => { throw new Error(text) });
            }
            return response.json();
        });
    }
});
