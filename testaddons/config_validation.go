package testaddons

import (
	"fmt"
	"time"

	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

// getConfigDetailsWithRetry retrieves configuration details with retry logic to handle API timing issues
func (options *TestAddonOptions) getConfigDetailsWithRetry(configID string, maxRetries int, retryDelay time.Duration) (*projectv1.ProjectConfig, error) {
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		currentConfigDetails, _, err := options.CloudInfoService.GetConfig(&cloudinfo.ConfigDetails{
			ProjectID: options.currentProjectConfig.ProjectID,
			ConfigID:  configID,
		})

		if err != nil {
			lastError = err
			if attempt < maxRetries {
				options.Logger.ShortInfo(fmt.Sprintf("Attempt %d/%d: Error getting configuration details, retrying in %v: %v", attempt, maxRetries, retryDelay, err))
				time.Sleep(retryDelay)
				continue
			}
			break
		}

		// Success
		if attempt > 1 {
			options.Logger.ShortInfo(fmt.Sprintf("Successfully retrieved configuration on attempt %d/%d", attempt, maxRetries))
		}
		return currentConfigDetails, nil
	}

	return nil, fmt.Errorf("failed to get configuration after %d attempts: %v", maxRetries, lastError)
}

// validateRequiredInputs checks if all required inputs are present in the configuration
func (options *TestAddonOptions) validateRequiredInputs(configDetails *projectv1.ProjectConfig, targetAddon cloudinfo.AddonConfig) (bool, []string) {
	var missingInputs []string

	for _, input := range targetAddon.OfferingInputs {
		if !input.Required || input.Key == "ibmcloud_api_key" {
			continue
		}

		value, exists := configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Inputs[input.Key]
		if !exists || fmt.Sprintf("%v", value) == "" {
			if input.DefaultValue == nil || fmt.Sprintf("%v", input.DefaultValue) == "" || fmt.Sprintf("%v", input.DefaultValue) == "__NOT_SET__" {
				configIdentifier := fmt.Sprintf("%s (missing: %s)", *configDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse).Name, input.Key)
				missingInputs = append(missingInputs, configIdentifier)
			}
		}
	}

	return len(missingInputs) == 0, missingInputs
}

