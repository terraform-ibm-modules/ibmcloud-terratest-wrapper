package testschematic

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/schematics-go-sdk/schematicsv1"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/testhelper"
)

func (options *TestSchematicOptions) RunSchematicTest() error {

	// create new schematic service with authenticator, set pointer of service in options for use later
	var svcErr error
	options.SchematicsSvc, svcErr = CreateSchematicsService(options.RequiredEnvironmentVars[ibmcloudApiKeyVar])
	if svcErr != nil {
		return svcErr
	}

	// get the root path of this project
	projectPath, pathErr := testhelper.GitRootPath(".")
	if pathErr != nil {
		return pathErr
	}

	// create a new tar file for the project
	tarballName, tarballErr := CreateSchematicTar(projectPath, &options.TarIncludePatterns)
	if tarballErr != nil {
		return tarballErr
	}
	defer os.Remove(tarballName) // just to cleanup

	// create a new empty workspace, resulting in "draft" status
	workspace, wsErr := CreateTestWorkspace(options.SchematicsSvc, options.Prefix, options.ResourceGroup, options.Tags, options)
	if wsErr != nil {
		return wsErr
	}

	workspaceID := *workspace.ID
	templateID := *workspace.TemplateData[0].ID

	// upload the terraform code
	uploadErr := UploadTarToWorkspace(options.SchematicsSvc, workspaceID, templateID, tarballName)
	if uploadErr != nil {
		return uploadErr
	}

	// wait for the upload terraform action to complete
	time.Sleep(1 * time.Minute)

	// update the default template with variables
	// NOTE: doing this AFTER terraform is loaded so that sensitive variables in Variablestore are already created in template,
	// to prevent things like api keys being exposed
	updateErr := UpdateTestTemplateVars(options.SchematicsSvc, workspaceID, templateID, options.TerraformVars)
	if updateErr != nil {
		return updateErr
	}

	return nil
}

func CreateSchematicsService(ibmcloudApiKey string) (SchematicsSvcI, error) {

	schematicsSvc, newErr := schematicsv1.NewSchematicsV1(&schematicsv1.SchematicsV1Options{
		URL: "https://schematics.cloud.ibm.com",
		Authenticator: &core.IamAuthenticator{
			ApiKey: ibmcloudApiKey, // pragma: allowlist secret
		},
	})
	if newErr != nil {
		return nil, newErr
	}

	return schematicsSvc, nil
}

func CreateTestWorkspace(svc SchematicsSvcI, name string, resourceGroup string, tags []string, options *TestSchematicOptions) (*schematicsv1.WorkspaceResponse, error) {

	// create env and input vars template
	templateModel := &schematicsv1.TemplateSourceDataRequest{
		Folder: core.StringPtr("."),
		Type:   core.StringPtr("terraform_v1.2"),
		EnvValues: []interface{}{
			map[string]string{
				"__netrc__": fmt.Sprintf("[['github.ibm.com','%s','%s']]", options.RequiredEnvironmentVars[gitUser], options.RequiredEnvironmentVars[gitToken]),
			},
			map[string]string{
				"API_DATA_IS_SENSITIVE": "true", // for RestAPI provider
			},
		},
		EnvValuesMetadata: []schematicsv1.EnvironmentValuesMetadata{
			{Name: core.StringPtr("__netrc__"), Hidden: core.BoolPtr(false), Secure: core.BoolPtr(true)},
			{Name: core.StringPtr("API_DATA_IS_SENSITIVE"), Hidden: core.BoolPtr(false), Secure: core.BoolPtr(false)},
		},
	}

	createWorkspaceOptions := &schematicsv1.CreateWorkspaceOptions{
		Description:   core.StringPtr("Goldeneye CI Test for " + name),
		Name:          core.StringPtr(name),
		TemplateData:  []schematicsv1.TemplateSourceDataRequest{*templateModel},
		Type:          []string{"terraform_v1.2"},
		Location:      core.StringPtr(defaultRegion),
		ResourceGroup: core.StringPtr(resourceGroup),
		Tags:          tags,
	}

	workspace, _, workspaceErr := svc.CreateWorkspace(createWorkspaceOptions)
	if workspaceErr != nil {
		return nil, workspaceErr
	}

	return workspace, nil
}

func UpdateTestTemplateVars(svc SchematicsSvcI, workspaceID string, templateID string, vars []TestSchematicTerraformVar) error {

	// set up an array of workspace variables based on TerraformVars supplied.
	var strVal string
	var strErr error
	variables := []schematicsv1.WorkspaceVariableRequest{}
	for _, tfVar := range vars {
		// if tfVal is an array, convert to json array string
		if IsArray(tfVar.Value) {
			strVal, strErr = ConvertArrayToJsonString(tfVar.Value)
			if strErr != nil {
				return strErr
			}
		} else {
			strVal = fmt.Sprintf("%v", tfVar.Value)
		}
		variables = append(variables, schematicsv1.WorkspaceVariableRequest{
			Name:   core.StringPtr(tfVar.Name),
			Value:  core.StringPtr(strVal),
			Type:   core.StringPtr(tfVar.DataType),
			Secure: core.BoolPtr(tfVar.Secure),
		})
	}

	templateModel := &schematicsv1.ReplaceWorkspaceInputsOptions{
		WID:           core.StringPtr(workspaceID),
		TID:           core.StringPtr(templateID),
		Variablestore: variables,
	}

	// now update template
	_, _, updateErr := svc.ReplaceWorkspaceInputs(templateModel)
	if updateErr != nil {
		return updateErr
	}

	return nil
}

func UploadTarToWorkspace(svc SchematicsSvcI, workspaceID string, templateID string, tarPath string) error {
	fileReader, _ := os.Open(tarPath)
	fileReaderWrapper := io.NopCloser(fileReader)

	uploadTarOptions := &schematicsv1.TemplateRepoUploadOptions{
		WID:             core.StringPtr(workspaceID),
		TID:             core.StringPtr(templateID),
		File:            fileReaderWrapper,
		FileContentType: core.StringPtr("application/octet-stream"),
	}

	_, _, uploadErr := svc.TemplateRepoUpload(uploadTarOptions)
	if uploadErr != nil {
		return uploadErr
	}

	return nil
}
