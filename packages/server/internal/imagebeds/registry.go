package imagebeds

import (
	"embed"
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
)

type ProviderField struct {
	Key            string `json:"key"`
	Type           string `json:"type"`
	Label          string `json:"label"`
	LabelKey       string `json:"labelKey"`
	Placeholder    string `json:"placeholder"`
	PlaceholderKey string `json:"placeholderKey"`
	HelpText       string `json:"helpText"`
	HelpTextKey    string `json:"helpTextKey"`
	InputMode      string `json:"inputMode"`
	Required       bool   `json:"required"`
}

type RequestHeader struct {
	Key           string `json:"key"`
	ValueTemplate string `json:"valueTemplate"`
}

type QueryParam struct {
	Key           string `json:"key"`
	ValueTemplate string `json:"valueTemplate"`
	Required      bool   `json:"required"`
}

type FormField struct {
	Key           string `json:"key"`
	ValueTemplate string `json:"valueTemplate"`
	Required      bool   `json:"required"`
	OmitIfEmpty   bool   `json:"omitIfEmpty"`
}

type UploadRule struct {
	Method            string          `json:"method"`
	URLTemplate       string          `json:"urlTemplate"`
	FileField         string          `json:"fileField"`
	Headers           []RequestHeader `json:"headers"`
	Query             []QueryParam    `json:"query"`
	FormFields        []FormField     `json:"formFields"`
	SuccessJSONPath   string          `json:"successJsonPath"`
	SuccessEquals     string          `json:"successEquals"`
	ResultURLPaths    []string        `json:"resultUrlPaths"`
	ErrorMessagePaths []string        `json:"errorMessagePaths"`
}

type RuntimeRule struct {
	DefaultBaseURL string `json:"defaultBaseUrl"`
	BaseURLEnv     string `json:"baseUrlEnv"`
	APITokenEnv    string `json:"apiTokenEnv"`
}

type Provider struct {
	ProviderType string          `json:"providerType"`
	DisplayName  string          `json:"displayName"`
	Description  string          `json:"description"`
	Fields       []ProviderField `json:"fields"`
	Runtime      RuntimeRule     `json:"runtime"`
	Upload       UploadRule      `json:"upload"`
}

//go:embed providers/*.json
var providerFS embed.FS

var loadedProviders = loadProviders()

func loadProviders() []Provider {
	entries, err := providerFS.ReadDir("providers")
	if err != nil {
		panic(err)
	}

	providers := make([]Provider, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		payload, err := providerFS.ReadFile("providers/" + entry.Name())
		if err != nil {
			panic(err)
		}

		var provider Provider
		if err := json.Unmarshal(payload, &provider); err != nil {
			panic(err)
		}
		if err := validateProvider(provider); err != nil {
			panic(err)
		}
		if _, exists := seen[provider.ProviderType]; exists {
			panic("duplicate image bed provider type: " + provider.ProviderType)
		}
		seen[provider.ProviderType] = struct{}{}
		providers = append(providers, provider)
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ProviderType < providers[j].ProviderType
	})
	return providers
}

func validateProvider(provider Provider) error {
	if strings.TrimSpace(provider.ProviderType) == "" {
		return errors.New("providerType is required")
	}
	if strings.TrimSpace(provider.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	for _, field := range provider.Fields {
		if strings.TrimSpace(field.Key) == "" {
			return errors.New("field key is required")
		}
		if strings.TrimSpace(field.Type) == "" {
			return errors.New("field type is required")
		}
	}
	if strings.TrimSpace(provider.Upload.Method) == "" {
		return errors.New("upload method is required")
	}
	if strings.TrimSpace(provider.Upload.URLTemplate) == "" {
		return errors.New("upload urlTemplate is required")
	}
	if strings.TrimSpace(provider.Upload.FileField) == "" {
		return errors.New("upload fileField is required")
	}
	if len(provider.Upload.ResultURLPaths) == 0 {
		return errors.New("upload resultUrlPaths is required")
	}
	return nil
}

func ListProviders() []Provider {
	out := make([]Provider, len(loadedProviders))
	copy(out, loadedProviders)
	return out
}

func IsProviderSupported(providerType string) bool {
	_, ok := GetProviderByType(providerType)
	return ok
}

func GetProviderByType(providerType string) (Provider, bool) {
	trimmed := strings.TrimSpace(providerType)
	for _, provider := range loadedProviders {
		if provider.ProviderType == trimmed {
			return provider, true
		}
	}
	return Provider{}, false
}
