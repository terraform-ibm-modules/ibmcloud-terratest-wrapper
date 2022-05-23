package testhelper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sample1 = "sample/terraform/sample1"
var sample2 = "sample/terraform/sample2"
var sample3 = "sample/terraform/sample3"

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
}
