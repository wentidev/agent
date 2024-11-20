package utils

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

func NewIngressInfo() IngressInfo {
	return IngressInfo{
		Name:        "default-name",
		Description: "default-description",
		Target:      "localhost",
		Port:        "8080",
		Protocol:    "http",
		Path:        "/",
		Method:      "GET",
		Timeout:     "30s",
		Interval:    "60s",
		HTTPCode:    "200",
		Enabled:     true,
	}
}
