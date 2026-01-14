// Traces Simulation JavaScript

(function() {
    'use strict';

    // DOM Elements
    const testNameInput = document.getElementById('traces-test-name');
    const epsInput = document.getElementById('traces-eps');
    const minutesInput = document.getElementById('traces-minutes');
    const startBtn = document.getElementById('start-traces-btn');
    const stopBtn = document.getElementById('stop-traces-btn');
    const refreshBtn = document.getElementById('refresh-traces-status-btn');
    const statusIndicator = document.getElementById('traces-status-indicator');
    const statusText = document.getElementById('traces-status-text');
    const statusDetails = document.getElementById('traces-status-details');
    const runningName = document.getElementById('traces-running-name');
    const runningEps = document.getElementById('traces-running-eps');
    const runningTime = document.getElementById('traces-running-time');

    // Polling interval (in milliseconds)
    const POLL_INTERVAL = 5000;
    let pollTimer = null;

    // Initialize
    function init() {
        if (!startBtn || !stopBtn) {
            console.log('Traces UI elements not found, skipping initialization');
            return;
        }

        // Attach event listeners
        startBtn.addEventListener('click', handleStart);
        stopBtn.addEventListener('click', handleStop);
        refreshBtn.addEventListener('click', refreshStatus);

        // Initial status check
        refreshStatus();
    }

    // Start traces simulation
    async function handleStart() {
        const testName = testNameInput.value.trim();
        const eps = parseInt(epsInput.value, 10);
        const minutes = parseInt(minutesInput.value, 10);

        // Validation
        if (!testName) {
            alert('Please enter a test name');
            testNameInput.focus();
            return;
        }

        if (!eps || eps <= 0) {
            alert('Please enter a valid EPS value (greater than 0)');
            epsInput.focus();
            return;
        }

        if (!minutes || minutes <= 0) {
            alert('Please enter a valid duration in minutes (greater than 0)');
            minutesInput.focus();
            return;
        }

        // Disable start button and show loading state
        startBtn.disabled = true;
        startBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Starting...</span>';

        try {
            const response = await fetch('/api/traces/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    test_name: testName,
                    eps: eps,
                    minutes: minutes
                })
            });

            const data = await response.json();

            if (data.success) {
                updateUIForRunning(testName, eps);
                startPolling();
            } else {
                alert('Failed to start traces simulation: ' + data.message);
                resetStartButton();
            }
        } catch (error) {
            console.error('Error starting traces simulation:', error);
            alert('Error starting traces simulation: ' + error.message);
            resetStartButton();
        }
    }

    // Stop traces simulation
    async function handleStop() {
        stopBtn.disabled = true;
        stopBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Stopping...</span>';

        try {
            const response = await fetch('/api/traces/stop', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            const data = await response.json();

            if (data.success) {
                updateUIForIdle();
                stopPolling();
            } else {
                alert('Failed to stop traces simulation: ' + data.message);
            }
        } catch (error) {
            console.error('Error stopping traces simulation:', error);
            alert('Error stopping traces simulation: ' + error.message);
        }

        // Reset stop button
        stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop</span>';
    }

    // Refresh status from server
    async function refreshStatus() {
        try {
            const response = await fetch('/api/traces/status');
            const data = await response.json();

            if (data.success && data.data) {
                const status = data.data;

                if (status.running) {
                    updateUIForRunning(status.test_name, status.eps, status.started_at);
                    startPolling();
                } else {
                    updateUIForIdle();
                    stopPolling();
                }
            }
        } catch (error) {
            console.error('Error fetching traces status:', error);
        }
    }

    // Update UI for running state
    function updateUIForRunning(testName, eps, startedAt) {
        startBtn.disabled = true;
        startBtn.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start Traces Test</span>';

        stopBtn.disabled = false;
        stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop</span>';

        statusIndicator.classList.remove('bg-slate-400', 'bg-red-500');
        statusIndicator.classList.add('bg-green-500');
        statusText.textContent = 'Status: Running';
        statusText.classList.remove('text-slate-600');
        statusText.classList.add('text-green-600');

        statusDetails.classList.remove('hidden');
        runningName.textContent = testName || '-';
        runningEps.textContent = eps || '-';

        if (startedAt) {
            const startTime = new Date(startedAt);
            runningTime.textContent = startTime.toLocaleTimeString();
        } else {
            runningTime.textContent = new Date().toLocaleTimeString();
        }

        // Disable input fields while running
        testNameInput.disabled = true;
        epsInput.disabled = true;
        minutesInput.disabled = true;
    }

    // Update UI for idle state
    function updateUIForIdle() {
        startBtn.disabled = false;
        startBtn.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start Traces Test</span>';

        stopBtn.disabled = true;
        stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop</span>';

        statusIndicator.classList.remove('bg-green-500', 'bg-red-500');
        statusIndicator.classList.add('bg-slate-400');
        statusText.textContent = 'Status: Idle';
        statusText.classList.remove('text-green-600');
        statusText.classList.add('text-slate-600');

        statusDetails.classList.add('hidden');

        // Enable input fields
        testNameInput.disabled = false;
        epsInput.disabled = false;
        minutesInput.disabled = false;
    }

    // Reset start button to default state
    function resetStartButton() {
        startBtn.disabled = false;
        startBtn.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start Traces Test</span>';
    }

    // Start polling for status updates
    function startPolling() {
        if (pollTimer) return; // Already polling

        pollTimer = setInterval(refreshStatus, POLL_INTERVAL);
    }

    // Stop polling
    function stopPolling() {
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = null;
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
