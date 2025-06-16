package cloudinfo

import (
	"testing"
)

func TestAddonConfigHelpers(t *testing.T) {
	// Test the Terraform specific helper
	configTerraform := NewAddonConfigTerraform("test-prefix", "test-addon", "test-flavor", map[string]interface{}{
		"prefix": "test-prefix",
	})

	if configTerraform.OfferingName != "test-addon" {
		t.Errorf("Expected OfferingName to be 'test-addon', got '%s'", configTerraform.OfferingName)
	}

	if configTerraform.OfferingFlavor != "test-flavor" {
		t.Errorf("Expected OfferingFlavor to be 'test-flavor', got '%s'", configTerraform.OfferingFlavor)
	}

	if configTerraform.OfferingInstallKind != InstallKindTerraform {
		t.Errorf("Expected OfferingInstallKind to be InstallKindTerraform, got '%s'", configTerraform.OfferingInstallKind)
	}

	if prefix, ok := configTerraform.Inputs["prefix"]; !ok || prefix != "test-prefix" {
		t.Errorf("Expected Inputs to contain 'prefix' with value 'test-prefix', got '%v'", configTerraform.Inputs)
	}

	// Test the Stack specific helper
	configStack := NewAddonConfigStack("stack-prefix", "stack-addon", "stack-flavor", map[string]interface{}{
		"region": "us-south",
	})

	if configStack.OfferingName != "stack-addon" {
		t.Errorf("Expected OfferingName to be 'stack-addon', got '%s'", configStack.OfferingName)
	}

	if configStack.OfferingInstallKind != InstallKindStack {
		t.Errorf("Expected OfferingInstallKind to be InstallKindStack, got '%s'", configStack.OfferingInstallKind)
	}

	if region, ok := configStack.Inputs["region"]; !ok || region != "us-south" {
		t.Errorf("Expected Inputs to contain 'region' with value 'us-south', got '%v'", configStack.Inputs)
	}
}
