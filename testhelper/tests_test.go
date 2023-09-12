package testhelper

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sample1 = "sample/terraform/sample1"
var sample1ExpectedOutputs = []string{"world"}
var sample2 = "sample/terraform/sample2"
var sample2ExpectedOutputs = []string{}
var sample3 = "sample/terraform/sample3"
var sample3ExpectedOutputs = []string{"world"}

var terraformVars = map[string]interface{}{
	"hello": "hello from the tests!"}

func TestRunTest(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:       t,
		TerraformDir:  sample1,
		Prefix:        "testRun",
		ResourceGroup: "test-rg",
		Region:        "us-south",
		TerraformVars: terraformVars,
	})
	output, err := options.RunTest()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
	_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample1ExpectedOutputs...)
	assert.Nil(t, outErr, outErr)

}

func TestRunTestRelativeModule(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:       t,
		TerraformDir:  sample2,
		Prefix:        "testRunRel",
		ResourceGroup: "test-rg",
		Region:        "us-south",
		TerraformVars: terraformVars,
	})
	output, err := options.RunTest()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
	_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample2ExpectedOutputs...)
	assert.Nil(t, outErr, outErr)

}

func TestRunTestImplicitDestroy(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:          t,
		TerraformDir:     sample1,
		Prefix:           "testRunImp",
		ResourceGroup:    "test-rg",
		Region:           "us-south",
		TerraformVars:    terraformVars,
		ImplicitDestroy:  []string{"null_resource.remove"},
		ImplicitRequired: true,
	})
	output, err := options.RunTest()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
	_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample1ExpectedOutputs...)
	assert.Nil(t, outErr, outErr)

}

func TestRunTestImplicitDestroyRelativeModule(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:          t,
		TerraformDir:     sample2,
		Prefix:           "testRunImpRel",
		ResourceGroup:    "test-rg",
		Region:           "us-south",
		TerraformVars:    terraformVars,
		ImplicitDestroy:  []string{"module.sample1.null_resource.remove"},
		ImplicitRequired: true,
	})
	output, err := options.RunTest()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
	_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample2ExpectedOutputs...)
	assert.Nil(t, outErr, outErr)

}

func TestRunTestResultStruct(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:       t,
		TerraformDir:  sample1,
		Prefix:        "testPlan",
		ResourceGroup: "test-rg",
		Region:        "us-south",
		TerraformVars: terraformVars,
	})
	output, err := options.RunTestPlan()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	// Check if options.LastTestTerraformOutputs is an empty map
	isEmpty := reflect.DeepEqual(options.LastTestTerraformOutputs, map[string]interface{}{})

	assert.True(t, isEmpty, "Expected no Terraform outputs")

}

func TestRunTestResultStructRelativeModule(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:       t,
		TerraformDir:  sample2,
		Prefix:        "testPlanRel",
		ResourceGroup: "test-rg",
		Region:        "us-south",
		TerraformVars: terraformVars,
	})
	output, err := options.RunTestPlan()
	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	// Check if options.LastTestTerraformOutputs is an empty map
	isEmpty := reflect.DeepEqual(options.LastTestTerraformOutputs, map[string]interface{}{})

	assert.True(t, isEmpty, "Expected no Terraform outputs")
}

func TestRunUpgradeTestInPlace(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:       t,
		TerraformDir:  sample1,
		Prefix:        "testRunUpgradeInplace",
		ResourceGroup: "test-rg",
		Region:        "us-south",
		TerraformVars: terraformVars,
	})
	output, _ := options.RunTestUpgrade()

	if !options.UpgradeTestSkipped {
		assert.NotNil(t, output, "Expected some output")
		assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
		_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample1ExpectedOutputs...)
		assert.Nil(t, outErr, outErr)
	}
}

func TestRunUpgradeTestInPlaceRelativeModule(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:       t,
		TerraformDir:  sample2,
		Prefix:        "testRunUpgradeInplaceRel",
		ResourceGroup: "test-rg",
		Region:        "us-south",
		TerraformVars: terraformVars,
	})
	output, _ := options.RunTestUpgrade()
	if !options.UpgradeTestSkipped {
		assert.NotNil(t, output, "Expected some output")
		assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
		_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample2ExpectedOutputs...)
		assert.Nil(t, outErr, outErr)
	}
}

func TestRunTestConsistency(t *testing.T) {
	t.Parallel()
	os.Setenv("TF_VAR_ibmcloud_api_key", "12345")
	options := TestOptionsDefaultWithVars(&TestOptions{
		Testing:        t,
		TerraformDir:   sample3,
		Prefix:         "testRunConsistency",
		ResourceGroup:  "test-rg",
		Region:         "us-south",
		TerraformVars:  terraformVars,
		IgnoreDestroys: Exemptions{List: []string{"null_resource.sample"}},
	})
	output, err := options.RunTestConsistency()

	assert.Nil(t, err, "This should not have errored")
	assert.NotNil(t, output, "Expected some output")
	assert.NotNil(t, options.LastTestTerraformOutputs, "Expected some Terraform outputs")
	_, outErr := ValidateTerraformOutputs(options.LastTestTerraformOutputs, sample3ExpectedOutputs...)
	assert.Nil(t, outErr, outErr)
}
