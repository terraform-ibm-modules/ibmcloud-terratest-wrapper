package testschematic

import (
	"fmt"
	"os"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
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

	// PANIC CATCH and TEAR DOWN
	// This defer will set up two things:
	// A catch of a panic and recover, to continue all tests in case of panic
	// Set up teardown to be performed after test is complete normally or if panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("=== RECOVER FROM PANIC ===")
			options.Testing.Errorf("Recovered from panic: %v", r)
		}
		testTearDown(svc, options)
	}()

	// create IAM authenticator if needed
	if svc.ApiAuthenticator == nil {
		svc.CreateAuthenticator(options.RequiredEnvironmentVars[ibmcloudApiKeyVar])
	}

	// create external API service if needed
	if options.SchematicsApiSvc != nil {
		svc.SchematicsApiSvc = options.SchematicsApiSvc
	} else {
		svcErr := svc.InitializeSchematicsService()
		if svcErr != nil {
			return fmt.Errorf("error creating schematics sdk service: %w", svcErr)
		}
	}

	// get the root path of this project
	projectPath, pathErr := common.GitRootPath(".")
	if pathErr != nil {
		return fmt.Errorf("error getting root path of git project: %w", pathErr)
	}

	// create a new tar file for the project
	options.Testing.Log("[SCHEMATICS] Creating TAR file")
	tarballName, tarballErr := CreateSchematicTar(projectPath, &options.TarIncludePatterns)
	if tarballErr != nil {
		return fmt.Errorf("error creating tar file: %w", tarballErr)
	}
	defer os.Remove(tarballName) // just to cleanup

	// create a new empty workspace, resulting in "draft" status
	options.Testing.Log("[SCHEMATICS] Creating Test Workspace")
	_, wsErr := svc.CreateTestWorkspace(options.Prefix, options.ResourceGroup, options.TemplateFolder, options.TerraformVersion, options.Tags)
	if wsErr != nil {
		return fmt.Errorf("error creating new schematic workspace: %w", wsErr)
	}

	options.Testing.Logf("[SCHEMATICS] Workspace Created: %s (%s)", svc.WorkspaceName, svc.WorkspaceID)
	// can be used in error messages to repeat workspace name
	workspaceNameString := fmt.Sprintf("[ %s (%s) ]", svc.WorkspaceName, svc.WorkspaceID)

	// upload the terraform code
	options.Testing.Log("[SCHEMATICS] Uploading TAR file")
	uploadErr := svc.UploadTarToWorkspace(tarballName)
	if uploadErr != nil {
		return fmt.Errorf("error uploading tar file to workspace: %w - %s", uploadErr, workspaceNameString)
	}

	// -------- UPLOAD TAR FILE ----------
	// find the tar upload job
	uploadJob, uploadJobErr := svc.FindLatestWorkspaceJobByName(SchematicsJobTypeUpload)
	if uploadJobErr != nil {
		return fmt.Errorf("error finding the upload tar action: %w - %s", uploadJobErr, workspaceNameString)
	}
	// wait for it to finish
	uploadJobStatus, uploadJobStatusErr := svc.WaitForFinalJobStatus(*uploadJob.ActionID)
	if uploadJobStatusErr != nil {
		return fmt.Errorf("error waiting for upload of tar to finish: %w - %s", uploadJobStatusErr, workspaceNameString)
	}
	// check if complete
	if uploadJobStatus != SchematicsJobStatusCompleted {
		return fmt.Errorf("tar upload has failed with status %s - %s", uploadJobStatus, workspaceNameString)
	}

	// ------ FINAL WORKSPACE CONFIG ------
	// update the default template with variables
	// NOTE: doing this AFTER terraform is loaded so that sensitive variables in Variablestore are already created in template,
	// to prevent things like api keys being exposed
	options.Testing.Log("[SCHEMATICS] Updating Workspace Variablestore")
	updateErr := svc.UpdateTestTemplateVars(options.TerraformVars)
	if updateErr != nil {
		return fmt.Errorf("error updating template with Variablestore: %w - %s", updateErr, workspaceNameString)
	}

	// TERRAFORM TESTING BEGINS
	// At this point our workspace is set up and we can start the Terraform testing.
	// From this point on we will do unit test assertions for actual Terraform actions.
	// The test tear-down routine also changes slightly when we reach this point.
	svc.TerraformTestStarted = true

	// ------ PLAN ------
	planResponse, planErr := svc.CreatePlanJob()
	if assert.NoErrorf(options.Testing, planErr, "error creating PLAN - %s", workspaceNameString) {
		options.Testing.Log("[SCHEMATICS] Starting PLAN job ...")
		planJobStatus, planStatusErr := svc.WaitForFinalJobStatus(*planResponse.Activityid)
		if assert.NoErrorf(options.Testing, planStatusErr, "error waiting for PLAN to finish - %s", workspaceNameString) {
			assert.Equalf(options.Testing, SchematicsJobStatusCompleted, planJobStatus, "PLAN has failed with status %s - %s", planJobStatus, workspaceNameString)
		}
	}

	// ------ APPLY ------
	if !options.Testing.Failed() {
		applyResponse, applyErr := svc.CreateApplyJob()
		if assert.NoErrorf(options.Testing, applyErr, "error creating APPLY - %s", workspaceNameString) {

			options.Testing.Log("[SCHEMATICS] Starting APPLY job ...")
			svc.TerraformResourcesCreated = true // at this point we might have resources deployed

			applyJobStatus, applyStatusErr := svc.WaitForFinalJobStatus(*applyResponse.Activityid)
			if assert.NoErrorf(options.Testing, applyStatusErr, "error waiting for APPLY to finish - %s", workspaceNameString) {
				assert.Equalf(options.Testing, SchematicsJobStatusCompleted, applyJobStatus, "APPLY has failed with status %s - %s", applyJobStatus, workspaceNameString)
			}
		}
	}

	return nil
}

