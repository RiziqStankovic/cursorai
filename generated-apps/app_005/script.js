// Navigation functionality
document.addEventListener('DOMContentLoaded', function() {
    // Initialize navigation
    initNavigation();
    
    // Initialize copy buttons
    initCopyButtons();
    
    // Initialize toggle switches
    initToggleSwitches();
    
    // Initialize action buttons
    initActionButtons();
});

// Navigation system
function initNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const pages = document.querySelectorAll('.page');
    const pageTitle = document.querySelector('.page-title');
    
    // Page titles mapping
    const pageTitles = {
        'overview': 'Overview',
        'database': 'Database',
        'auth': 'Authentication',
        'storage': 'Storage',
        'api': 'API',
        'settings': 'Settings'
    };
    
    navItems.forEach(item => {
        item.addEventListener('click', function(e) {
            e.preventDefault();
            
            // Remove active class from all nav items
            navItems.forEach(nav => nav.classList.remove('active'));
            
            // Add active class to clicked item
            this.classList.add('active');
            
            // Get target page
            const targetPage = this.getAttribute('data-page');
            
            // Hide all pages
            pages.forEach(page => page.classList.add('hidden'));
            
            // Show target page
            const targetPageElement = document.getElementById(`${targetPage}-page`);
            if (targetPageElement) {
                targetPageElement.classList.remove('hidden');
                
                // Update page title
                if (pageTitle && pageTitles[targetPage]) {
                    pageTitle.textContent = pageTitles[targetPage];
                }
            }
        });
    });
}

// Copy to clipboard functionality
function initCopyButtons() {
    const copyButtons = document.querySelectorAll('.btn-copy');
    
    copyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const value = this.getAttribute('data-value') || 
                         this.previousElementSibling.value;
            
            // Copy to clipboard
            navigator.clipboard.writeText(value).then(() => {
                // Visual feedback
                const originalHTML = this.innerHTML;
                this.innerHTML = `
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                        <path d="M13 4L6 11l-3-3" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                `;
                this.style.color = 'var(--accent-success)';
                
                // Reset after 2 seconds
                setTimeout(() => {
                    this.innerHTML = originalHTML;
                    this.style.color = '';
                }, 2000);
            }).catch(err => {
                console.error('Failed to copy:', err);
                alert('Failed to copy to clipboard');
            });
        });
    });
}

// Toggle switches functionality
function initToggleSwitches() {
    const toggles = document.querySelectorAll('.toggle-switch input');
    
    toggles.forEach(toggle => {
        toggle.addEventListener('change', function() {
            const providerItem = this.closest('.provider-item');
            const statusElement = providerItem.querySelector('.provider-status');
            
            if (this.checked) {
                statusElement.textContent = 'Enabled';
                statusElement.classList.remove('disabled');
                statusElement.classList.add('enabled');
            } else {
                statusElement.textContent = 'Disabled';
                statusElement.classList.remove('enabled');
                statusElement.classList.add('disabled');
            }
        });
    });
}

// Action buttons functionality
function initActionButtons() {
    const actionButtons = document.querySelectorAll('.action-btn');
    
    actionButtons.forEach(button => {
        button.addEventListener('click', function() {
            const actionText = this.querySelector('span').textContent;
            
            // Visual feedback
            this.style.transform = 'scale(0.95)';
            setTimeout(() => {
                this.style.transform = '';
            }, 150);
            
            // Show notification (you can replace this with actual functionality)
            console.log(`Action clicked: ${actionText}`);
            
            // Simulate action
            showNotification(`${actionText} clicked!`, 'info');
        });
    });
}

// Notification system
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.style.cssText = `
        position: fixed;
        top: 80px;
        right: 24px;
        background-color: var(--bg-secondary);
        border: 1px solid var(--border-color);
        border-left: 4px solid ${
            type === 'success' ? 'var(--accent-success)' :
            type === 'error' ? 'var(--accent-danger)' :
            'var(--accent-primary)'
        };
        color: var(--text-primary);
        padding: 16px 20px;
        border-radius: 6px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        z-index: 1000;
        animation: slideIn 0.3s ease-out;
        max-width: 300px;
        font-size: 14px;
    `;
    notification.textContent = message;
    
    // Add animation
    const style = document.createElement('style');
    style.textContent = `
        @keyframes slideIn {
            from {
                transform: translateX(400px);
                opacity: 0;
            }
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }
        @keyframes slideOut {
            from {
                transform: translateX(0);
                opacity: 1;
            }
            to {
                transform: translateX(400px);
                opacity: 0;
            }
        }
    `;
    document.head.appendChild(style);
    
    document.body.appendChild(notification);
    
    // Remove after 3 seconds
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease-out';
        setTimeout(() => {
            notification.remove();
        }, 300);
    }, 3000);
}

