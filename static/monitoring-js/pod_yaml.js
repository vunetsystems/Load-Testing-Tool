// ==============================================
// Pod YAML JavaScript Module
// Handles fetching, cleaning, formatting, and displaying pod YAML data
// ==============================================

// ✅ Include these in your HTML once:
// <script src="https://cdn.jsdelivr.net/npm/js-yaml@4.1.0/dist/js-yaml.min.js"></script>
// <link href="https://cdn.jsdelivr.net/npm/prismjs/themes/prism-tomorrow.min.css" rel="stylesheet">
// <script src="https://cdn.jsdelivr.net/npm/prismjs/prism.js"></script>
// <script src="https://cdn.jsdelivr.net/npm/prismjs/components/prism-yaml.min.js"></script>

class PodYAMLManager {
  constructor() {
    this.yamlCache = {}; // Cache YAML data by pod name
    this.isLoading = false;
  }

  // ==============================================
  // Fetch YAML for a specific pod
  // ==============================================
  async fetchPodYAML(namespace, podName) {
    // Return cached data if already available
    if (this.yamlCache[podName]) {
      return this.yamlCache[podName];
    }

    if (this.isLoading) {
      // Wait for ongoing request
      return new Promise((resolve, reject) => {
        const checkCache = () => {
          if (this.yamlCache[podName]) {
            resolve(this.yamlCache[podName]);
          } else {
            setTimeout(checkCache, 100);
          }
        };
        // Timeout after 30 seconds
        setTimeout(() => reject(new Error('YAML fetch timeout')), 30000);
        checkCache();
      });
    }

    this.isLoading = true;

    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 30000); // 30-second timeout

      const response = await fetch(
        `/api/monitoring/k8/pod-yaml?namespace=${encodeURIComponent(namespace)}&podName=${encodeURIComponent(podName)}`,
        { signal: controller.signal }
      );

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result = await response.json();

      if (result.success && result.data) {
        // Parse JSON safely
        let parsedData;
        try {
          parsedData = typeof result.data === 'string' ? JSON.parse(result.data) : result.data;
        } catch (e) {
          console.warn('YAML data was already in YAML format.');
          parsedData = result.data;
        }

        // 🧹 Clean noisy K8s fields
        const cleanedData = this.removeK8sNoise(parsedData);

        // Convert JSON → YAML with js-yaml
        const formattedYAML = jsyaml.dump(cleanedData, { indent: 2, lineWidth: -1 });

        // Cache the formatted YAML
        this.yamlCache[podName] = formattedYAML;

        return formattedYAML;
      } else {
        throw new Error(result.message || 'Failed to fetch pod YAML');
      }
    } catch (error) {
      if (error.name === 'AbortError') {
        throw new Error('YAML fetch timed out after 30 seconds');
      }
      console.error('Error fetching pod YAML:', error);
      throw error;
    } finally {
      this.isLoading = false;
    }
  }

  // ==============================================
  // 🧽 Remove unwanted K8s internal fields
  // ==============================================
  removeK8sNoise(obj) {
    if (!obj || typeof obj !== 'object') return obj;
    const clone = JSON.parse(JSON.stringify(obj));

    if (clone.metadata) {
      delete clone.metadata.managedFields;
      delete clone.metadata.finalizers;
      delete clone.metadata.selfLink;
      delete clone.metadata.generation;
      delete clone.metadata.ownerReferences?.[0]?.controller;
      delete clone.metadata.ownerReferences?.[0]?.blockOwnerDeletion;
    }

    // Remove other noisy sections you don’t need in the UI
    delete clone.status; // removes conditions, podIPs, etc.
    delete clone.spec?.securityContext; // optional
    delete clone.spec?.tolerations; // optional

    return clone;
  }

  // ==============================================
  // Display YAML in the YAML tab (with syntax highlighting)
  // ==============================================
  displayYAML(podName) {
    const yamlContent = document.getElementById('pod-yaml-content');
    if (!yamlContent) return;

    const yamlData = this.yamlCache[podName];

    if (yamlData) {
      // Highlight YAML using Prism.js
      const highlighted = Prism.highlight(yamlData, Prism.languages.yaml, 'yaml');

      yamlContent.innerHTML = `
        <code class="language-yaml text-yellow-100">${highlighted}</code>
      `;
    } else {
      yamlContent.innerHTML = `<code class="text-slate-400 italic">Loading YAML...</code>`;

      // Try to fetch from cache fallback
      this.fetchPodYAMLFromCache(podName)
        .then(() => {
          const updatedData = this.yamlCache[podName];
          if (updatedData) {
            const highlighted = Prism.highlight(updatedData, Prism.languages.yaml, 'yaml');
            yamlContent.innerHTML = `<code class="language-yaml text-yellow-100">${highlighted}</code>`;
          } else {
            yamlContent.innerHTML = `<code class="text-red-400">Failed to load YAML data. Please try again.</code>`;
          }
        })
        .catch((error) => {
          console.error('Error displaying YAML:', error);
          yamlContent.innerHTML = `<code class="text-red-400">Error loading YAML data. Check console for details.</code>`;
        });
    }
  }

  // ==============================================
  // Helper: Fetch from cache with error handling
  // ==============================================
  async fetchPodYAMLFromCache(podName) {
    const cached = this.getCachedYAML(podName);
    if (cached) {
      return cached;
    }
    throw new Error('YAML data not available in cache');
  }

  // ==============================================
  // Cache utilities
  // ==============================================
  clearCache(podName) {
    delete this.yamlCache[podName];
  }

  clearAllCache() {
    this.yamlCache = {};
  }

  getCachedYAML(podName) {
    return this.yamlCache[podName] || null;
  }
}

// ==============================================
// Export for use in other modules
// ==============================================
window.PodYAMLManager = PodYAMLManager;
