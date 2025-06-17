// This test ensures that our critical dependencies compile correctly
// and helps catch breaking changes in external libraries
//
// STRATEGY: Focus on IBM Cloud SDK packages which are:
// 1. Most likely to have breaking changes (frequently updated)
// 2. Critical to our functionality (service clients we instantiate)
// 3. Have proven problematic in the past (containerv2)
//
// NOT INCLUDED (low risk): go-sdk-core, terratest, standard library
// ADD HERE: If you encounter compilation issues with other dependencies
package cloudinfo

import (
	"testing"

	// IBM Cloud SDK dependencies - highest risk for breaking changes
	"github.com/IBM-Cloud/bluemix-go/api/container/containerv2"
	"github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/platform-services-go-sdk/catalogmanagementv1"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/project-go-sdk/projectv1"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

// TestCriticalDependenciesCompilation verifies that our critical dependencies compile
// This test will fail if external libraries introduce breaking changes
// Focus on IBM SDK packages which are most likely to have breaking changes
func TestCriticalDependenciesCompilation(t *testing.T) {
	t.Log("Testing compilation of critical IBM Cloud SDK dependencies...")

	// Test container service interfaces (already proven problematic)
	var _ containerv2.Clusters
	var _ containerv2.Alb
	t.Log("✓ containerv2 interfaces available")

	// Test key service clients that we instantiate
	var _ *catalogmanagementv1.CatalogManagementV1
	var _ *resourcecontrollerv2.ResourceControllerV2
	var _ *resourcemanagerv2.ResourceManagerV2
	var _ *vpcv1.VpcV1
	var _ *contextbasedrestrictionsv1.ContextBasedRestrictionsV1
	var _ *iamidentityv1.IamIdentityV1
	var _ *clouddatabasesv5.CloudDatabasesV5
	var _ *projectv1.ProjectV1
	var _ *schematicsv1.SchematicsV1
	t.Log("✓ IBM service clients available")

	// Test key model types we use
	var _ *models.CloudConnection // power-go-client
	t.Log("✓ Power client models available")

	// Test that we can create service options (common breaking change)
	_ = &catalogmanagementv1.CatalogManagementV1Options{}
	_ = &resourcecontrollerv2.ResourceControllerV2Options{}
	_ = &vpcv1.VpcV1Options{}
	t.Log("✓ Service options structs available")

	t.Log("All critical dependencies compile successfully")
}