// testTearDown is a helper function, typically called via golang "defer", that will clean up and remove any existing resources that were
// created for the test.
// The removal of some resources may be influenced by certain conditions or optional settings.
func testTearDown(svc *SchematicsTestService, options *TestSchematicOptions) {

	// PANIC CATCH and TEAR DOWN
	// if there is a panic during resource destroy, recover and fail test but do not continue with teardown
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("=== RECOVER FROM PANIC IN testschematic.testTearDown() ===")
			options.Testing.Error("Panic recovery during schematics teardown")
		}
	}()

	// ------ DESTROY RESOURCES ------
	// only run destroy if we had potentially created resources
	if svc.TerraformResourcesCreated {
		// Once we enter this block, turn the Created to false
		// This is to prevent this part from running again in case of panic and tear down is executed a 2nd time
		svc.TerraformResourcesCreated = false

		// Check if "DO_NOT_DESTROY_ON_FAILURE" is set
		envVal, _ := os.LookupEnv("DO_NOT_DESTROY_ON_FAILURE")
		if options.Testing.Failed() && strings.ToLower(envVal) == "true" {
			options.Testing.Log("[SCHEMATICS] Schematics APPLY failed. Debug the Test and delete resources manually.")
		} else {
			destroyResponse, destroyErr := svc.CreateDestroyJob()
			if assert.NoErrorf(options.Testing, destroyErr, "error creating DESTROY - %s", svc.WorkspaceName) {
				options.Testing.Log("[SCHEMATICS] Starting DESTROY job ...")
				destroyJobStatus, destroyStatusErr := svc.WaitForFinalJobStatus(*destroyResponse.Activityid)
				if assert.NoErrorf(options.Testing, destroyStatusErr, "error waiting for DESTROY to finish - %s", svc.WorkspaceName) {
					assert.Equalf(options.Testing, SchematicsJobStatusCompleted, destroyJobStatus, "DESTROY has failed with status %s - %s", destroyJobStatus, svc.WorkspaceName)
				}
			}
		}
	}

	// ------ DELETE WORKSPACE ------
	// only delete workspace if one of these is true:
	// * terraform hasn't been started yet
	// * no failures
	// * failed and DeleteWorkspaceOnFail is true
	if !svc.TerraformTestStarted ||
		!options.Testing.Failed() ||
		(options.Testing.Failed() && options.DeleteWorkspaceOnFail) {

		options.Testing.Log("[SCHEMATICS] Deleting Workspace")
		_, deleteWsErr := svc.DeleteWorkspace()
		if deleteWsErr != nil {
			options.Testing.Logf("[SCHEMATICS] WARNING: Schematics WORKSPACE DELETE failed! Remove manually if required. Name: %s (%s)", svc.WorkspaceName, svc.WorkspaceID)
		}
	}
}
