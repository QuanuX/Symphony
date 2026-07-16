package model

// Status is the safe, metadata-only SSIAG status projection.
type Status struct {
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Ready         bool   `json:"ready"`
	Mode          string `json:"mode"`
	TOPSID        string `json:"tops_id"`
	TOPSName      string `json:"tops_name"`
	Transport     string `json:"transport"`
	ProviderCount int    `json:"provider_count"`
}

// ProviderDescriptor describes capabilities without exposing provider credentials.
type ProviderDescriptor struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
	Exportable   bool     `json:"exportable"`
	Interactive  bool     `json:"interactive"`
}

// ProvidersResponse is the versioned provider discovery response.
type ProvidersResponse struct {
	Schema    string               `json:"schema"`
	Providers []ProviderDescriptor `json:"providers"`
}

// ErrorResponse contains only safe error metadata.
type ErrorResponse struct {
	Schema  string `json:"schema"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
