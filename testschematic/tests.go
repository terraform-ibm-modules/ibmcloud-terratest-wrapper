package testschematic

import (
	"fmt"
	"os"
	"strings"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
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
	svc, setupErr := testSetup(options)
	if setupErr != nil {
		return setupErr
	}

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
	_, wsErr := svc.CreateTestWorkspace(options.Prefix, options.ResourceGroup, svc.WorkspaceLocation, options.TemplateFolder, options.TerraformVersion, options.Tags)
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
	planSuccess := false // will only flip to true if job completes
	planResponse, planErr := svc.CreatePlanJob()
	if assert.NoErrorf(options.Testing, planErr, "error creating PLAN - %s", workspaceNameString) {
		options.Testing.Log("[SCHEMATICS] Starting PLAN job ...")
		planJobStatus, planStatusErr := svc.WaitForFinalJobStatus(*planResponse.Activityid)
		if assert.NoErrorf(options.Testing, planStatusErr, "error waiting for PLAN to finish - %s", workspaceNameString) {
			planSuccess = assert.Equalf(options.Testing, SchematicsJobStatusCompleted, planJobStatus, "PLAN has failed with status %s - %s", planJobStatus, workspaceNameString)
		}

		if !planSuccess || options.PrintAllSchematicsLogs {
			printPlanLogErr := svc.printWorkspaceJobLogToTestLog(*planResponse.Activityid, "PLAN")
			if printPlanLogErr != nil {
				options.Testing.Logf("Error printing PLAN logs:%s", printPlanLogErr)
			}
		}
	}

	// ------ APPLY ------
	applySuccess := false // will only flip to true if job completes
	if !options.Testing.Failed() {
		applyResponse, applyErr := svc.CreateApplyJob()
		if assert.NoErrorf(options.Testing, applyErr, "error creating APPLY - %s", workspaceNameString) {

			options.Testing.Log("[SCHEMATICS] Starting APPLY job ...")
			svc.TerraformResourcesCreated = true // at this point we might have resources deployed

			applyJobStatus, applyStatusErr := svc.WaitForFinalJobStatus(*applyResponse.Activityid)
			if assert.NoErrorf(options.Testing, applyStatusErr, "error waiting for APPLY to finish - %s", workspaceNameString) {
				applySuccess = assert.Equalf(options.Testing, SchematicsJobStatusCompleted, applyJobStatus, "APPLY has failed with status %s - %s", applyJobStatus, workspaceNameString)
			}

			if !applySuccess || options.PrintAllSchematicsLogs {
				printApplyLogErr := svc.printWorkspaceJobLogToTestLog(*applyResponse.Activityid, "APPLY")
				if printApplyLogErr != nil {
					options.Testing.Logf("Error printing APPLY logs:%s", printApplyLogErr)
				}
			}
		}
	}

	// ------ CONSISTENCY PLAN ------
	consistencyPlanSuccess := false // will only flip to true if job completes
	if !options.Testing.Failed() {
		consistencyPlanResponse, consistencyPlanErr := svc.CreatePlanJob()
		if assert.NoErrorf(options.Testing, consistencyPlanErr, "error creating PLAN - %s", workspaceNameString) {
			options.Testing.Log("[SCHEMATICS] Starting CONSISTENCY PLAN job ...")
			consistencyPlanJobStatus, consistencyPlanStatusErr := svc.WaitForFinalJobStatus(*consistencyPlanResponse.Activityid)
			if assert.NoErrorf(options.Testing, consistencyPlanStatusErr, "error waiting for CONSISTENCY PLAN to finish - %s", workspaceNameString) {
				if assert.Equalf(options.Testing, SchematicsJobStatusCompleted, consistencyPlanJobStatus, "CONSISTENCY PLAN has failed with status %s - %s", consistencyPlanJobStatus, workspaceNameString) {
					// if the consistency plan was successful, get the plan json and check consistency
					consistencyPlanJson, consistencyPlanJsonErr := svc.TestOptions.CloudInfoService.GetSchematicsJobPlanJson(*consistencyPlanResponse.Activityid, svc.WorkspaceLocation)
					if assert.NoErrorf(options.Testing, consistencyPlanJsonErr, "error retrieving CONSISTENCY PLAN JSON - %w - %s", consistencyPlanJsonErr, workspaceNameString) {
						// convert the json string into a terratest plan struct
						planStruct, planStructErr := terraform.ParsePlanJSON(consistencyPlanJson)
						if assert.NoErrorf(options.Testing, planStructErr, "error converting plan string into struct: %w -%s", planStructErr, workspaceNameString) {
							// base the success not on job complete, but on if consistency test finds any problems
							// CheckConsistency returns TRUE if it finds issues, so we will negate that for success
							foundConsistencyIssues := testhelper.CheckConsistency(planStruct, options)
							consistencyPlanSuccess = !foundConsistencyIssues
						}
					}
				}
			}

			if !consistencyPlanSuccess || options.PrintAllSchematicsLogs {
				printConsistencyLogErr := svc.printWorkspaceJobLogToTestLog(*consistencyPlanResponse.Activityid, "CONSISTENCY PLAN")
				if printConsistencyLogErr != nil {
					options.Testing.Logf("Error printing PLAN logs:%s", printConsistencyLogErr)
				}
			}
		}
	}

	return nil
}