// Table row click handlers (for database and storage pages)
document.addEventListener('DOMContentLoaded', function() {
    const tableRows = document.querySelectorAll('.data-table tbody tr');
    
    tableRows.forEach(row => {
        row.addEventListener('click', function(e) {
            // Don't trigger if clicking on action buttons
            if (e.target.closest('.table-actions')) {
                return;
            }
            
            // You can add functionality to view/edit table details here
            console.log('Row clicked:', this);
        });
    });
    
    // Delete button handlers
    const deleteButtons = document.querySelectorAll('.btn-icon-small[title="Delete"]');
    deleteButtons.forEach(button => {
        button.addEventListener('click', function(e) {
            e.stopPropagation();
            const row = this.closest('tr');
            const tableName = row.querySelector('.table-name span').textContent;
            
            if (confirm(`Are you sure you want to delete ${tableName}?`)) {
                row.style.animation = 'fadeOut 0.3s ease-out';
                setTimeout(() => {
                    row.remove();
                    showNotification(`${tableName} deleted successfully`, 'success');
                }, 300);
            }
        });
    });
    
    // Add fadeOut animation
    const style = document.createElement('style');
    style.textContent = `
        @keyframes fadeOut {
            from {
                opacity: 1;
                transform: translateX(0);
            }
            to {
                opacity: 0;
                transform: translateX(-20px);
            }
        }
    `;
    document.head.appendChild(style);
});

// Form submission handlers
document.addEventListener('DOMContentLoaded', function() {
    const saveButtons = document.querySelectorAll('.btn-primary');
    
    saveButtons.forEach(button => {
        if (button.textContent.includes('Save')) {
            button.addEventListener('click', function(e) {
                e.preventDefault();
                showNotification('Settings saved successfully!', 'success');
            });
        }
    });
    
    const cancelButtons = document.querySelectorAll('.btn-secondary');
    cancelButtons.forEach(button => {
        if (button.textContent.includes('Cancel')) {
            button.addEventListener('click', function(e) {
                e.preventDefault();
                // Reset form or navigate away
                showNotification('Changes cancelled', 'info');
            });
        }
    });
});

// Add smooth scrolling
document.addEventListener('DOMContentLoaded', function() {
    // Smooth scroll for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
});

// Simulate real-time updates (for demo purposes)
document.addEventListener('DOMContentLoaded', function() {
    // Update stats every 5 seconds
    setInterval(() => {
        const statValues = document.querySelectorAll('.stat-value');
        statValues.forEach(stat => {
            // Add a subtle pulse animation
            stat.style.animation = 'pulse 0.5s ease-in-out';
            setTimeout(() => {
                stat.style.animation = '';
            }, 500);
        });
    }, 5000);
    
    // Add pulse animation
    const style = document.createElement('style');
    style.textContent = `
        @keyframes pulse {
            0%, 100% {
                transform: scale(1);
            }
            50% {
                transform: scale(1.05);
            }
        }
    `;
    document.head.appendChild(style);
});

// Mobile menu toggle (for responsive design)
document.addEventListener('DOMContentLoaded', function() {
    // You can add a mobile menu button here if needed
    const createMobileMenuButton = () => {
        if (window.innerWidth <= 768) {
            const header = document.querySelector('.header-left');
            const menuButton = document.createElement('button');
            menuButton.className = 'mobile-menu-btn btn-icon';
            menuButton.innerHTML = `
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                    <path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                </svg>
            `;
            menuButton.addEventListener('click', () => {
                document.querySelector('.sidebar').classList.toggle('open');
            });
            header.insertBefore(menuButton, header.firstChild);
        }
    };
    
    // Check on load and resize
    createMobileMenuButton();
    window.addEventListener('resize', () => {
        const existingBtn = document.querySelector('.mobile-menu-btn');
        if (existingBtn && window.innerWidth > 768) {
            existingBtn.remove();
            document.querySelector('.sidebar').classList.remove('open');
        } else if (!existingBtn && window.innerWidth <= 768) {
            createMobileMenuButton();
        }
    });
});

// Keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Ctrl/Cmd + K for search (you can implement search functionality)
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        showNotification('Search functionality coming soon!', 'info');
    }
    
    // Escape to close modals/sidebar on mobile
    if (e.key === 'Escape') {
        if (window.innerWidth <= 768) {
            document.querySelector('.sidebar').classList.remove('open');
        }
    }
});

// Initialize tooltips (simple hover tooltips)
document.addEventListener('DOMContentLoaded', function() {
    const elementsWithTitle = document.querySelectorAll('[title]');
    elementsWithTitle.forEach(element => {
        element.addEventListener('mouseenter', function() {
            // Native browser tooltip will show
        });
    });
});
