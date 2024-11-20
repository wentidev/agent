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

func FindHealthCheck(resource IngressInfo) (bool, string) {
	client, err := CreateClient()
	if err != nil {
		log.Log.Error(err, "(find) unable to create client")
		return false, ""
	}
	params := &clientsdk.GetApiV1HealthchecksParams{
		Target: &resource.Target,
		Method: &resource.Method,
		Path:   &resource.Path,
	}

	resp, err := client.GetApiV1HealthchecksWithResponse(context.Background(), params)
	if err != nil {
		log.Log.Error(err, "(find) unable to retrieve health checks")
		return false, ""
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		log.Log.Error(err, "(find) statusCode or Content-Type is not valid")
		return false, ""
	}

	if resp.JSON200 == nil {
		log.Log.Error(err, "(find) JSON200 is nil")
		return false, ""
	}

	log.Log.Info("(find) health check response", "body", resp.JSON200.HttpChecks)
	log.Log.Info("(find) health check response", "count", resp.JSON200.Count)

	if *resp.JSON200.Count != 0 {
		checks := *resp.JSON200.HttpChecks
		log.Log.Info("(find) health check found", "Name", *checks[0].Name)
		return true, *checks[0].Id
	}

	return false, ""
}

func DeleteHealthCheck(resource IngressInfo) (string, error) {
	findBool, healthCheckID := FindHealthCheck(resource)
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
	findBool, healthCheckID := FindHealthCheck(resource)

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

	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		log.Log.Error(err, "(delete) statusCode or Content-Type is not valid")
		body := bytes.NewReader(resp.Body)
		data, err := io.ReadAll(body)
		if err != nil {
			log.Log.Error(err, "unable to read response body")
			return err
		}
		log.Log.Info("delete", "data", string(data))
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
		Description: resource.Description,
		Enabled:     true,
		HttpCode:    resource.HTTPCode,
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

	if resp.HTTPResponse.StatusCode != http.StatusCreated {
		log.Log.Info("create", "statusCode", resp.HTTPResponse.StatusCode)
		log.Log.Info("create", "headers", resp.HTTPResponse.Header.Get("Content-Type"))
		body := bytes.NewReader(resp.Body)
		data, err := io.ReadAll(body)
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
		Description: resource.Description,
		Enabled:     true,
		HttpCode:    resource.HTTPCode,
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

	if resp.HTTPResponse.StatusCode != http.StatusNoContent {
		body := bytes.NewReader(resp.Body)
		data, err := io.ReadAll(body)
		if err != nil {
			log.Log.Error(err, "unable to read response body")
			return "", err
		}
		log.Log.Info("update", "data", string(data))
		log.Log.Info("update", "statusCode", resp.HTTPResponse.StatusCode)
		log.Log.Error(err, "(update) statusCode or Content-Type is not valid")
		return "", err
	}

	return "updated", nil
}