// testSetup is a helper function that will initialize and setup the SchematicsTestService in preparation for a test
// Any errors in this section will be considerd "unexpected" and returned to the calling unit test
// to short-circuit and quit the test.
func testSetup(options *TestSchematicOptions) (*SchematicsTestService, error) {
	// create new schematic service with authenticator, set pointer of service in options for use later
	var svc *SchematicsTestService
	if options.schematicsTestSvc == nil {
		svc = &SchematicsTestService{}
	} else {
		svc = options.schematicsTestSvc
	}

	svc.TestOptions = options

	// create new CloudInfoService if not supplied
	if options.CloudInfoService == nil {
		cloudInfoSvc, cloudInfoErr := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
		if cloudInfoErr != nil {
			return nil, cloudInfoErr
		}
		svc.CloudInfoService = cloudInfoSvc
		options.CloudInfoService = cloudInfoSvc
	} else {
		svc.CloudInfoService = options.CloudInfoService
	}

	// pick random region for workspace if it was not supplied
	// if no region specified, choose a random one
	if len(options.WorkspaceLocation) > 0 {
		svc.WorkspaceLocation = options.WorkspaceLocation
	} else {
		svc.WorkspaceLocation = cloudinfo.GetRandomSchematicsLocation()
		svc.TestOptions.Testing.Logf("[SCHEMATICS] Random Workspace region chosen: %s", svc.WorkspaceLocation)
	}

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
			return nil, fmt.Errorf("error creating schematics sdk service: %w", svcErr)
		}
	}

	return svc, nil
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

	// only perform if skip is not set
	if !options.SkipTestTearDown {
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
				destroySuccess := false // will only flip to true if job completes
				destroyResponse, destroyErr := svc.CreateDestroyJob()
				if assert.NoErrorf(options.Testing, destroyErr, "error creating DESTROY - %s", svc.WorkspaceName) {
					options.Testing.Log("[SCHEMATICS] Starting DESTROY job ...")
					destroyJobStatus, destroyStatusErr := svc.WaitForFinalJobStatus(*destroyResponse.Activityid)
					if assert.NoErrorf(options.Testing, destroyStatusErr, "error waiting for DESTROY to finish - %s", svc.WorkspaceName) {
						destroySuccess = assert.Equalf(options.Testing, SchematicsJobStatusCompleted, destroyJobStatus, "DESTROY has failed with status %s - %s", destroyJobStatus, svc.WorkspaceName)
					}

					if !destroySuccess || options.PrintAllSchematicsLogs {
						printDestroyLogErr := svc.printWorkspaceJobLogToTestLog(*destroyResponse.Activityid, "DESTROY")
						if printDestroyLogErr != nil {
							options.Testing.Logf("Error printing DESTROY logs:%s", printDestroyLogErr)
						}
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
}

// SPECIAL NOTE: We do not want to fail the test if there is any issue/error retrieving or printing a log.
// In this function we will be capturing most errors and to simply short-circuit and return
// the error to the caller, to avoid any panic or test failure.
func (svc *SchematicsTestService) printWorkspaceJobLogToTestLog(jobID string, jobType string) error {

	// if for some reason cloudInfo has not been initialized, return immediately
	if svc.CloudInfoService == nil {
		return fmt.Errorf("could not get workspace logs, CloudInfoService was not initialized which is unexpected - JobID %s", jobID)
	}

	// retrieve job log
	jobLog, jobLogErr := svc.CloudInfoService.GetSchematicsJobLogsText(jobID, svc.WorkspaceLocation)
	if jobLogErr != nil {
		return jobLogErr
	}
	if len(jobLog) == 0 {
		return fmt.Errorf("workspace job log was empty which is unexpected - JobID %s", jobID)
	}

	// create some headers and footers
	logHeader := fmt.Sprintf("=============== BEGIN %s JOB LOG (%s) ===============", strings.ToUpper(jobType), svc.WorkspaceID)
	logFooter := fmt.Sprintf("=============== END %s JOB LOG (%s) ===============", strings.ToUpper(jobType), svc.WorkspaceID)
	finalLog := fmt.Sprintf("%s\n%s\n%s", logHeader, jobLog, logFooter)

	// print out log text
	svc.TestOptions.Testing.Log(finalLog)

	return nil
}
