// Sample deployment data
const sampleDeployments = [
    {
        title: 'v2.1.0 - Production',
        meta: 'Deployed by John Doe • 2 hours ago',
        status: 'success'
    },
    {
        title: 'v2.0.9 - Staging',
        meta: 'Deployed by Jane Smith • 5 hours ago',
        status: 'success'
    },
    {
        title: 'v2.0.8 - Development',
        meta: 'Deployed by Bob Wilson • 1 day ago',
        status: 'success'
    },
    {
        title: 'v2.0.7 - Production',
        meta: 'Deployed by Alice Brown • 2 days ago',
        status: 'success'
    }
];

// Sample log entries
const sampleLogs = [
    { level: 'info', message: 'Server initialized successfully' },
    { level: 'success', message: 'Database connection established' },
    { level: 'info', message: 'API endpoints loaded' },
    { level: 'warning', message: 'High memory usage detected on server-1' },
    { level: 'success', message: 'Deployment v2.1.0 completed successfully' },
    { level: 'info', message: 'Health check passed' },
    { level: 'error', message: 'Failed to connect to backup server (retrying...)' },
    { level: 'info', message: 'Cache cleared successfully' },
    { level: 'success', message: 'SSL certificate renewed' },
    { level: 'info', message: 'System backup completed' }
];

// Initialize the dashboard
function initDashboard() {
    loadDeployments();
    startMetricsUpdate();
}

// Load deployment history
function loadDeployments() {
    const deploymentList = document.getElementById('deploymentList');
    deploymentList.innerHTML = '';

    sampleDeployments.forEach(deployment => {
        const deploymentItem = document.createElement('div');
        deploymentItem.className = 'deployment-item';
        
        deploymentItem.innerHTML = `
            <div class="deployment-info">
                <div class="deployment-title">${deployment.title}</div>
                <div class="deployment-meta">${deployment.meta}</div>
            </div>
            <span class="deployment-status status-${deployment.status}">
                ${deployment.status === 'success' ? '✓ Success' : 
                  deployment.status === 'pending' ? '⏳ Pending' : '✗ Failed'}
            </span>
        `;
        
        deploymentList.appendChild(deploymentItem);
    });
}

// Update server metrics randomly
function updateMetrics() {
    const servers = [
        { id: 1, cpu: 'cpu1', memory: 'memory1' },
        { id: 2, cpu: 'cpu2', memory: 'memory2' },
        { id: 3, cpu: 'cpu3', memory: 'memory3' }
    ];

    servers.forEach(server => {
        const cpuElement = document.getElementById(server.cpu);
        const memoryElement = document.getElementById(server.memory);
        
        if (cpuElement && memoryElement) {
            // Generate random values within reasonable ranges
            const cpuValue = Math.floor(Math.random() * 40) + 20; // 20-60%
            const memoryValue = Math.floor(Math.random() * 40) + 30; // 30-70%
            
            cpuElement.textContent = `${cpuValue}%`;
            memoryElement.textContent = `${memoryValue}%`;
        }
    });
}

// Start automatic metrics update
function startMetricsUpdate() {
    updateMetrics(); // Initial update
    setInterval(updateMetrics, 5000); // Update every 5 seconds
}

// Trigger deployment
function triggerDeployment() {
    const deployBtn = document.querySelector('.deploy-btn');
    const originalText = deployBtn.innerHTML;
    
    // Show loading state
    deployBtn.innerHTML = '<span>⏳</span><span>Deploying...</span>';
    deployBtn.disabled = true;
    deployBtn.style.opacity = '0.7';
    
    // Simulate deployment process
    setTimeout(() => {
        // Add new deployment to the top
        const deploymentList = document.getElementById('deploymentList');
        const newDeployment = document.createElement('div');
        newDeployment.className = 'deployment-item';
        
        const now = new Date();
        const timeAgo = 'just now';
        
        newDeployment.innerHTML = `
            <div class="deployment-info">
                <div class="deployment-title">v2.1.1 - Production</div>
                <div class="deployment-meta">Deployed by System • ${timeAgo}</div>
            </div>
            <span class="deployment-status status-success">
                ✓ Success
            </span>
        `;
        
        deploymentList.insertBefore(newDeployment, deploymentList.firstChild);
        
        // Restore button
        deployBtn.innerHTML = originalText;
        deployBtn.disabled = false;
        deployBtn.style.opacity = '1';
        
        // Show notification (you can enhance this with a toast notification)
        addLogEntry('success', 'Deployment v2.1.1 completed successfully');
    }, 2000);
}

// Refresh metrics
function refreshMetrics() {
    const refreshBtn = document.querySelector('.refresh-btn');
    const originalText = refreshBtn.innerHTML;
    
    // Show loading state
    refreshBtn.innerHTML = '<span>🔄</span><span>Refreshing...</span>';
    refreshBtn.disabled = true;
    
    // Update metrics
    updateMetrics();
    
    // Restore button
    setTimeout(() => {
        refreshBtn.innerHTML = originalText;
        refreshBtn.disabled = false;
    }, 1000);
}

// View logs
function viewLogs() {
    const logsSection = document.getElementById('logsSection');
    const logsContent = document.getElementById('logsContent');
    
    // Display logs
    logsContent.innerHTML = '';
    sampleLogs.forEach(log => {
        addLogEntry(log.level, log.message);
    });
    
    // Show logs section
    logsSection.style.display = 'flex';
    logsSection.scrollIntoView({ behavior: 'smooth' });
}

// Close logs
function closeLogs() {
    const logsSection = document.getElementById('logsSection');
    logsSection.style.display = 'none';
}

// Add log entry
function addLogEntry(level, message) {
    const logsContent = document.getElementById('logsContent');
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${level}`;
    logEntry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logsContent.appendChild(logEntry);
    
    // Auto-scroll to bottom
    logsContent.scrollTop = logsContent.scrollHeight;
}

// Initialize dashboard when page loads
document.addEventListener('DOMContentLoaded', initDashboard);

// Add some dynamic log entries periodically
setInterval(() => {
    const logsSection = document.getElementById('logsSection');
    if (logsSection.style.display !== 'none') {
        const levels = ['info', 'success', 'warning'];
        const messages = [
            'Health check completed',
            'Cache updated',
            'Request processed successfully',
            'Monitoring system active',
            'Backup scheduled'
        ];
        
        const randomLevel = levels[Math.floor(Math.random() * levels.length)];
        const randomMessage = messages[Math.floor(Math.random() * messages.length)];
        
        addLogEntry(randomLevel, randomMessage);
    }
}, 10000); // Add a log entry every 10 seconds if logs are open
