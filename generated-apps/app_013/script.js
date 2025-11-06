// CI/CD Pipeline Monitoring Dashboard - JavaScript

// Mock data generator
const generateMockJobs = (count = 100) => {
    const statuses = ['success', 'failed', 'running', 'pending'];
    const pipelines = ['Frontend CI', 'Backend CI', 'E2E Tests', 'Deployment', 'Security Scan', 'Code Quality'];
    const branches = ['main', 'develop', 'feature/new-feature', 'hotfix/bugfix', 'release/v1.0'];
    const jobs = [];

    for (let i = 1; i <= count; i++) {
        const status = statuses[Math.floor(Math.random() * statuses.length)];
        const startTime = new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000);
        const duration = status === 'running' ? null : Math.floor(Math.random() * 3600) + 60;
        
        jobs.push({
            id: `JOB-${String(i).padStart(6, '0')}`,
            name: `${pipelines[Math.floor(Math.random() * pipelines.length)]} #${i}`,
            status: status,
            pipeline: pipelines[Math.floor(Math.random() * pipelines.length)],
            branch: branches[Math.floor(Math.random() * branches.length)],
            duration: duration,
            started: startTime
        });
    }

    return jobs;
};

// State management
let allJobs = [];
let filteredJobs = [];
let currentPage = 1;
let itemsPerPage = 25;
let autoRefreshInterval = null;
let autoRefreshEnabled = false;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    allJobs = generateMockJobs(150);
    filteredJobs = [...allJobs];
    
    initializeEventListeners();
    updateStats();
    renderTable();
    updatePagination();
});

// Event Listeners
function initializeEventListeners() {
    // Refresh button
    document.getElementById('refreshBtn').addEventListener('click', () => {
        refreshData();
    });

    // Auto refresh toggle
    document.getElementById('autoRefreshBtn').addEventListener('click', () => {
        toggleAutoRefresh();
    });

    // Status filter
    document.getElementById('statusFilter').addEventListener('change', (e) => {
        applyFilters();
    });

    // Search input
    document.getElementById('searchInput').addEventListener('input', (e) => {
        applyFilters();
    });

    // Items per page
    document.getElementById('itemsPerPage').addEventListener('change', (e) => {
        itemsPerPage = parseInt(e.target.value);
        currentPage = 1;
        renderTable();
        updatePagination();
    });

    // Pagination buttons
    document.getElementById('firstPageBtn').addEventListener('click', () => {
        currentPage = 1;
        renderTable();
        updatePagination();
    });

    document.getElementById('prevPageBtn').addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            renderTable();
            updatePagination();
        }
    });

    document.getElementById('nextPageBtn').addEventListener('click', () => {
        const totalPages = Math.ceil(filteredJobs.length / itemsPerPage);
        if (currentPage < totalPages) {
            currentPage++;
            renderTable();
            updatePagination();
        }
    });

    document.getElementById('lastPageBtn').addEventListener('click', () => {
        const totalPages = Math.ceil(filteredJobs.length / itemsPerPage);
        currentPage = totalPages;
        renderTable();
        updatePagination();
    });
}

// Filter and search functionality
function applyFilters() {
    const statusFilter = document.getElementById('statusFilter').value;
    const searchQuery = document.getElementById('searchInput').value.toLowerCase().trim();

    filteredJobs = allJobs.filter(job => {
        const statusMatch = statusFilter === 'all' || job.status === statusFilter;
        const searchMatch = !searchQuery || 
            job.name.toLowerCase().includes(searchQuery) ||
            job.id.toLowerCase().includes(searchQuery) ||
            job.pipeline.toLowerCase().includes(searchQuery) ||
            job.branch.toLowerCase().includes(searchQuery);
        
        return statusMatch && searchMatch;
    });

    currentPage = 1;
    updateStats();
    renderTable();
    updatePagination();
}

// Update statistics
function updateStats() {
    const stats = {
        success: 0,
        failed: 0,
        running: 0,
        pending: 0,
        total: filteredJobs.length
    };

    filteredJobs.forEach(job => {
        if (stats.hasOwnProperty(job.status)) {
            stats[job.status]++;
        }
    });

    document.getElementById('successCount').textContent = stats.success;
    document.getElementById('failedCount').textContent = stats.failed;
    document.getElementById('runningCount').textContent = stats.running;
    document.getElementById('pendingCount').textContent = stats.pending;
    document.getElementById('totalCount').textContent = stats.total;
}

