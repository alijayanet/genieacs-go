/**
 * GO-ACS Sidebar Component
 * Include this script in any page to automatically load the sidebar
 * Usage: <script src="/static/js/sidebar.js"></script>
 */

const sidebarConfig = {
    logo: {
        icon: 'fa-network-wired',
        text: 'GO-ACS',
        href: '/dashboard'
    },
    menuItems: [
        { id: 'dashboard', icon: 'fa-th-large', text: 'Dashboard', href: '/dashboard' },
        { id: 'devices', icon: 'fa-router', text: 'Devices', href: '/devices' },
        { id: 'map', icon: 'fa-map-marked-alt', text: 'Map', href: '/map' },
        { id: 'customers', icon: 'fa-users', text: 'Customers', href: '/customers' },
        { id: 'packages', icon: 'fa-cube', text: 'Packages', href: '/packages' },
        { id: 'billing', icon: 'fa-file-invoice-dollar', text: 'Billing', href: '/billing' },
        { id: 'provisions', icon: 'fa-magic', text: 'Provisions', href: '/provisions' },
        { id: 'tickets', icon: 'fa-headset', text: 'Support', href: '/tickets' },
        { id: 'logs', icon: 'fa-history', text: 'Logs', href: '/logs' },
        { id: 'settings', icon: 'fa-cog', text: 'Settings', href: '/settings' },
        { id: 'update', icon: 'fa-cloud-download-alt', text: 'Update', href: '/update' }
    ],
    footer: {
        version: 'v1.0.0',
        copyright: '© 2026 ALIJAYA-NET',
        contact: '081947215703'
    }
};

function getCurrentPage() {
    const path = window.location.pathname;
    if (path === '/dashboard' || path === '/') return 'dashboard';
    if (path === '/devices' || path.startsWith('/device/')) return 'devices';
    if (path === '/map') return 'map';
    if (path === '/customers') return 'customers';
    if (path === '/packages') return 'packages';
    if (path === '/billing') return 'billing';
    if (path === '/provisions') return 'provisions';
    if (path === '/tickets') return 'tickets';
    if (path === '/logs') return 'logs';
    if (path === '/settings') return 'settings';
    if (path === '/update') return 'update';
    return '';
}

function renderSidebar() {
    const currentPage = getCurrentPage();
    const sidebar = document.createElement('aside');
    sidebar.className = 'sidebar';
    sidebar.id = 'sidebar';

    // Logo
    let html = `
        <a href="${sidebarConfig.logo.href}" class="sidebar-logo">
            <i class="fas ${sidebarConfig.logo.icon}"></i> ${sidebarConfig.logo.text}
        </a>
        <nav>
    `;

    // Menu items
    sidebarConfig.menuItems.forEach(item => {
        const activeClass = item.id === currentPage ? 'active' : '';
        html += `
            <a href="${item.href}" class="nav-item ${activeClass}" data-page="${item.id}">
                <i class="fas ${item.icon}"></i> ${item.text}
            </a>
        `;
    });

    html += `</nav>`;

    // Footer
    html += `
        <div class="sidebar-footer">
            <a href="/logout" class="sidebar-logout" onclick="logout(event)">
                <i class="fas fa-sign-out-alt"></i> Logout
            </a>
            ${sidebarConfig.footer.contact ? `
            <div class="sidebar-contact">
                <i class="fas fa-phone"></i> ${sidebarConfig.footer.contact}
            </div>
            ` : ''}
            <div class="sidebar-version">${sidebarConfig.footer.version}</div>
            <div class="sidebar-copyright">${sidebarConfig.footer.copyright}</div>
        </div>
    `;

    sidebar.innerHTML = html;

    // Insert at beginning of body
    document.body.insertBefore(sidebar, document.body.firstChild);
}

