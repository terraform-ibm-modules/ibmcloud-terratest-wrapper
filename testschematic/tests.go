package testschematic

import (
	"fmt"
	"os"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

func (options *TestSchematicOptions) RunSchematicTest() error {

	// create new schematic service with authenticator, set pointer of service in options for use later
	var svcErr error
	options.SchematicsSvc, svcErr = CreateSchematicsService(options.RequiredEnvironmentVars[ibmcloudApiKeyVar])
	if svcErr != nil {
		return fmt.Errorf("error creating schematics sdk service: %w", svcErr)
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
	workspace, wsErr := CreateTestWorkspace(options.SchematicsSvc, options.Prefix, options.ResourceGroup, options.Tags, options)
	if wsErr != nil {
		return fmt.Errorf("error creating new schematic workspace: %w", wsErr)
	}

	workspaceID := *workspace.ID
	templateID := *workspace.TemplateData[0].ID

	// upload the terraform code
	uploadErr := UploadTarToWorkspace(options.SchematicsSvc, workspaceID, templateID, tarballName)
	if uploadErr != nil {
		return fmt.Errorf("error uploading tar file to workspace: %w", uploadErr)
	}

	// -------- UPLOAD TAR FILE ----------
	// find the tar upload job
	uploadJob, uploadJobErr := FindLatestWorkspaceJobByName(options.SchematicsSvc, workspaceID, SchematicsJobTypeUpload)
	if uploadJobErr != nil {
		return fmt.Errorf("error finding the upload tar action: %w", uploadJobErr)
	}
	// wait for it to finish
	uploadJobStatus, uploadJobStatusErr := WaitForFinalJobStatus(options.SchematicsSvc, workspaceID, templateID, *uploadJob.ActionID, options)
	if uploadJobStatusErr != nil {
		return fmt.Errorf("error waiting for upload of tar to finish: %w", uploadJobStatusErr)
	}
	// check if complete
	if uploadJobStatus != SchematicsJobStatusCompleted {
		return fmt.Errorf("tar upload has failed with status %s", uploadJobStatus)
	}

	// update the default template with variables
	// NOTE: doing this AFTER terraform is loaded so that sensitive variables in Variablestore are already created in template,
	// to prevent things like api keys being exposed
	updateErr := UpdateTestTemplateVars(options.SchematicsSvc, workspaceID, templateID, options.TerraformVars)
	if updateErr != nil {
		return fmt.Errorf("error updating template with Variablestore: %w", updateErr)
	}

	return nil
}
