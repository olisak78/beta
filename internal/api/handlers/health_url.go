package handlers

import (
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/service"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ErrComponentHealthDisabled indicates the component's health flag is explicitly disabled in metadata.
var ErrComponentHealthDisabled = errors.New("component 'health' flag is not set to 'true'")

// BuildComponentHealthURL computes the final health URL for a component given the optional project-level
// URL template and the component/landscape context. It supports the following placeholders:
//   - {landscape_domain}  -> landscape.Domain
//   - {health_suffix}     -> metadata["health_suffix"] (string; empty if missing)
//   - {subdomain}         -> metadata["subdomain"] (string; omitted when empty)
//   - {component_name}    -> component.Name
//
// If a template is provided by projectService, it will be used. Otherwise, a legacy fallback URL is constructed:
//   - With subdomain: https://{subdomain}.{component_name}.cfapps.{landscape_domain}/health
//   - Without subdomain: https://{component_name}.cfapps.{landscape_domain}/health
//
// If the component's metadata has "health": false, ErrComponentHealthDisabled is returned.
func BuildComponentHealthURL(
	componentService service.ComponentServiceInterface,
	landscapeService service.LandscapeServiceInterface,
	projectService service.ProjectServiceInterface,
	componentID uuid.UUID,
	landscapeID uuid.UUID,
) (string, error) {
	// Resolve component and landscape
	// First fetch component (matches previous handler behavior)
	component, err := componentService.GetByID(componentID)
	if err != nil {
		return "", err
	}
	if landscapeService == nil {
		return "", apperrors.ErrLandscapeNotConfigured
	}
	landscape, err := landscapeService.GetLandscapeByID(landscapeID)
	if err != nil {
		return "", err
	}

	// Get project health URL template
	healthURLTemplate := ""
	if projectService != nil && component.ProjectID != uuid.Nil {
		var err error
		healthURLTemplate, err = projectService.GetHealthURL(component.ProjectID)
		if err != nil {
			return "", err
		}
	}

	// Extract metadata values: health flag, subdomain, health_suffix
	subdomain := ""    // default for {subdomain} is empty (omit segment when absent)
	healthSuffix := "" // used for {health_suffix}
	if len(component.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(component.Metadata, &meta); err == nil {
			// get 'health' flag from metadata to check if health is enabled:
			if healthRaw, ok := meta["health"]; ok {
				if healthBool, ok := healthRaw.(bool); ok && !healthBool {
					return "", ErrComponentHealthDisabled
				}
			}
			// subdomain from metadata (if exists)
			if sdRaw, ok := meta["subdomain"]; ok {
				if sdStr, ok := sdRaw.(string); ok && sdStr != "" {
					subdomain = sdStr
				}
			}
			// health suffix from metadata (if exists)
			if hsRaw, ok := meta["health_suffix"]; ok {
				if hsStr, ok := hsRaw.(string); ok {
					healthSuffix = hsStr
				}
			}
		}
	}

	// Determine URL template and perform placeholder substitution
	if strings.TrimSpace(healthURLTemplate) != "" {
		// Replace placeholders in provided template with optional {subdomain} handling
		t := healthURLTemplate
		t = strings.ReplaceAll(t, "{landscape_domain}", landscape.Domain)
		t = strings.ReplaceAll(t, "{health_suffix}", healthSuffix)

		if strings.TrimSpace(subdomain) == "" {
			// Remove optional subdomain segment including adjacent dots if subdomain is not provided
			t = strings.ReplaceAll(t, "{subdomain}.", "")
			t = strings.ReplaceAll(t, ".{subdomain}", "")
			t = strings.ReplaceAll(t, "{subdomain}", "")
		} else {
			t = strings.ReplaceAll(t, "{subdomain}", subdomain)
		}

		t = strings.ReplaceAll(t, "{component_name}", component.Name)
		return t, nil
	}

	// Legacy fallback: compose host and build URL
	host := component.Name
	if subdomain != "" {
		host = fmt.Sprintf("%s.%s", subdomain, component.Name)
	}
	return fmt.Sprintf("https://%s.cfapps.%s/health", host, landscape.Domain), nil
}
