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
        var toggle = document.querySelector("[data-check-all]");
        if (!toggle) return;
        var group = document.querySelectorAll(toggle.getAttribute("data-check-all"));
        toggle.addEventListener("change", function () {
            Array.prototype.forEach.call(group, function (cb) {
                cb.checked = toggle.checked;
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

    /* ---------- Boot ---------- */
    document.addEventListener("DOMContentLoaded", function () {
        initTheme();
        bindThemeToggle();
        initSidebar();
        initTableSearch();
        initSelectAll();
        initConfirmModal();
        initFlash();
    });
})();
