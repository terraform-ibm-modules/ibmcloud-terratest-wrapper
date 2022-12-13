package testschematic

import (
	"fmt"
	"os"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

func (options *TestSchematicOptions) RunSchematicTest() error {

	// create new schematic service with authenticator, set pointer of service in options for use later
	svc := &SchematicsTestService{
		TestOptions: options,
	}
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
	tarballName, tarballErr := CreateSchematicTar(projectPath, &options.TarIncludePatterns)
	if tarballErr != nil {
		return fmt.Errorf("error creating tar file: %w", tarballErr)
	}
	defer os.Remove(tarballName) // just to cleanup

	// create a new empty workspace, resulting in "draft" status
	_, wsErr := svc.CreateTestWorkspace(options.Prefix, options.ResourceGroup, options.Tags)
	if wsErr != nil {
		return fmt.Errorf("error creating new schematic workspace: %w", wsErr)
	}

	// upload the terraform code
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
	updateErr := svc.UpdateTestTemplateVars(options.TerraformVars)
	if updateErr != nil {
		return fmt.Errorf("error updating template with Variablestore: %w", updateErr)
	}

	// ------ PLAN ------
	planResponse, planErr := svc.CreatePlanJob()
	if planErr != nil {
		return fmt.Errorf("error creating PLAN: %w", planErr)
	}
	planJobStatus, planStatusErr := svc.WaitForFinalJobStatus(*planResponse.Activityid)
	if planStatusErr != nil {
		return fmt.Errorf("error waiting for PLAN to finish: %w", planStatusErr)
	}
	if planJobStatus != SchematicsJobStatusCompleted {
		return fmt.Errorf("PLAN has failed with status %s", planJobStatus)
	}

	return nil
}
