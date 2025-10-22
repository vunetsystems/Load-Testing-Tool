package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"vuDataSim/src/logger"
)

// KubernetesPod represents the transformed pod data
type KubernetesPod struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Node        string `json:"node"`
	Ready       string `json:"ready"`
	Status      string `json:"status"`
	CPU         string `json:"cpu"`
	Mem         string `json:"mem"`
	Restarts    int    `json:"restarts"`
	LastRestart string `json:"last_restart"`
	IP          string `json:"ip"`
	QoS         string `json:"qos"`
	Age         string `json:"age"`
}

// HandleAPIGetKubernetesPods handles GET /api/kubernetes/pods
func HandleAPIGetKubernetesPods(w http.ResponseWriter, r *http.Request) {
	logger.LogWithNode("System", "Kubernetes", "Fetching pod data from Kubernetes API", "info")

	// Make request to Kubernetes API
	resp, err := http.Get("http://127.0.0.1:8001/api/v1/pods")
	if err != nil {
		logger.LogError("System", "Kubernetes", fmt.Sprintf("Failed to connect to Kubernetes API: %v", err))
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to connect to Kubernetes API: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.LogError("System", "Kubernetes", fmt.Sprintf("Kubernetes API returned status: %d", resp.StatusCode))
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Kubernetes API returned status: %d", resp.StatusCode),
		})
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError("System", "Kubernetes", fmt.Sprintf("Failed to read response body: %v", err))
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to read response from Kubernetes API",
		})
		return
	}

	// Parse the Kubernetes API response
	var k8sResponse struct {
		Items []struct {
			Metadata struct {
				Namespace         string    `json:"namespace"`
				Name              string    `json:"name"`
				CreationTimestamp time.Time `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				NodeName   string `json:"nodeName"`
				Containers []struct {
					Resources struct {
						Requests struct {
							CPU    string `json:"cpu,omitempty"`
							Memory string `json:"memory,omitempty"`
						} `json:"requests,omitempty"`
					} `json:"resources,omitempty"`
				} `json:"containers"`
			} `json:"spec"`
			Status struct {
				Phase              string `json:"phase"`
				PodIP              string `json:"podIP"`
				QoSClass           string `json:"qosClass"`
				ContainerStatuses []struct {
					Ready        bool   `json:"ready"`
					RestartCount int    `json:"restartCount"`
					State        struct {
						Terminated struct {
							FinishedAt time.Time `json:"finishedAt,omitempty"`
						} `json:"terminated,omitempty"`
					} `json:"state,omitempty"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &k8sResponse); err != nil {
		logger.LogError("System", "Kubernetes", fmt.Sprintf("Failed to parse Kubernetes API response: %v", err))
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to parse Kubernetes API response",
		})
		return
	}

	// Transform the data according to the jq filter
	var pods []KubernetesPod
	for _, item := range k8sResponse.Items {
		pod := KubernetesPod{
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			Node:      item.Spec.NodeName,
			Status:    item.Status.Phase,
			IP:        item.Status.PodIP,
			QoS:       item.Status.QoSClass,
			Age:       item.Metadata.CreationTimestamp.Format(time.RFC3339),
		}

		// Ready status
		var readyStatuses []string
		for _, cs := range item.Status.ContainerStatuses {
			if cs.Ready {
				readyStatuses = append(readyStatuses, "true")
			} else {
				readyStatuses = append(readyStatuses, "false")
			}
		}
		pod.Ready = fmt.Sprintf("%s", readyStatuses) // Simplified, join with "/"

		// CPU and Memory requests
		var cpus, mems []string
		for _, container := range item.Spec.Containers {
			if container.Resources.Requests.CPU != "" {
				cpus = append(cpus, container.Resources.Requests.CPU)
			}
			if container.Resources.Requests.Memory != "" {
				mems = append(mems, container.Resources.Requests.Memory)
			}
		}
		pod.CPU = fmt.Sprintf("%s", cpus) // Join with ", "
		pod.Mem = fmt.Sprintf("%s", mems) // Join with ", "

		// Restarts
		totalRestarts := 0
		for _, cs := range item.Status.ContainerStatuses {
			totalRestarts += cs.RestartCount
		}
		pod.Restarts = totalRestarts

		// Last restart
		var lastRestarts []string
		for _, cs := range item.Status.ContainerStatuses {
			if !cs.State.Terminated.FinishedAt.IsZero() {
				lastRestarts = append(lastRestarts, cs.State.Terminated.FinishedAt.Format(time.RFC3339))
			}
		}
		pod.LastRestart = fmt.Sprintf("%s", lastRestarts) // Join with ", "

		pods = append(pods, pod)
	}

	logger.LogWithNode("System", "Kubernetes", fmt.Sprintf("Successfully retrieved %d pods from Kubernetes API", len(pods)), "info")

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Kubernetes pods retrieved successfully",
		Data:    pods,
	})
}