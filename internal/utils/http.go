package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	clientsdk "github.com/wentidev/sdk-go"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var HealthCheckPath string = "wenti.dev/health-check-path"
var HealthCheckProtocol string = "wenti.dev/health-check-protocol"
var HealthCheckMethod string = "wenti.dev/health-check-method"
var HealthCheckHTTPCode string = "wenti.dev/health-check-success-codes"
var HealthCheckTimeout string = "wenti.dev/health-check-timeout"
var HealthCheckInterval string = "wenti.dev/health-check-interval"
var HealthCheckPort string = "wenti.dev/health-check-port"

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

// GetStringAnnotation Find value of annotation in ingress
func GetStringAnnotation(ingress *networkingv1.Ingress, annotation string) string {
	if ingress.Annotations == nil {
		return ""
	}
	if value, ok := ingress.Annotations[annotation]; ok {
		return value
	}
	return ""
}

func HeaderInterceptor(ctx context.Context, req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", AppToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func CreateClient() (*clientsdk.ClientWithResponses, error) {
	client, err := clientsdk.NewClientWithResponses(AppURL, clientsdk.WithRequestEditorFn(HeaderInterceptor))
	if err != nil {
		log.Log.Error(err, "unable to create client")
		return nil, err
	}
	return client, nil
}

