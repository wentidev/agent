package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	networkingv1 "k8s.io/api/networking/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type IngressInfo struct {
	Name string `json:"name"`

	// optional
	Description string `json:"description"`
	Target      string `json:"target"`
	Port        string `json:"port"`
	Protocol    string `json:"protocol"`
	Path        string `json:"path"`
	Method      string `json:"method"`
	Timeout     string `json:"timeout"`
	Interval    string `json:"interval"`
	HTTPCode    string `json:"httpCode"`
	Enabled     bool   `json:"enabled"`
}

var HealthCheckPath string = "wenti.dev/health-check-path"
var HealthCheckProtocol string = "wenti.dev/health-check-protocol"
var HealthCheckMethod string = "wenti.dev/health-check-method"
var HealthCheckHTTPCode string = "wenti.dev/health-check-success-codes"
var HealthCheckTimeout string = "wenti.dev/health-check-timeout"
var HealthCheckInterval string = "wenti.dev/health-check-interval"
var HealthCheckPort string = "wenti.dev/health-check-port"

// Find value of annotation in ingress
func GetStringAnnotation(ingress *networkingv1.Ingress, annotation string) string {
	if ingress.Annotations == nil {
		return ""
	}
	if value, ok := ingress.Annotations[annotation]; ok {
		return value
	}
	return ""
}

type HealthCheck struct {
	Count      int `json:"count"`
	HTTPChecks []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Target      string `json:"target"`
		Port        int    `json:"port"`
		Protocol    string `json:"protocol"`
		Path        string `json:"path"`
		Method      string `json:"method"`
		Timeout     int    `json:"timeout"`
		Interval    int    `json:"interval"`
		ValidStatus int    `json:"valid-status"`
		ContentType any    `json:"contentType"`
		Headers     any    `json:"headers"`
		Query       any    `json:"query"`
		Body        any    `json:"body"`
		Labels      any    `json:"labels"`
	} `json:"http-checks"`
}

func FindHealthCheck(name string) (bool, string) {
	log.Log.Info("looking for health check", "Name", name)
	resp, err := ExecuteAPIRequest("GET", "/api/v1/healthchecks", nil)
	if err != nil {
		log.Log.Error(err, "unable to retrieve health checks")
		return false, ""
	}
	var healthChecks HealthCheck
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log.Error(err, "unable to read response body")
		return false, ""
	}
	err = json.Unmarshal(data, &healthChecks)
	if err != nil {
		log.Log.Error(err, "unable to unmarshal health checks")
		return false, ""
	}
	log.Log.Info(("health checks found"), "HealthChecks", healthChecks)
	for _, healthCheck := range healthChecks.HTTPChecks {
		if healthCheck.Name == name {
			log.Log.Info(("health check found"), "Name", healthCheck.Name)
			log.Log.Info(("health check found"), "ID", healthCheck.ID)
			return true, healthCheck.ID
		}
	}
	return false, ""
}

func ExecuteAPIRequest(method string, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", AppURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AppToken))

	// Send the request
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	//defer resp.Body.Close()
	return resp, nil
}

func DeleteHealthCheck(resource IngressInfo) (string, error) {
	findBool, healthCheckID := FindHealthCheck(resource.Name)
	if !findBool {
		log.Log.Info("health check does not exist for bool")
		return "", nil
	}
	log.Log.Info("health check exists with ID", "HealthCheckID", healthCheckID)
	request, err := ExecuteAPIRequest("DELETE", fmt.Sprintf("/api/v1/healthchecks/%s", healthCheckID), nil)
	if err != nil {
		return "", err
	}
	log.Log.Info("health check deleted", "Status", request.Status)
	return request.Status, nil
}

func CreateOrUpdateHealthCheck(resource IngressInfo) (string, error) {
	jsonData, err := json.Marshal(resource)
	findBool, healthCheckID := FindHealthCheck(resource.Name)
	if findBool {
		updateRequest, err := ExecuteAPIRequest("PUT", fmt.Sprintf("/api/v1/healthchecks/%s", healthCheckID), jsonData)
		if err != nil {
			return "", err
		}
		log.Log.Info("health check updated", "Status", updateRequest.Status)
		log.Log.Info("health check already exists")
		return "", nil
	}
	log.Log.Info("health check does not exist, creating it")
	if err != nil {
		return "", err
	}
	request, err := ExecuteAPIRequest("POST", "/api/v1/healthchecks", jsonData)
	if err != nil {
		return "", err
	}
	return request.Status, nil
}