// Render table
function renderTable() {
    const tbody = document.getElementById('jobsTableBody');
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    const jobsToDisplay = filteredJobs.slice(startIndex, endIndex);

    if (jobsToDisplay.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="8" class="empty-state">
                    <div class="empty-state-icon">📭</div>
                    <div class="empty-state-text">No jobs found matching your filters</div>
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = jobsToDisplay.map(job => `
        <tr>
            <td><strong>${job.id}</strong></td>
            <td>${job.name}</td>
            <td><span class="status-badge ${job.status}">${job.status}</span></td>
            <td>${job.pipeline}</td>
            <td><code>${job.branch}</code></td>
            <td>${formatDuration(job.duration)}</td>
            <td>${formatDate(job.started)}</td>
            <td>
                <button class="action-btn" onclick="viewJobDetails('${job.id}')">View</button>
            </td>
        </tr>
    `).join('');
}

// Format duration
function formatDuration(seconds) {
    if (seconds === null || seconds === undefined) {
        return '<span class="loading">Running...</span>';
    }
    
    if (seconds < 60) {
        return `${seconds}s`;
    } else if (seconds < 3600) {
        const minutes = Math.floor(seconds / 60);
        const secs = seconds % 60;
        return `${minutes}m ${secs}s`;
    } else {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        return `${hours}h ${minutes}m`;
    }
}

// Format date
function formatDate(date) {
    if (!(date instanceof Date)) {
        date = new Date(date);
    }
    
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) {
        return 'Just now';
    } else if (diffMins < 60) {
        return `${diffMins} minute${diffMins > 1 ? 's' : ''} ago`;
    } else if (diffHours < 24) {
        return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    } else if (diffDays < 7) {
        return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
    } else {
        return date.toLocaleDateString('en-US', { 
            month: 'short', 
            day: 'numeric', 
            year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined 
        });
    }
}

// Update pagination
function updatePagination() {
    const totalPages = Math.ceil(filteredJobs.length / itemsPerPage);
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = Math.min(startIndex + itemsPerPage, filteredJobs.length);

    document.getElementById('currentPage').textContent = currentPage;
    document.getElementById('totalPages').textContent = totalPages || 1;
    document.getElementById('showingFrom').textContent = filteredJobs.length > 0 ? startIndex + 1 : 0;
    document.getElementById('showingTo').textContent = endIndex;
    document.getElementById('totalFiltered').textContent = filteredJobs.length;

    // Update button states
    document.getElementById('firstPageBtn').disabled = currentPage === 1;
    document.getElementById('prevPageBtn').disabled = currentPage === 1;
    document.getElementById('nextPageBtn').disabled = currentPage >= totalPages;
    document.getElementById('lastPageBtn').disabled = currentPage >= totalPages;
}

// Refresh data
function refreshData() {
    // Simulate API call
    const refreshBtn = document.getElementById('refreshBtn');
    refreshBtn.disabled = true;
    refreshBtn.textContent = 'Refreshing...';

    // Update some running jobs randomly
    setTimeout(() => {
        allJobs.forEach(job => {
            if (job.status === 'running' && Math.random() > 0.7) {
                job.status = Math.random() > 0.5 ? 'success' : 'failed';
                job.duration = Math.floor(Math.random() * 3600) + 60;
            } else if (job.status === 'pending' && Math.random() > 0.8) {
                job.status = 'running';
                job.duration = null;
            }
        });

        // Add some new jobs occasionally
        if (Math.random() > 0.7) {
            const newJobs = generateMockJobs(5);
            allJobs = [...newJobs, ...allJobs].slice(0, 200);
        }

        applyFilters();
        refreshBtn.disabled = false;
        refreshBtn.textContent = 'Refresh';
    }, 1000);
}

// Toggle auto refresh
function toggleAutoRefresh() {
    autoRefreshEnabled = !autoRefreshEnabled;
    const autoRefreshBtn = document.getElementById('autoRefreshBtn');

    if (autoRefreshEnabled) {
        autoRefreshBtn.textContent = 'Auto Refresh: ON';
        autoRefreshBtn.classList.add('active');
        autoRefreshInterval = setInterval(() => {
            refreshData();
        }, 30000); // Refresh every 30 seconds
    } else {
        autoRefreshBtn.textContent = 'Auto Refresh: OFF';
        autoRefreshBtn.classList.remove('active');
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
            autoRefreshInterval = null;
        }
    }
}

// View job details (placeholder)
function viewJobDetails(jobId) {
    const job = allJobs.find(j => j.id === jobId);
    if (job) {
        alert(`Job Details:\n\nID: ${job.id}\nName: ${job.name}\nStatus: ${job.status}\nPipeline: ${job.pipeline}\nBranch: ${job.branch}\nDuration: ${formatDuration(job.duration)}\nStarted: ${formatDate(job.started)}`);
    }
}

// Simulate real-time updates for running jobs
setInterval(() => {
    allJobs.forEach(job => {
        if (job.status === 'running') {
            // Randomly complete some running jobs
            if (Math.random() > 0.95) {
                job.status = Math.random() > 0.3 ? 'success' : 'failed';
                job.duration = Math.floor(Math.random() * 3600) + 60;
            }
        }
    });
    
    if (filteredJobs.length > 0) {
        applyFilters();
    }
}, 5000); // Check every 5 seconds