func FindHealthCheck(name string) (bool, string) {
	log.Log.Info("looking for health check", "Name", name)
	client, err := CreateClient()
	if err != nil {
		log.Log.Error(err, "unable to create client")
		return false, ""
	}
	resp, err := client.GetApiV1HealthchecksWithResponse(context.Background())
	if err != nil {
		log.Log.Error(err, "unable to retrieve health checks")
		return false, ""
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK || resp.HTTPResponse.Header.Get("Content-Type") != "application/json" {
		log.Log.Error(err, "statusCode or Content-Type is not valid")
		return false, ""
	}

	if resp.JSON200 == nil {
		log.Log.Error(err, "JSON200 is nil")
		return false, ""
	}

	for _, healthCheck := range *resp.JSON200.HttpChecks {
		if healthCheck.Name == &name {
			log.Log.Info("health check found", "Name", healthCheck.Name)
			log.Log.Info("health check found", "ID", healthCheck.Id)
			return true, *healthCheck.Id
		}
	}
	return false, ""
}

func DeleteHealthCheck(resource IngressInfo) (string, error) {
	findBool, healthCheckID := FindHealthCheck(resource.Name)
	if !findBool {
		log.Log.Info("health check does not exist for bool")
		return "", nil
	}
	log.Log.Info("health check exists with ID", "HealthCheckID", healthCheckID)
	err := wentiApiDeleteHealthCheck(healthCheckID)
	if err != nil {
		return "", err
	}
	return "deleted", nil
}

func CreateOrUpdateHealthCheck(resource IngressInfo) (string, error) {
	findBool, healthCheckID := FindHealthCheck(resource.Name)

	if findBool {
		status, err := wentiApiUpdateHealthCheck(resource, healthCheckID)
		if err != nil {
			return "", err
		}
		return status, nil
	}
	log.Log.Info("health check does not exist, creating it")
	status, err := wentiApiCreateHealthCheck(resource)
	if err != nil {
		return "", err
	}
	return status, nil

}

func wentiApiDeleteHealthCheck(HealthCheckId string) error {
	client, err := CreateClient()
	if err != nil {
		log.Log.Error(err, "(delete) unable to update client")
		return err
	}
	resp, err := client.DeleteApiV1HealthchecksIdWithResponse(context.Background(), HealthCheckId)
	if err != nil {
		log.Log.Error(err, "(delete) error in API")
		return err
	}

	if resp.HTTPResponse.StatusCode != http.StatusNoContent || resp.HTTPResponse.Header.Get("Content-Type") != "application/json" {
		log.Log.Error(err, "(delete) statusCode or Content-Type is not valid")
		return err
	}
	return nil
}

func wentiApiCreateHealthCheck(resource IngressInfo) (string, error) {
	client, err := CreateClient()
	if err != nil {
		log.Log.Error(err, "(create) unable to update client")
		return "", err
	}

	// Convert for interval
	interval, err := ConvertStringToInt(resource.Interval)
	if err != nil {
		log.Log.Error(err, "(create) unable to convert string to int")
		return "", err
	}

	// Convert for timeout
	timeout, err := ConvertStringToInt(resource.Timeout)
	if err != nil {
		log.Log.Error(err, "(create) unable to convert string to int")
		return "", err
	}

	// Convert for port
	port, err := ConvertStringToInt(resource.Port)
	if err != nil {
		log.Log.Error(err, "(create) unable to convert string to int")
		return "", err
	}
	resp, err := client.PostApiV1HealthchecksWithResponse(context.Background(), clientsdk.PostApiV1HealthchecksJSONRequestBody{
		Body:        &resource.Target,
		Description: resource.Description,
		ContentType: &resource.Protocol,
		Enabled:     true,
		HttpCode:    resource.Method,
		Interval:    interval,
		Method:      resource.Method,
		Name:        resource.Name,
		Path:        resource.Path,
		Port:        port,
		Protocol:    resource.Protocol,
		Target:      resource.Target,
		Timeout:     timeout,
	})
	if err != nil {
		log.Log.Error(err, "(create) unable to retrieve health checks")
		return "", err
	}

	if resp.HTTPResponse.StatusCode != http.StatusCreated || resp.HTTPResponse.Header.Get("Content-Type") != "application/json" {
		log.Log.Info("create", "statusCode", resp.HTTPResponse.StatusCode)
		log.Log.Info("create", "headers", resp.HTTPResponse.Header.Get("Content-Type"))
		toto := bytes.NewReader(resp.Body)
		data, err := io.ReadAll(toto)
		if err != nil {
			log.Log.Error(err, "unable to read response body")
			return "", err
		}
		log.Log.Info("create", "data", string(data))

		log.Log.Error(err, "(create) statusCode or Content-Type is not valid")
		return "", err
	}
	return "created", nil
}

func wentiApiUpdateHealthCheck(resource IngressInfo, HealthCheckID string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		log.Log.Error(err, "(update) unable to update client")
		return "", err
	}

	// Convert for interval
	interval, err := ConvertStringToInt(resource.Interval)
	if err != nil {
		log.Log.Error(err, "(update) unable to convert string to int")
		return "", err
	}

	// Convert for timeout
	timeout, err := ConvertStringToInt(resource.Timeout)
	if err != nil {
		log.Log.Error(err, "(update) unable to convert string to int")
		return "", err
	}

	// Convert for port
	port, err := ConvertStringToInt(resource.Port)
	if err != nil {
		log.Log.Error(err, "(update) unable to convert string to int")
		return "", err
	}

	resp, err := client.PutApiV1HealthchecksIdWithResponse(context.Background(), HealthCheckID, clientsdk.PutApiV1HealthchecksIdJSONRequestBody{
		Body:        &resource.Target,
		Description: resource.Description,
		ContentType: &resource.Protocol,
		Enabled:     true,
		HttpCode:    resource.Method,
		Interval:    interval,
		Method:      resource.Method,
		Name:        resource.Name,
		Path:        resource.Path,
		Port:        port,
		Protocol:    resource.Protocol,
		Target:      resource.Target,
		Timeout:     timeout,
	})
	if err != nil {
		log.Log.Error(err, "(update) unable to retrieve health checks")
		return "", err
	}

	if resp.HTTPResponse.StatusCode != http.StatusNoContent || resp.HTTPResponse.Header.Get("Content-Type") != "application/json" {
		log.Log.Error(err, "(update) statusCode or Content-Type is not valid")
		return "", err
	}

	return "updated", nil
}