// validateInputsWithRetry validates required inputs for a configuration with retry logic
// This handles the case where the backend database hasn't been updated yet after configuration changes
func (options *TestAddonOptions) validateInputsWithRetry(configID string, targetAddon cloudinfo.AddonConfig, maxRetries int, retryDelay time.Duration) (bool, []string) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Get configuration details with retry logic
		currentConfigDetails, err := options.getConfigDetailsWithRetry(configID, 1, 0) // Single attempt per validation retry
		if err != nil {
			if attempt < maxRetries {
				options.Logger.ShortInfo(fmt.Sprintf("Attempt %d/%d: Could not retrieve config, retrying in %v: %v", attempt, maxRetries, retryDelay, err))
				time.Sleep(retryDelay)
				continue
			}
			options.Logger.ShortError(fmt.Sprintf("Failed to get configuration after %d attempts: %v", maxRetries, err))
			return false, []string{fmt.Sprintf("Failed to get configuration: %v", err)}
		}

		// Validate inputs
		allInputsPresent, missingInputs := options.validateRequiredInputs(currentConfigDetails, targetAddon)

		if allInputsPresent {
			if attempt > 1 {
				options.Logger.ShortInfo(fmt.Sprintf("Input validation succeeded on attempt %d/%d after retrying", attempt, maxRetries))
			}
			return true, nil
		}

		// If this isn't the last attempt, wait and retry
		if attempt < maxRetries {
			options.Logger.ShortInfo(fmt.Sprintf("Attempt %d/%d: Some required inputs appear missing, retrying in %v (this may be due to database update timing)", attempt, maxRetries, retryDelay))

			// Show which inputs are missing on this attempt for debugging
			if len(missingInputs) > 0 {
				options.Logger.ShortInfo(fmt.Sprintf("Missing inputs on attempt %d:", attempt))
				for _, missing := range missingInputs {
					options.Logger.ShortInfo(fmt.Sprintf("  %s", missing))
				}
			}

			time.Sleep(retryDelay)
		}
	}

	// All attempts failed - get final configuration state for detailed debugging
	finalConfigDetails, finalErr := options.getConfigDetailsWithRetry(configID, 1, 0)
	if finalErr != nil {
		options.Logger.ShortError(fmt.Sprintf("Input validation failed after %d attempts due to configuration retrieval error: %v", maxRetries, finalErr))
		return false, []string{fmt.Sprintf("Failed to get configuration: %v", finalErr)}
	}

	_, missingInputs := options.validateRequiredInputs(finalConfigDetails, targetAddon)

	options.Logger.ShortWarn(fmt.Sprintf("Input validation failed after %d attempts - inputs still appear missing (test will continue and attempt deployment in case inputs resolve during deployment):", maxRetries))
	for _, missing := range missingInputs {
		options.Logger.ShortWarn(fmt.Sprintf("  %s", missing))
	}

	// Show detailed retry debug information when all attempts fail
	options.Logger.ShortWarn("=== RETRY VALIDATION DEBUG INFO ===")
	options.Logger.ShortWarn(fmt.Sprintf("Configuration ID: %s", configID))
	options.Logger.ShortWarn(fmt.Sprintf("Retry attempts: %d", maxRetries))
	options.Logger.ShortWarn(fmt.Sprintf("Retry delay: %v", retryDelay))

	options.Logger.ShortWarn("Final configuration state:")
	if finalConfigDetails.State != nil {
		options.Logger.ShortWarn(fmt.Sprintf("  State: %s", *finalConfigDetails.State))
	}
	if finalConfigDetails.StateCode != nil {
		options.Logger.ShortWarn(fmt.Sprintf("  StateCode: %s", string(*finalConfigDetails.StateCode)))
	}

	options.Logger.ShortWarn("All inputs in final configuration:")
	if finalConfigDetails.Definition != nil {
		if resp, ok := finalConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
			for key, value := range resp.Inputs {
				// Use structured sensitive data detection instead of fragile string matching
				maskedValue := GetSafeMaskedValue(key, value)
				if IsSensitiveField(key) {
					options.Logger.ShortWarn(fmt.Sprintf("    %s: %s", key, maskedValue))
				} else {
					options.Logger.ShortWarn(fmt.Sprintf("    %s: %v (type: %T)", key, value, value))
				}
			}
		} else {
			options.Logger.ShortWarn("    No inputs found in configuration definition")
		}
	}

	options.Logger.ShortWarn("Missing required inputs summary:")
	for _, input := range targetAddon.OfferingInputs {
		if input.Required && input.Key != "ibmcloud_api_key" {
			found := false
			if finalConfigDetails.Definition != nil {
				if resp, ok := finalConfigDetails.Definition.(*projectv1.ProjectConfigDefinitionResponse); ok && resp.Inputs != nil {
					if _, exists := resp.Inputs[input.Key]; exists {
						found = true
					}
				}
			}
			status := "✓ PROVIDED"
			if !found {
				status = "✗ MISSING"
			}
			options.Logger.ShortWarn(fmt.Sprintf("    %s: %s", input.Key, status))
		}
	}

	options.Logger.ShortWarn("Required inputs that were checked:")
	for _, input := range targetAddon.OfferingInputs {
		if input.Required && input.Key != "ibmcloud_api_key" {
			defaultInfo := "no default"
			if input.DefaultValue != nil {
				defaultInfo = fmt.Sprintf("default: %v", input.DefaultValue)
			}
			options.Logger.ShortWarn(fmt.Sprintf("    %s (%s)", input.Key, defaultInfo))
		}
	}
	options.Logger.ShortCustom("=== END RETRY DEBUG INFO ===", common.Colors.Cyan)

	return false, missingInputs
}
