package testschematic

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

// RunSchematicTest will use the supplied options to run an end-to-end Terraform test of a project in an
// IBM Cloud Schematics Workspace.
// This test will include the following steps:
// 1. create test workspace
// 2. create and upload tar file of terraform project to workspace
// 3. configure supplied test variables in workspace
// 4. run PLAN/APPLY/DESTROY steps on workspace to provision and destroy resources
// 5. delete the test workspace
func (options *TestSchematicOptions) RunSchematicTest() error {

	// WORKSPACE SETUP
	// In this section of the test we are setting up the workspace.
	// Any errors in this section will be considerd "unexpected" and returned to the calling unit test
	// to short-circuit and quit the test.
	// The official start of the unit test, with assertions, will begin AFTER workspace is properly created.

	// create new schematic service with authenticator, set pointer of service in options for use later
	var svc *SchematicsTestService
	if options.schematicsTestSvc == nil {
		svc = &SchematicsTestService{}
	} else {
		svc = options.schematicsTestSvc
	}
	svc.TestOptions = options
	svc.TerraformTestStarted = false
	svc.TerraformResourcesCreated = false

	svc.CreateAuthenticator(options.RequiredEnvironmentVars[ibmcloudApiKeyVar])
	if options.SchematicsApiSvc != nil {
		svc.SchematicsApiSvc = options.SchematicsApiSvc
	} else {
		svcErr := svc.InitializeSchematicsService()
		if svcErr != nil {
			return fmt.Errorf("error creating schematics sdk service: %w", svcErr)
		}
	}

	// get the root path of this project
	projectPath, pathErr := testhelper.GitRootPath(".")
	if pathErr != nil {
		return fmt.Errorf("error getting root path of git project: %w", pathErr)
	}

	// create a new tar file for the project
	log.Println("[SCHEMATICS] Creating TAR file")
	tarballName, tarballErr := CreateSchematicTar(projectPath, &options.TarIncludePatterns)
	if tarballErr != nil {
		return fmt.Errorf("error creating tar file: %w", tarballErr)
	}
	defer os.Remove(tarballName) // just to cleanup

	// create a new empty workspace, resulting in "draft" status
	log.Println("[SCHEMATICS] Creating Test Workspace")
	_, wsErr := svc.CreateTestWorkspace(options.Prefix, options.ResourceGroup, ".", options.TerraformVersion, options.Tags)
	if wsErr != nil {
		return fmt.Errorf("error creating new schematic workspace: %w", wsErr)
	}

	// since workspace is now created, always call the teardown to remove
	defer testTearDown(svc, options)

	// upload the terraform code
	log.Println("[SCHEMATICS] Uploading TAR file")
	uploadErr := svc.UploadTarToWorkspace(tarballName)
	if uploadErr != nil {
		return fmt.Errorf("error uploading tar file to workspace: %w", uploadErr)
	}

	// -------- UPLOAD TAR FILE ----------
	// find the tar upload job
	uploadJob, uploadJobErr := svc.FindLatestWorkspaceJobByName(SchematicsJobTypeUpload)
	if uploadJobErr != nil {
		return fmt.Errorf("error finding the upload tar action: %w", uploadJobErr)
	}
	// wait for it to finish
	uploadJobStatus, uploadJobStatusErr := svc.WaitForFinalJobStatus(*uploadJob.ActionID)
	if uploadJobStatusErr != nil {
		return fmt.Errorf("error waiting for upload of tar to finish: %w", uploadJobStatusErr)
	}
	// check if complete
	if uploadJobStatus != SchematicsJobStatusCompleted {
		return fmt.Errorf("tar upload has failed with status %s", uploadJobStatus)
	}

	// ------ FINAL WORKSPACE CONFIG ------
	// update the default template with variables
	// NOTE: doing this AFTER terraform is loaded so that sensitive variables in Variablestore are already created in template,
	// to prevent things like api keys being exposed
	log.Println("[SCHEMATICS] Updating Workspace Variablestore")
	updateErr := svc.UpdateTestTemplateVars(options.TerraformVars)
	if updateErr != nil {
		return fmt.Errorf("error updating template with Variablestore: %w", updateErr)
	}

	// TERRAFORM TESTING BEGINS
	// At this point our workspace is set up and we can start the Terraform testing.
	// From this point on we will do unit test assertions for actual Terraform actions.
	// The test tear-down routine also changes slightly when we reach this point.
	svc.TerraformTestStarted = true

	// ------ PLAN ------
	planResponse, planErr := svc.CreatePlanJob()
	if assert.NoError(options.Testing, planErr, "error creating PLAN") {
		log.Println("[SCHEMATICS] Starting PLAN job ...")
		planJobStatus, planStatusErr := svc.WaitForFinalJobStatus(*planResponse.Activityid)
		if assert.NoError(options.Testing, planStatusErr, "error waiting for PLAN to finish") {
			assert.Equalf(options.Testing, SchematicsJobStatusCompleted, planJobStatus, "PLAN has failed with status %s", planJobStatus)
		}
	}

	// ------ APPLY ------
	if !options.Testing.Failed() {
		applyResponse, applyErr := svc.CreateApplyJob()
		if assert.NoError(options.Testing, applyErr, "error creating APPLY") {

			log.Println("[SCHEMATICS] Starting APPLY job ...")
			svc.TerraformResourcesCreated = true // at this point we might have resources deployed

			applyJobStatus, applyStatusErr := svc.WaitForFinalJobStatus(*applyResponse.Activityid)
			if assert.NoError(options.Testing, applyStatusErr, "error waiting for APPLY to finish") {
				assert.Equalf(options.Testing, SchematicsJobStatusCompleted, applyJobStatus, "APPLY has failed with status %s", applyJobStatus)
			}
		}
	}

	// ------ DESTROY ------
	// only run destroy if we had potentially created resources
	if svc.TerraformResourcesCreated {
		// Check if "DO_NOT_DESTROY_ON_FAILURE" is set
		envVal, _ := os.LookupEnv("DO_NOT_DESTROY_ON_FAILURE")
		if options.Testing.Failed() && strings.ToLower(envVal) == "true" {
			log.Println("[SCHEMATICS] Schematics APPLY failed. Debug the Test and delete resources manually.")
		} else {
			destroyResponse, destroyErr := svc.CreateDestroyJob()
			if assert.NoError(options.Testing, destroyErr, "error creating DESTROY") {
				log.Println("[SCHEMATICS] Starting DESTROY job ...")
				destroyJobStatus, destroyStatusErr := svc.WaitForFinalJobStatus(*destroyResponse.Activityid)
				if assert.NoError(options.Testing, destroyStatusErr, "error waiting for DESTROY to finish") {
					assert.Equalf(options.Testing, SchematicsJobStatusCompleted, destroyJobStatus, "DESTROY has failed with status %s", destroyJobStatus)
				}
			}
		}
	}

	return nil
}

// testTearDown is a helper function, typically called via golang "defer", that will clean up and remove any existing resources that were
// created for the test.
// The removal of some resources may be influenced by certain conditions or optional settings.
func testTearDown(svc *SchematicsTestService, options *TestSchematicOptions) {
	// ------ DELETE WORKSPACE ------
	// only delete workspace if one of these is true:
	// * terraform hasn't been started yet
	// * no failures
	// * failed and DeleteWorkspaceOnFail is true
	if !svc.TerraformTestStarted ||
		!options.Testing.Failed() ||
		(options.Testing.Failed() && options.DeleteWorkspaceOnFail) {

		log.Println("[SCHEMATICS] Deleting Workspace")
		_, deleteWsErr := svc.DeleteWorkspace()
		if deleteWsErr != nil {
			log.Println("[SCHEMATICS] WARNING: Schematics WORKSPACE DELETE failed! Remove manually if required.")
		}
	}
}
