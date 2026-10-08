/* ============================================================
   Archivist — small vanilla JS enhancements
   (no external dependencies)
   ============================================================ */
(function () {
    "use strict";

    /* ---------- Theme toggle ---------- */
    var THEME_KEY = "archivist-theme";

    function applyTheme(theme) {
        var root = document.documentElement;
        if (theme === "dark") {
            root.setAttribute("data-theme", "dark");
        } else {
            root.setAttribute("data-theme", "light");
        }
        localStorage.setItem(THEME_KEY, theme);
    }

    function initTheme() {
        var saved = localStorage.getItem(THEME_KEY);
        if (!saved) {
            saved = "light";
        }
        applyTheme(saved);
    }

    function bindThemeToggle() {
        var btn = document.querySelector(".theme-toggle-btn");
        if (!btn) return;
        btn.addEventListener("click", function () {
            var next = document.documentElement.getAttribute("data-theme") === "dark" ? "light" : "dark";
            applyTheme(next);
        });
    }

    /* ---------- Mobile sidebar ---------- */
    function initSidebar() {
        var toggle = document.querySelector(".mobile-menu-btn");
        var backdrop = document.querySelector(".sidebar-backdrop");
        if (!toggle || !backdrop) return;

        function close() {
            document.body.classList.remove("sidebar-open");
        }

        toggle.addEventListener("click", function () {
            document.body.classList.toggle("sidebar-open");
        });

        backdrop.addEventListener("click", close);

        var sidebar = document.querySelector(".app-sidebar");
        if (sidebar) {
            sidebar.addEventListener("click", function (e) {
                if (e.target.tagName === "A") close();
            });
        }
    }

    /* ---------- Table search ---------- */
    function initTableSearch() {
        var boxes = document.querySelectorAll("[data-table-search]");
        Array.prototype.forEach.call(boxes, function (box) {
            var target = document.querySelector(box.getAttribute("data-table-search"));
            if (!target) return;
            var rows = Array.prototype.slice.call(target.querySelectorAll("tbody tr"));
            box.addEventListener("input", function () {
                var q = box.value.toLowerCase().trim();
                rows.forEach(function (row) {
                    row.style.display = row.textContent.toLowerCase().indexOf(q) > -1 ? "" : "none";
                });
            });
        });
    }

    /* ---------- Select all / none for checkboxes ---------- */
    function initSelectAll() {
        var toggles = document.querySelectorAll("[data-check-all]");
        Array.prototype.forEach.call(toggles, function (toggle) {
            var selector = toggle.getAttribute("data-check-all");
            if (!selector) return;
            function resolveTargets() {
                var found = document.querySelectorAll(selector);
                // Allow pointing at a container (e.g. "#group") instead of the inputs.
                if (found.length === 1 && found[0].querySelectorAll) {
                    var inner = found[0].querySelectorAll('input[type="checkbox"]');
                    if (inner.length > 0) return inner;
                }
                return found;
            }
            toggle.addEventListener("change", function () {
                var group = resolveTargets();
                Array.prototype.forEach.call(group, function (cb) {
                    if (cb !== toggle) cb.checked = toggle.checked;
                    cb.dispatchEvent(new Event("change", { bubbles: true }));
                });
            });
        });
    }

    /* ---------- Confirm modal ---------- */
    function initConfirmModal() {
        var template = [
            '<div class="modal confirm-modal">',
            '  <div class="modal-background"></div>',
            '  <div class="modal-card">',
            '    <header class="modal-card-head">',
            '      <p class="modal-card-title">Confirmar</p>',
            '    </header>',
            '    <section class="modal-card-body"></section>',
            '    <footer class="modal-card-foot">',
            '      <button type="button" class="button is-danger" data-confirm-ok>Confirmar</button>',
            '      <button type="button" class="button" data-confirm-cancel>Cancelar</button>',
            '    </footer>',
            '  </div>',
            '</div>'
        ].join("");

        var modal = null;
        var currentForm = null;

        function show(message) {
            if (!modal) {
                var el = document.createElement("div");
                el.innerHTML = template;
                modal = el.firstChild;
                document.body.appendChild(modal);
                modal.querySelector("[data-confirm-cancel]").addEventListener("click", close);
                modal.querySelector(".modal-background").addEventListener("click", close);
                modal.querySelector("[data-confirm-ok]").addEventListener("click", function () {
                    if (currentForm) {
                        currentForm.submit();
                    }
                    close();
                });
            }
            modal.querySelector(".modal-card-body").textContent = message;
            modal.classList.add("is-active");
        }

        function close() {
            if (modal) modal.classList.remove("is-active");
            currentForm = null;
        }

        document.addEventListener("submit", function (e) {
            var form = e.target;
            if (form.tagName !== "FORM") return;
            var msg = form.getAttribute("data-confirm");
            if (!msg) return;
            e.preventDefault();
            currentForm = form;
            show(msg);
        }, true);
    }

    /* ---------- Sortable tables ---------- */
    function initTableSort() {
        var tables = document.querySelectorAll("table.sortable");
        Array.prototype.forEach.call(tables, function (table) {
            var headers = table.querySelectorAll("thead th");
            var tbody = table.tBodies[0];
            if (!tbody || !headers.length) return;

            // Auto-mark Acciones column as unsortable if not already marked
            Array.prototype.forEach.call(headers, function (th) {
                var text = th.textContent.trim().toLowerCase();
                if (text === "acciones" || text === "acción") {
                    th.setAttribute("data-unsortable", "");
                }
            });

            function getCellValue(row, idx) {
                var cell = row.cells[idx];
                if (!cell) return "";
                // Prefer data-sort-value if present
                if (cell.getAttribute("data-sort-value") !== null) {
                    return cell.getAttribute("data-sort-value");
                }
                return cell.textContent.trim();
            }

            function detectType(values) {
                var numeric = 0;
                var dateLike = 0;
                var checked = 0;
                for (var i = 0; i < Math.min(values.length, 10); i++) {
                    var v = values[i];
                    if (v === "") continue;
                    checked++;
                    if (/^-?\d+([.,]\d+)?$/.test(v.replace(",", "."))) numeric++;
                    if (/^\d{4}-\d{2}-\d{2}/.test(v)) dateLike++;
                }
                if (checked === 0) return "string";
                if (dateLike === checked) return "date";
                if (numeric === checked) return "number";
                // Mixed numeric/string -> check if majority numeric
                if (numeric > 0 && numeric >= checked / 2) return "number";
                return "string";
            }

            Array.prototype.forEach.call(headers, function (th, idx) {
                if (th.hasAttribute("data-unsortable")) return;
                th.addEventListener("click", function () {
                    var isAsc = th.classList.contains("is-sorted-asc");
                    var dir = isAsc ? "desc" : "asc";

                    // Clear other headers
                    Array.prototype.forEach.call(headers, function (h) {
                        h.classList.remove("is-sorted-asc", "is-sorted-desc");
                        h.removeAttribute("aria-sort");
                    });
                    th.classList.add(dir === "asc" ? "is-sorted-asc" : "is-sorted-desc");
                    th.setAttribute("aria-sort", dir === "asc" ? "ascending" : "descending");

                    var rows = Array.prototype.slice.call(tbody.rows);
                    // Detect type from current column values
                    var vals = rows.map(function (r) { return getCellValue(r, idx); });
                    var type = detectType(vals);

                    rows.sort(function (a, b) {
                        var va = getCellValue(a, idx);
                        var vb = getCellValue(b, idx);
                        var cmp = 0;
                        if (type === "number") {
                            var na = parseFloat(va.replace(",", "."));
                            var nb = parseFloat(vb.replace(",", "."));
                            if (isNaN(na)) na = 0;
                            if (isNaN(nb)) nb = 0;
                            cmp = na - nb;
                        } else if (type === "date") {
                            var da = Date.parse(va);
                            var db = Date.parse(vb);
                            if (isNaN(da)) da = 0;
                            if (isNaN(db)) db = 0;
                            cmp = da - db;
                        } else {
                            cmp = va.localeCompare(vb, "es", { sensitivity: "base", numeric: true });
                        }
                        return dir === "asc" ? cmp : -cmp;
                    });

                    // Re-append in sorted order
                    rows.forEach(function (row) { tbody.appendChild(row); });
                });
            });
        });
    }

    /* ---------- Flash auto-dismiss ---------- */
    function initFlash() {
        var notif = document.querySelector(".flash .notification");
        if (!notif) return;
        var del = notif.querySelector(".delete");
        if (del) {
            del.addEventListener("click", function () {
                notif.parentNode.removeChild(notif.parentNode.querySelector(".flash"));
            });
        }
        setTimeout(function () {
            var flash = notif.parentNode.querySelector(".flash");
            if (flash) flash.style.display = "none";
        }, 6000);
    }

    /* ---------- Import page: dropzone + preview + spinner ---------- */
    function initImportForm() {
        var form = document.getElementById("import-form");
        if (!form) return;
        var dropzone = document.getElementById("import-dropzone");
        var input = document.getElementById("import-file");
        var preview = document.getElementById("import-file-preview");
        var fileName = document.getElementById("import-file-name");
        var fileSize = document.getElementById("import-file-size");
        var clearBtn = document.getElementById("import-file-clear");
        var errorMsg = document.getElementById("import-file-error");
        var submitBtn = document.getElementById("import-submit");
        var resetBtn = document.getElementById("import-reset");
        var countTag = document.getElementById("import-sheets-count");
        var restoreBox = document.getElementById("import-restore-confirm");
        var restoreCheck = document.getElementById("import-restore-check");
        var sheetsBox = document.getElementById("import-sheets-box");
        if (!dropzone || !input) return;

        function formatSize(bytes) {
            if (!bytes && bytes !== 0) return "";
            if (bytes < 1024) return bytes + " B";
            if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
            return (bytes / (1024 * 1024)).toFixed(2) + " MB";
        }

        function isJsonFile(file) {
            if (!file) return false;
            var name = (file.name || "").toLowerCase();
            if (name.endsWith(".json")) return true;
            var type = file.type || "";
            return type === "application/json" || type.endsWith("+json");
        }

        function isValidImportFile(file) {
            if (!file) return false;
            var name = (file.name || "").toLowerCase();
            if (name.endsWith(".xlsx") || name.endsWith(".json")) return true;
            var type = file.type || "";
            return type === "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
                type === "application/json" ||
                type.endsWith("+json");
        }

        function applyFileMode(file) {
            var json = isJsonFile(file);
            if (restoreBox) restoreBox.hidden = !json;
            if (sheetsBox) sheetsBox.hidden = json;
            if (restoreCheck) {
                restoreCheck.required = json;
                if (!json) restoreCheck.checked = false;
            }
        }

        function updateSheetsCount() {
            if (!countTag) return;
            var boxes = document.querySelectorAll('#import-sheets-checkboxes input[type="checkbox"]');
            var total = boxes.length;
            var checked = 0;
            Array.prototype.forEach.call(boxes, function (cb) {
                if (cb.checked) checked++;
            });
            countTag.textContent = checked + " de " + total;
        }

        function showPreview(file) {
            if (!file || !preview) return;
            if (fileName) fileName.textContent = file.name;
            if (fileName) fileName.title = file.name;
            if (fileSize) fileSize.textContent = formatSize(file.size);
            preview.hidden = false;
            dropzone.classList.add("has-file");
            if (errorMsg) errorMsg.hidden = true;
        }

        function clearFile() {
            input.value = "";
            if (preview) preview.hidden = true;
            dropzone.classList.remove("has-file");
            if (errorMsg) errorMsg.hidden = true;
            applyFileMode(null);
        }

        function setFile(file) {
            if (!file) return;
            if (!isValidImportFile(file)) {
                if (errorMsg) errorMsg.hidden = false;
                return;
            }
            applyFileMode(file);
            showPreview(file);
        }

        input.addEventListener("change", function () {
            if (input.files && input.files.length > 0) {
                setFile(input.files[0]);
                // If invalid, reset so `required` keeps working.
                if (errorMsg && !errorMsg.hidden) input.value = "";
            }
        });

        ["dragenter", "dragover"].forEach(function (evt) {
            dropzone.addEventListener(evt, function (e) {
                e.preventDefault();
                dropzone.classList.add("is-dragover");
            });
        });
        ["dragleave", "drop"].forEach(function (evt) {
            dropzone.addEventListener(evt, function (e) {
                e.preventDefault();
                dropzone.classList.remove("is-dragover");
            });
        });
        dropzone.addEventListener("drop", function (e) {
            var files = e.dataTransfer && e.dataTransfer.files;
            if (files && files.length > 0) {
                try {
                    input.files = files;
                } catch (err) {
                    // Some browsers disallow direct assignment; fall back to preview only.
                }
                setFile(files[0]);
                if (errorMsg && !errorMsg.hidden) input.value = "";
            }
        });
        dropzone.addEventListener("keydown", function (e) {
            if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                input.click();
            }
        });

        if (clearBtn) {
            clearBtn.addEventListener("click", function (e) {
                e.preventDefault();
                e.stopPropagation();
                clearFile();
            });
        }

        var sheets = document.querySelectorAll('#import-sheets-checkboxes input[type="checkbox"]');
        Array.prototype.forEach.call(sheets, function (cb) {
            cb.addEventListener("change", updateSheetsCount);
        });
        updateSheetsCount();

        form.addEventListener("reset", function () {
            setTimeout(function () {
                clearFile();
                updateSheetsCount();
                if (submitBtn) {
                    submitBtn.classList.remove("is-loading");
                    submitBtn.disabled = false;
                }
            }, 0);
        });

        form.addEventListener("submit", function () {
            if (submitBtn) {
                submitBtn.classList.add("is-loading");
                submitBtn.disabled = true;
            }
        });
    }

    /* ---------- Boot ---------- */
    document.addEventListener("DOMContentLoaded", function () {
        initTheme();
        bindThemeToggle();
        initSidebar();
        initTableSearch();
        initTableSort();
        initSelectAll();
        initConfirmModal();
        initFlash();
        initImportForm();
    });
})();