function initSidebar() {
    // Add sidebar styles if not already present
    if (!document.getElementById('sidebar-styles')) {
        const style = document.createElement('style');
        style.id = 'sidebar-styles';
        style.textContent = `
            .sidebar {
                position: fixed;
                left: 0;
                top: 0;
                bottom: 0;
                width: 240px;
                background: var(--dark, #1e1b4b);
                border-right: 1px solid var(--border, rgba(255,255,255,0.1));
                padding: 1.5rem 1rem;
                display: flex;
                flex-direction: column;
                z-index: 100;
            }
            
            .sidebar-logo {
                display: flex;
                align-items: center;
                gap: 12px;
                font-size: 1.25rem;
                font-weight: 700;
                color: var(--light, #f8fafc);
                text-decoration: none;
                padding: 0.5rem;
                margin-bottom: 2rem;
            }
            
            .sidebar-logo i {
                color: var(--primary, #6366f1);
            }
            
            .sidebar nav {
                flex: 1;
                display: flex;
                flex-direction: column;
                gap: 0.25rem;
            }
            
            .sidebar .nav-item {
                display: flex;
                align-items: center;
                gap: 12px;
                padding: 0.75rem 1rem;
                color: var(--gray, #64748b);
                text-decoration: none;
                border-radius: 10px;
                transition: all 0.2s;
                font-weight: 500;
            }
            
            .sidebar .nav-item:hover {
                background: rgba(255, 255, 255, 0.05);
                color: var(--light, #f8fafc);
            }
            
            .sidebar .nav-item.active {
                background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(14, 165, 233, 0.1));
                color: var(--primary, #6366f1);
            }
            
            .sidebar .nav-item i {
                width: 20px;
                text-align: center;
            }
            
            .sidebar-footer {
                padding-top: 1rem;
                border-top: 1px solid var(--border, rgba(255,255,255,0.1));
                text-align: center;
            }
            
            .sidebar-version {
                font-size: 0.75rem;
                color: var(--primary, #6366f1);
                font-weight: 600;
            }
            
            .sidebar-copyright {
                font-size: 0.65rem;
                color: var(--gray, #64748b);
                margin-top: 0.25rem;
            }
            
            .sidebar-logout {
                display: flex;
                align-items: center;
                justify-content: center;
                gap: 8px;
                padding: 0.75rem;
                margin-bottom: 1rem;
                color: #ef4444;
                text-decoration: none;
                border-radius: 8px;
                transition: all 0.2s;
                font-weight: 500;
                border: 1px solid rgba(239, 68, 68, 0.2);
            }
            
            .sidebar-logout:hover {
                background: rgba(239, 68, 68, 0.1);
                color: #dc2626;
            }
            
            .sidebar-contact {
                display: flex;
                align-items: center;
                justify-content: center;
                gap: 8px;
                padding: 0.5rem;
                margin-bottom: 0.75rem;
                font-size: 0.75rem;
                color: var(--primary, #6366f1);
                font-weight: 500;
            }
            
            .sidebar-contact i {
                font-size: 0.85rem;
            }
            
            /* Mobile Sidebar Toggle */
            .sidebar-toggle {
                display: none;
                position: fixed;
                bottom: 1rem;
                left: 1rem;
                width: 50px;
                height: 50px;
                background: var(--primary, #6366f1);
                border: none;
                border-radius: 50%;
                color: white;
                font-size: 1.25rem;
                cursor: pointer;
                z-index: 101;
                box-shadow: 0 4px 20px rgba(99, 102, 241, 0.4);
            }
            
            @media (max-width: 768px) {
                .sidebar {
                    transform: translateX(-100%);
                    transition: transform 0.3s ease;
                }
                
                .sidebar.open {
                    transform: translateX(0);
                }
                
                .sidebar-toggle {
                    display: flex;
                    align-items: center;
                    justify-content: center;
                }
                
                .main-content {
                    margin-left: 0 !important;
                }
            }
        `;
        document.head.appendChild(style);
    }

    renderSidebar();

    // Add mobile toggle button
    const toggle = document.createElement('button');
    toggle.className = 'sidebar-toggle';
    toggle.innerHTML = '<i class="fas fa-bars"></i>';
    toggle.onclick = () => {
        document.getElementById('sidebar').classList.toggle('open');
    };
    document.body.appendChild(toggle);
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initSidebar);
} else {
    initSidebar();
}

// Export for manual use
window.GOACSSidebar = {
    config: sidebarConfig,
    render: renderSidebar,
    init: initSidebar,

    // Helper to add a new menu item dynamically
    addMenuItem: function (item, position = -1) {
        if (position === -1) {
            sidebarConfig.menuItems.push(item);
        } else {
            sidebarConfig.menuItems.splice(position, 0, item);
        }
        const sidebar = document.getElementById('sidebar');
        if (sidebar) {
            sidebar.remove();
            renderSidebar();
        }
    },

    // Helper to remove a menu item
    removeMenuItem: function (id) {
        const index = sidebarConfig.menuItems.findIndex(item => item.id === id);
        if (index > -1) {
            sidebarConfig.menuItems.splice(index, 1);
            const sidebar = document.getElementById('sidebar');
            if (sidebar) {
                sidebar.remove();
                renderSidebar();
            }
        }
    }
};

// Add theme toggle functionality
window.toggleTheme = function() {
    const body = document.body;
    const themeToggle = document.getElementById('theme-toggle');
    const icon = themeToggle ? themeToggle.querySelector('i') : null;
    
    if (body.classList.contains('light-theme')) {
        body.classList.remove('light-theme');
        localStorage.setItem('theme', 'dark');
        if (icon) icon.className = 'fas fa-moon';
    } else {
        body.classList.add('light-theme');
        localStorage.setItem('theme', 'light');
        if (icon) icon.className = 'fas fa-sun';
    }
};

// Initialize theme
window.initTheme = function() {
    const savedTheme = localStorage.getItem('theme');
    const body = document.body;
    const themeToggle = document.getElementById('theme-toggle');
    const icon = themeToggle ? themeToggle.querySelector('i') : null;
    
    if (savedTheme === 'light') {
        body.classList.add('light-theme');
        if (icon) icon.className = 'fas fa-sun';
    } else {
        if (icon) icon.className = 'fas fa-moon';
    }
};

// Show bottom navigation on mobile
window.initBottomNav = function() {
    const mediaQuery = window.matchMedia('(max-width: 768px)');
    
    function handleMediaChange(e) {
        const bottomNav = document.querySelector('.bottom-nav');
        if (!bottomNav) return;
        
        if (e.matches) {
            bottomNav.classList.add('show');
        } else {
            bottomNav.classList.remove('show');
        }
    }
    
    const bottomNav = document.querySelector('.bottom-nav');
    if (bottomNav) {
        handleMediaChange(mediaQuery);
        mediaQuery.addListener(handleMediaChange);
    }
};

// Scroll to section function
window.scrollToSection = function(sectionId) {
    const element = document.getElementById(sectionId);
    if (element) {
        element.scrollIntoView({ behavior: 'smooth' });
    }
};

// Initialize when page loads
window.addEventListener('DOMContentLoaded', function() {
    initTheme();
    initBottomNav();
});

// Logout function
window.logout = function(event) {
    event.preventDefault();
    
    if (confirm('Are you sure you want to logout?')) {
        // Clear token from localStorage
        localStorage.removeItem('token');
        
        // Call logout API
        fetch('/api/auth/logout', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        }).then(() => {
            // Redirect to login page
            window.location.href = '/';
        }).catch(err => {
            console.error('Logout error:', err);
            // Redirect anyway on error
            window.location.href = '/';
        });
    }
};
