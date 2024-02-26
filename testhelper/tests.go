package testhelper

import (
	"encoding/json"
	"fmt"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	tfjson "github.com/hashicorp/terraform-json"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gruntwork-io/terratest/modules/files"

	"github.com/go-git/go-git/v5"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func (options *TestOptions) skipUpgradeTest(source_repo string, branch string) bool {

	// Set upstream to the source repo
	remote_out, remote_err := exec.Command("/bin/sh", "-c", "git remote add upstream "+source_repo).Output()
	if remote_err != nil {
		logger.Log(options.Testing, "Add remote output:\n", remote_out)
		logger.Log(options.Testing, "Error adding upstream remote:\n", remote_err)
		return false
	}
	// Fetch the source repo
	fetch_out, fetch_err := exec.Command("/bin/sh", "-c", "git fetch upstream -f").Output()
	if fetch_err != nil {
		logger.Log(options.Testing, "Fetch output:\n", fetch_out)
		logger.Log(options.Testing, "Error fetching upstream:\n", fetch_err)
	} else {
		logger.Log(options.Testing, "Fetch output:\n", fetch_out)
	}
	// Get all the commit messages from the PR branch
	// NOTE: using the "origin" of the default branch as the start point, which will exist in a fresh
	// clone even if the default branch has not been checked out or pulled.
	cmd := exec.Command("/bin/sh", "-c", "git log upstream/master..", branch)
	out, _ := cmd.CombinedOutput()

	fmt.Printf("Commit Messages (master): \n%s", string(out))
	// Skip upgrade Test if BREAKING CHANGE OR SKIP UPGRADE TEST string found in commit messages
	doNotRunUpgradeTest := false
	if (strings.Contains(string(out), "BREAKING CHANGE") || strings.Contains(string(out), "SKIP UPGRADE TEST")) && !strings.Contains(string(out), "UNSKIP UPGRADE TEST") {
		doNotRunUpgradeTest = true
	}
	if !doNotRunUpgradeTest {
		// NOTE: using the "origin" of the default branch as the start point, which will exist in a fresh
		// clone even if the default branch has not been checked out or pulled.
		cmd = exec.Command("/bin/sh", "-c", "git log upstream/main..", branch)
		out, _ = cmd.CombinedOutput()

		fmt.Printf("Commit messages (main): \n%s", string(out))
		if (strings.Contains(string(out), "BREAKING CHANGE") || strings.Contains(string(out), "SKIP UPGRADE TEST")) && !strings.Contains(string(out), "UNSKIP UPGRADE TEST") {
			doNotRunUpgradeTest = true
		}
	}
	return doNotRunUpgradeTest
}

// sanitizeResourceChanges sanitizes the sensitive data in a Terraform JSON Change and returns the sanitized JSON.
func sanitizeResourceChanges(change *tfjson.Change, mergedSensitive map[string]interface{}) (string, error) {
	// Marshal the Change to JSON bytes
	changesBytes, err := json.MarshalIndent(change, "", "  ")
	if err != nil {
		return "", err
	}
	changesJson := string(changesBytes)

	// Perform sanitization of sensitive data
	changesJson, err = common.SanitizeSensitiveData(changesJson, mergedSensitive)
	return changesJson, err
}

// handleSanitizationError logs an error message if a sanitization error occurs.
func handleSanitizationError(err error, location string, options *TestOptions) {
	if err != nil {
		errorMessage := fmt.Sprintf("Error sanitizing sensitive data in %s", location)
		logger.Log(options.Testing, errorMessage)
	}
}

// checkConsistency Fails the test if any destroys are detected and the resource is not exempt.
// If any addresses are provided in IgnoreUpdates.List then fail on updates too unless the resource is exempt
func (options *TestOptions) CheckConsistency(plan *terraform.PlanStruct) {
	options.checkConsistency(plan)
}

// checkConsistency check consistency
func (options *TestOptions) checkConsistency(plan *terraform.PlanStruct) {
	validChange := false

	for _, resource := range plan.ResourceChangesMap {
		// get JSON string of full changes for the logs
		changesBytes, changesErr := json.MarshalIndent(resource.Change, "", "  ")
		// if it errors in the marshall step, just put a placeholder and move on, not important
		changesJson := "--UNAVAILABLE--"
		if changesErr == nil {
			changesJson = string(changesBytes)
		}

		var resourceDetails string

		// Treat all keys in the BeforeSensitive and AfterSensitive maps as sensitive
		// Assuming BeforeSensitive and AfterSensitive are of type interface{}
		beforeSensitive, beforeSensitiveOK := resource.Change.BeforeSensitive.(map[string]interface{})
		afterSensitive, afterSensitiveOK := resource.Change.AfterSensitive.(map[string]interface{})

		// Create the mergedSensitive map
		mergedSensitive := make(map[string]interface{})

		// Check if BeforeSensitive is of the expected type
		if beforeSensitiveOK {
			// Copy the keys and values from BeforeSensitive to the mergedSensitive map.
			for key, value := range beforeSensitive {
				mergedSensitive[key] = value
			}
		}

		// Check if AfterSensitive is of the expected type
		if afterSensitiveOK {
			// Copy the keys and values from AfterSensitive to the mergedSensitive map.
			for key, value := range afterSensitive {
				mergedSensitive[key] = value
			}
		}

		// Perform sanitization
		sanitizedChangesJson, err := sanitizeResourceChanges(resource.Change, mergedSensitive)
		if err != nil {
			sanitizedChangesJson = "Error sanitizing sensitive data"
			logger.Log(options.Testing, sanitizedChangesJson)
		}
		formatChangesJson, err := common.FormatJsonStringPretty(sanitizedChangesJson)

		var formatChangesJsonString string
		if err != nil {
			logger.Log(options.Testing, "Error formatting JSON, use unformatted")
			formatChangesJsonString = sanitizedChangesJson
		} else {
			formatChangesJsonString = string(formatChangesJson)
		}

		diff, diffErr := common.GetBeforeAfterDiff(changesJson)

		if diffErr != nil {
			diff = fmt.Sprintf("Error getting diff: %s", diffErr)
		} else {
			// Split the changesJson into "Before" and "After" parts
			beforeAfter := strings.Split(diff, "After: ")

			// Perform sanitization on "After" part
			var after string
			if len(beforeAfter) > 1 {
				after, err = common.SanitizeSensitiveData(beforeAfter[1], mergedSensitive)
				handleSanitizationError(err, "after diff", options)
			} else {
				after = fmt.Sprintf("Could not parse after from diff") // dont print incase diff contains sensitive values
			}

			// Perform sanitization on "Before" part
			var before string
			if len(beforeAfter) > 0 {
				before, err = common.SanitizeSensitiveData(strings.TrimPrefix(beforeAfter[0], "Before: "), mergedSensitive)
				handleSanitizationError(err, "before diff", options)
			} else {
				before = fmt.Sprintf("Could not parse before from diff") // dont print incase diff contains sensitive values
			}

			// Reassemble the sanitized diff string
			diff = "  Before: \n\t" + before + "\n  After: \n\t" + after
		}
		resourceDetails = fmt.Sprintf("\nName: %s\nAddress: %s\nActions: %s\nDIFF:\n%s\n\nChange Detail:\n%s", resource.Name, resource.Address, resource.Change.Actions, diff, formatChangesJsonString)

		var errorMessage string
		if !options.IgnoreDestroys.IsExemptedResource(resource.Address) {
			errorMessage = fmt.Sprintf("Resource(s) identified to be destroyed %s", resourceDetails)
			assert.False(options.Testing, resource.Change.Actions.Delete(), errorMessage)
			assert.False(options.Testing, resource.Change.Actions.DestroyBeforeCreate(), errorMessage)
			assert.False(options.Testing, resource.Change.Actions.CreateBeforeDestroy(), errorMessage)
			validChange = true
		}
		if !options.IgnoreUpdates.IsExemptedResource(resource.Address) {
			errorMessage = fmt.Sprintf("Resource(s) identified to be updated %s", resourceDetails)
			assert.False(options.Testing, resource.Change.Actions.Update(), errorMessage)
			validChange = true
		}
		// We only want to check pure Adds (creates without destroy) if the consistency test is
		// NOT the result of an Upgrade, as some adds are expected when doing the Upgrade test
		// (such as new resources were added as part of the pull request)
		if !options.IsUpgradeTest {
			if !options.IgnoreAdds.IsExemptedResource(resource.Address) {
				errorMessage = fmt.Sprintf("Resource(s) identified to be created %s", resourceDetails)
				assert.False(options.Testing, resource.Change.Actions.Create(), errorMessage)
				validChange = true
			}
		}
	}
	// Run plan again to output the nice human-readable plan if there are valid changes
	if validChange {
		terraform.Plan(options.Testing, options.TerraformOptions)
	}
}

// Function to setup testing environment.
//
// Summary of settings:
// * API_DATA_IS_SENSITIVE environment variable is set to true
// * If calling test had not provided its own TerraformOptions, then default settings are used
// * Temp directory is created
func (options *TestOptions) TestSetup() {
	oldSetupValue := options.SkipTestSetup
	options.SkipTestSetup = false
	options.testSetup()
	options.SkipTestSetup = oldSetupValue
}

// testSetup Setup test
func (options *TestOptions) testSetup() {
	if !options.SkipTestSetup {
		os.Setenv("API_DATA_IS_SENSITIVE", "true")
		// If calling test had not provided its own TerraformOptions, use the default settings
		if options.TerraformOptions == nil {
			// Construct the terraform options with default retryable errors to handle the most common
			// retryable errors in terraform testing.
			options.TerraformOptions = terraform.WithDefaultRetryableErrors(options.Testing, &terraform.Options{
				// Set the path to the Terraform code that will be tested.
				TerraformDir: options.TerraformDir,
				Vars:         options.TerraformVars,
				// Set Upgrade to true to ensure the latest version of providers and modules are used by terratest.
				// This is the same as setting the -upgrade=true flag with terraform.
				Upgrade: true,
			})
		}

		if !options.DisableTempWorkingDir {
			// Ensure always running from git root
			gitRoot, err := common.GitRootPath(".")

			if err != nil {
				require.Nil(options.Testing, err, "Error getting git root path")
			}

			// Create a temporary directory
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("terraform-%s", options.Prefix))
			if err != nil {
				logger.Log(options.Testing, err)
			} else {
				logger.Log(options.Testing, "TEMP CREATED: ", tempDir)

				// To avoid workspace collisions when running in parallel, ignoring any temp terraform files
				// NOTE: if it is an upgrade test, we need hidden .git files
				tempDirFilter := func(path string) bool {
					if !options.IsUpgradeTest && files.PathContainsHiddenFileOrFolder(path) {
						return false
					}
					if files.PathContainsTerraformStateOrVars(path) || files.PathIsTerraformLockFile(path) {
						return false
					}

					return true
				}

				// Define the source and destination directories for the directory copy
				srcDir := gitRoot
				dstDir := tempDir

				// Use CopyDirectory to copy the source directory to the destination directory with the filter
				err := common.CopyDirectory(srcDir, dstDir, tempDirFilter)
				if err != nil {
					require.Nil(options.Testing, err, "Error copying directory")
				}

				// Update Terraform options with the full path of the new temp location
				options.setTerraformDir(path.Join(dstDir, options.TerraformDir))
			}
		}

		options.WorkspacePath = options.TerraformOptions.TerraformDir
		if options.UseTerraformWorkspace {
			// Always run in a new clean workspace to avoid reusing existing state files
			options.WorkspaceName = terraform.WorkspaceSelectOrNew(options.Testing, options.TerraformOptions, options.Prefix)
			options.WorkspacePath = fmt.Sprintf("%s/terraform.tfstate.d/%s", options.WorkspacePath, options.Prefix)
		}
	} else {
		logger.Log(options.Testing, "Skipping automatic Test Setup")
	}
}

// Function to destroy all resources. Resources are not destroyed if tests failed and "DO_NOT_DESTROY_ON_FAILURE" environment variable is true.
// If options.ImplicitDestroy is set then these resources from the State file are removed to allow implicit destroy.
func (options *TestOptions) TestTearDown() {
	oldTearDownValue := options.SkipTestTearDown
	options.SkipTestTearDown = false
	options.testTearDown()
	options.SkipTestTearDown = oldTearDownValue
}

// testTearDown Tear down test
func (options *TestOptions) testTearDown() {
	// Get the output of the last terraform apply
	// NOTE: this is done before the destroy so that the output is available for debugging
	var outputErr error

	// Turn off logging for this step so sensitive data is not logged
	options.TerraformOptions.Logger = logger.Discard
	options.LastTestTerraformOutputs, outputErr = terraform.OutputAllE(options.Testing, options.TerraformOptions)
	options.TerraformOptions.Logger = logger.Default // turn log back on

	if outputErr != nil {
		logger.Log(options.Testing, "failed to get terraform output: ", outputErr)
	}

	if !options.SkipTestTearDown {
		// Check if "DO_NOT_DESTROY_ON_FAILURE" is set
		envVal, _ := os.LookupEnv("DO_NOT_DESTROY_ON_FAILURE")

		// Do not destroy if tests failed and "DO_NOT_DESTROY_ON_FAILURE" is true
		if options.Testing.Failed() && strings.ToLower(envVal) == "true" {
			fmt.Println("Terratest failed. Debug the Test and delete resources manually.")
		} else {

			for _, address := range options.ImplicitDestroy {
				// TODO: is this the correct path to the state file? and/or does it need to be updated upstream to a relative path(temp dir)?
				statefile := fmt.Sprintf("%s/terraform.tfstate", options.WorkspacePath)
				out, err := RemoveFromStateFile(statefile, address)
				if options.ImplicitRequired && err != nil {
					logger.Log(options.Testing, out)
					assert.Nil(options.Testing, err, "Could not remove from state file")
				} else {
					logger.Log(options.Testing, out)
				}
			}

			if options.CBRRuleListOutputVariable != "" {
				// Disable any CBR Rules before procceding with destroy
				expected_outputs := []string{options.CBRRuleListOutputVariable}
				_, err := ValidateTerraformOutputs(options.LastTestTerraformOutputs, expected_outputs...)
				if err == nil {
					cbr_rule_ids := options.LastTestTerraformOutputs[options.CBRRuleListOutputVariable].([]interface{})
					infosvc, err := cloudinfo.NewCloudInfoServiceFromEnv(ibmcloudApiKeyVar, cloudinfo.CloudInfoServiceOptions{})
					if err != nil {
						logger.Log(options.Testing, "Error creating CloudInfoService for testhelper, skipping CBR Rule disable")
					} else {
						for _, cbr_rule_id := range cbr_rule_ids {
							// Disable CBR Rule
							disable_error := infosvc.SetCBREnforcementMode(cbr_rule_id.(string), "disabled")
							if disable_error != nil {
								logger.Log(options.Testing, fmt.Sprintf("Error Disabling CBR Rule %s, %s", cbr_rule_id.(string), disable_error))
							} else {
								logger.Log(options.Testing, fmt.Sprintf("Disabled CBR Rule %s", cbr_rule_id.(string)))
							}
						}
					}
				} else {
					logger.Log(options.Testing, fmt.Sprintf("Error output containg CBRRuleList %s not found in Statefile, skipping CBR Rule disable", options.CBRRuleListOutputVariable))
				}
			}
			logger.Log(options.Testing, "START: Destroy")
			destroyOutput, destroyError := terraform.DestroyE(options.Testing, options.TerraformOptions)
			if assert.NoError(options.Testing, destroyError) == false {
				logger.Log(options.Testing, destroyError)
				// On destroy resource group failure, list remaining resources
				if common.StringContainsIgnoreCase(destroyError.Error(), "Error Deleting resource group") {
					logger.Log(options.Testing, "ERROR: Destroy failed attempting to list remaining resources")
					if options.LastTestTerraformOutputs != nil {
						// Check if resource_group_id or resource_group_ids are in the outputs
						expectedOutputs := []string{"resource_group_id", "resource_group_ids", "resource_group_name", "resource_group_names"}
						missingOutputs, _ := ValidateTerraformOutputs(options.LastTestTerraformOutputs, expectedOutputs...)
						actualOutputs := []string{}
						if missingOutputs != nil {
							// loop through expected outputs and if they are not in missing outputs then add them to actual outputs
							for _, expectedOutput := range expectedOutputs {
								if !common.StrArrayContains(missingOutputs, expectedOutput) {
									actualOutputs = append(actualOutputs, expectedOutput)
								}
							}
						} else {
							actualOutputs = append(actualOutputs, expectedOutputs...)
						}
						// If resource_group_id or resource_group_ids are in the outputs then list resources in the resource group
						if len(actualOutputs) > 0 {
							cloudInfoSvc, err := cloudinfo.NewCloudInfoServiceFromEnv(ibmcloudApiKeyVar, cloudinfo.CloudInfoServiceOptions{})
							if err != nil {
								logger.Log(options.Testing, "Error creating CloudInfoService for testhelper, skipping resource listing")
							} else {
								if common.StrArrayContains(actualOutputs, "resource_group_id") {
									resourceGroupID := options.LastTestTerraformOutputs["resource_group_id"].(string)
									logger.Log(options.Testing, fmt.Sprintf("Resource Group %s", resourceGroupID))
									// Get all resources in resource group
									resources, err := cloudInfoSvc.ListResourcesByGroupID(resourceGroupID)
									print_resources(options.Testing, resourceGroupID, resources, err)
								}
								if common.StrArrayContains(actualOutputs, "resource_group_ids") {
									resourceGroupIDs := options.LastTestTerraformOutputs["resource_group_ids"].([]interface{})
									for _, resourceGroupID := range resourceGroupIDs {
										// Get all resources in resource group
										logger.Log(options.Testing, fmt.Sprintf("Resource Group %s", resourceGroupID.(string)))
										resources, err := cloudInfoSvc.ListResourcesByGroupID(resourceGroupID.(string))
										print_resources(options.Testing, resourceGroupID.(string), resources, err)
									}
								}
								if common.StrArrayContains(actualOutputs, "resource_group_name") {
									resourceGroup := options.LastTestTerraformOutputs["resource_group_name"].(string)
									logger.Log(options.Testing, fmt.Sprintf("Resource Group %s", resourceGroup))
									// Get all resources in resource group
									resources, err := cloudInfoSvc.ListResourcesByGroupName(resourceGroup)
									print_resources(options.Testing, resourceGroup, resources, err)
								}
								if common.StrArrayContains(actualOutputs, "resource_group_names") {
									resourceGroups := options.LastTestTerraformOutputs["resource_group_names"].([]interface{})
									for _, resourceGroup := range resourceGroups {
										// Get all resources in resource group
										logger.Log(options.Testing, fmt.Sprintf("Resource Group %s", resourceGroup.(string)))
										resources, err := cloudInfoSvc.ListResourcesByGroupName(resourceGroup.(string))
										print_resources(options.Testing, resourceGroup.(string), resources, err)
									}
								}
							}
						}
					}
				}
			} else {
				logger.Log(options.Testing, destroyOutput)
			}
			if options.UseTerraformWorkspace {
				terraform.WorkspaceDelete(options.Testing, options.TerraformOptions, options.Prefix)
			}
			logger.Log(options.Testing, "END: Destroy")

			//Clean up terraform files
			CleanTerraformDir(options.TerraformDir)
		}
	} else {
		logger.Log(options.Testing, "Skipping automatic Test Teardown")
	}
}

// print_resources internal helper function that prints the resources in the resource group
func print_resources(t *testing.T, resourceGroup string, resources []resourcecontrollerv2.ResourceInstance, err error) {
	logger.Log(t, fmt.Sprintf("---------------------------"))
	if err != nil {
		logger.Log(t, fmt.Sprintf("Error listing resources in Resource Group %s, %s\n"+
			"Is this Resource Group already deleted?", resourceGroup, err))
	} else if len(resources) == 0 {
		logger.Log(t, fmt.Sprintf("No resources found in Resource Group %s", resourceGroup))
	} else {
		logger.Log(t, fmt.Sprintf("Resources in Resource Group %s:", resourceGroup))
		cloudinfo.PrintResources(resources)
	}
	logger.Log(t, fmt.Sprintf("---------------------------"))
}

// RunTestUpgrade runs the upgrade test to ensure that the Terraform configurations being tested
// do not result in any resources being destroyed during an upgrade. This is crucial to ensure that
// existing infrastructure remains intact during updates.
//
// The function performs the following steps:
//  1. Checks if the test is running in short mode and skips the upgrade test if so.
//  2. Determines the current PR branch.
//  3. Checks if the upgrade test should be skipped based on commit messages.
//  4. If not skipped:
//     a. Sets up the test environment, including creating temporary directories.
//     b. Copies the current code (from the PR branch) to a temporary directory.
//     c. Clones the base branch into a separate temporary directory.
//     d. Applies Terraform configurations on the base branch.
//     e. Moves the state file from the base branch directory to the PR branch directory.
//     f. Runs Terraform plan in the PR branch directory to check for any inconsistencies.
//     g. Optionally, it can also apply the Terraform configurations on the PR branch.
//
// Parameters:
// - options: TestOptions containing various settings and configurations for the test.
//
// Returns:
// - A terraform.PlanStruct containing the results of the Terraform plan.
// - An error if any step in the function fails.
func (options *TestOptions) RunTestUpgrade() (*terraform.PlanStruct, error) {

	var result *terraform.PlanStruct
	var resultErr error

	skipped := true

	// Skip upgrade Test in continuous testing pipeline which runs in short mode
	if testing.Short() {
		options.Testing.Skip("Skipping upgrade Test in short mode.")
	}

	// Determine the name of the PR branch
	_, prBranch, err := common.GetCurrentPrRepoAndBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to determine the PR repo and branch: %v", err)
	} else {
		logger.Log(options.Testing, "PR Branch:", prBranch)
	}

	baseRepo, baseBranch := common.GetBaseRepoAndBranch(options.BaseTerraformRepo, options.BaseTerraformBranch)
	if baseBranch == "" || baseRepo == "" {
		// No need to tearDown as nothing was created
		return nil, fmt.Errorf("failed to get default repo and branch: %s %s", baseRepo, baseBranch)
	} else {
		logger.Log(options.Testing, "Base Repo:", baseRepo)
		logger.Log(options.Testing, "Base Branch:", baseBranch)
	}

	if options.skipUpgradeTest(baseRepo, prBranch) {
		options.Testing.Log("Detected the string \"BREAKING CHANGE\" or \"SKIP UPGRADE TEST\" used in commit message, skipping upgrade Test.")
	} else {
		skipped = false
		options.IsUpgradeTest = true

		gitRoot, err := common.GitRootPath(".")
		if err != nil {
			return nil, fmt.Errorf("failed to get git root path: %v", err)
		}

		// Extract the relative path from the original TerraformDir
		originalTerraformDir := options.TerraformDir

		// Just in case an absolute path was provided, make it relative to the git root
		relativeTestSampleDir := strings.TrimPrefix(originalTerraformDir, gitRoot)

		// Disable the creation of a temporary directory in test setup, Upgrade Test will create its own
		// Backup the original value of DisableTempWorkingDir
		tempDirCreationBackup := options.DisableTempWorkingDir

		// Temporarily disable the creation of a temporary directory
		// Upgrade Test will create its own
		options.DisableTempWorkingDir = true

		// Temporarily disable workspace usage
		useTerraformWorkspaceBackup := options.UseTerraformWorkspace
		terraformWorkspaceBackup := options.WorkspacePath
		options.UseTerraformWorkspace = false
		logger.Log(options.Testing, "Temporarily disabling UseTerraformWorkspace in Upgrade Test as temporary directories are used instead of workspaces")
		defer func() {
			logger.Log(options.Testing, fmt.Sprintf("Restoring UseTerraformWorkspace and WorkspacePath to original values: %v %v", useTerraformWorkspaceBackup, terraformWorkspaceBackup))
			options.UseTerraformWorkspace = useTerraformWorkspaceBackup
			options.WorkspacePath = terraformWorkspaceBackup
		}()

		// Setup the test
		options.testSetup()
		// restore the original value
		options.DisableTempWorkingDir = tempDirCreationBackup

		prTempDir := gitRoot
		baseTempDir := ""
		if !options.DisableTempWorkingDir {
			// Create a temporary directory for the PR code
			prTempDir, err = os.MkdirTemp("", fmt.Sprintf("terraform-pr-%s", options.Prefix))
			if err != nil {
				// No need to tearDown as nothing was created
				return nil, fmt.Errorf("failed to create temp dir for PR branch: %v", err)
			} else {
				logger.Log(options.Testing, "TEMP PR DIR CREATED: ", prTempDir)
			}
			if !options.SkipTestTearDown {
				defer os.RemoveAll(prTempDir) // clean up
			}

			// Create a temporary directory for the base branch
			baseTempDir, err = os.MkdirTemp("", fmt.Sprintf("terraform-base-%s", options.Prefix))
			if err != nil {
				// No need to tearDown as nothing was created
				return nil, fmt.Errorf("failed to create temp dir for base branch: %v", err)
			} else {
				logger.Log(options.Testing, "TEMP UPGRADE BASE DIR CREATED: ", baseTempDir)
			}
			if !options.SkipTestTearDown {
				defer os.RemoveAll(baseTempDir) // clean up
			}

			// Copy the current code (from PR branch) to the PR temp directory with the filter
			errCopy := common.CopyDirectory(gitRoot, prTempDir, func(path string) bool {
				// Skip terraform state files or .terraform directories
				// Or terraform lock files
				// Do not skip .git directories as we need them for the upgrade test
				if files.PathContainsTerraformStateOrVars(path) ||
					files.PathIsTerraformLockFile(path) ||
					common.StringContainsIgnoreCase(path, ".terraform") {
					return false
				}

				return true
			})
			if errCopy != nil {
				// No need to tearDown as nothing was created
				return nil, fmt.Errorf("failed to copy PR directory to temp: %v", errCopy)
			} else {
				logger.Log(options.Testing, "Copied current code to PR branch dir:", prTempDir)
			}
		} else {
			// create temp dir for base branch in git root
			// This directory never gets deleted by automation if teardown is skipped
			baseTempDir, err = os.MkdirTemp("", baseTempDir)
			if err != nil {
				// No need to tearDown as nothing was created
				return nil, fmt.Errorf("failed to create temp dir for base branch in git root: %v", err)
			} else {
				logger.Log(options.Testing, "TEMP UPGRADE BASE DIR CREATED: ", baseTempDir)
			}
			if !options.SkipTestTearDown {
				defer os.RemoveAll(baseTempDir) // clean up
			}
		}

		authMethod, err := common.DetermineAuthMethod(baseRepo)
		if err != nil {
			logger.Log(options.Testing, "Failed to determine authentication method, trying without authentication...")

			// Convert SSH URL to HTTPS URL
			if strings.HasPrefix(baseRepo, "git@") {
				baseRepo = strings.Replace(baseRepo, ":", "/", 1)
				baseRepo = strings.Replace(baseRepo, "git@", "https://", 1)
				baseRepo = strings.TrimSuffix(baseRepo, ".git") + ".git"
			}

			// Try to clone without authentication
			_, errUnauth := git.PlainClone(baseTempDir, false, &git.CloneOptions{
				URL:           baseRepo,
				ReferenceName: plumbing.NewBranchReferenceName(baseBranch),
				SingleBranch:  true,
			})

			if errUnauth != nil {
				// If unauthenticated clone fails and we cannot determine authentication, return the error from the unauthenticated approach
				return nil, fmt.Errorf("failed to determine authentication method and clone base repo and branch without authentication: %v", errUnauth)
			} else {
				logger.Log(options.Testing, "Cloned base repo and branch without authentication")
			}
		} else {
			// Authentication method determined, try with authentication
			_, errAuth := git.PlainClone(baseTempDir, false, &git.CloneOptions{
				URL:           baseRepo,
				ReferenceName: plumbing.NewBranchReferenceName(baseBranch),
				SingleBranch:  true,
				Auth:          authMethod,
			})

			if errAuth != nil {
				logger.Log(options.Testing, "Failed to clone base repo and branch with authentication, trying without authentication...")
				// Convert SSH URL to HTTPS URL
				if strings.HasPrefix(baseRepo, "git@") {
					baseRepo = strings.Replace(baseRepo, ":", "/", 1)
					baseRepo = strings.Replace(baseRepo, "git@", "https://", 1)
					baseRepo = strings.TrimSuffix(baseRepo, ".git") + ".git"
				}

				// Try to clone without authentication
				_, errUnauth := git.PlainClone(baseTempDir, false, &git.CloneOptions{
					URL:           baseRepo,
					ReferenceName: plumbing.NewBranchReferenceName(baseBranch),
					SingleBranch:  true,
				})

				if errUnauth != nil {
					// If unauthenticated clone also fails, return the error from the authenticated approach
					return nil, fmt.Errorf("failed to clone base repo and branch with authentication: %v", errAuth)
				} else {
					logger.Log(options.Testing, "Cloned base repo and branch without authentication")
				}
			} else {
				logger.Log(options.Testing, "Cloned base repo and branch with authentication")
			}
		}

		// Set TerraformDir to the appropriate directory within baseTempDir
		options.setTerraformDir(path.Join(baseTempDir, relativeTestSampleDir))

		logger.Log(options.Testing, "Init / Apply on Base repo:", baseRepo)
		logger.Log(options.Testing, "Init / Apply on Base branch:", baseBranch)
		logger.Log(options.Testing, "Init / Apply on Base branch dir:", options.TerraformOptions.TerraformDir)

		_, resultErr = terraform.InitAndApplyE(options.Testing, options.TerraformOptions)
		if resultErr != nil {
			assert.Nilf(options.Testing, resultErr, "Terraform Apply on Base branch has failed")
			options.testTearDown()
			return nil, resultErr
		}
		// Get the path to the state file in baseTempDir
		baseStatePath := path.Join(options.TerraformOptions.TerraformDir, "terraform.tfstate")

		// Set TerraformDir to the appropriate directory within prTempDir
		options.setTerraformDir(path.Join(prTempDir, relativeTestSampleDir))

		// ensure terraform working files/folders are removed before copying state file ie .terraform, .terraform.lock.hcl, terraform.tfstate, terraform.tfstate.backup
		CleanTerraformDir(options.TerraformOptions.TerraformDir)

		// Copy the state file to the corresponding directory in prTempDir
		errCopyState := common.CopyFile(baseStatePath, path.Join(options.TerraformOptions.TerraformDir, "terraform.tfstate"))
		if errCopyState != nil {
			// Tear down the test
			options.testTearDown()
			return nil, fmt.Errorf("failed to copy state file: %v", errCopyState)
		} else {
			logger.Log(options.Testing, "State file copied to PR branch dir:", path.Join(options.TerraformOptions.TerraformDir, "terraform.tfstate"))
		}

		logger.Log(options.Testing, "Init / Plan on PR Branch:", prBranch)
		logger.Log(options.Testing, "Init / Plan on PR Branch dir:", options.TerraformOptions.TerraformDir)

		// Run Terraform plan in prTempDir
		result, resultErr = options.runTestPlan()

		if resultErr != nil {
			logger.Log(options.Testing, "Error during Terraform Plan on PR branch:", resultErr)
			assert.Nilf(options.Testing, resultErr, "Terraform Plan on PR branch has failed")

			// Tear down the test
			options.testTearDown()

			return nil, resultErr
		}

		logger.Log(options.Testing, "Parsing plan output to determine if any resources identified for destroy (PR branch)...")
		options.checkConsistency(result)

		// Check if optional upgrade support on PR Branch is needed
		if options.CheckApplyResultForUpgrade && !options.Testing.Failed() {
			logger.Log(options.Testing, "Validating Optional upgrade on Current Branch (PR):", prBranch)
			_, applyErr := terraform.ApplyE(options.Testing, options.TerraformOptions)
			if applyErr != nil {
				logger.Log(options.Testing, "Error during Terraform Apply on PR branch:", applyErr)
				assert.Nilf(options.Testing, applyErr, "Terraform Apply on PR branch has failed")

				// Tear down the test
				options.testTearDown()

				return nil, applyErr
			}
		}

		// Tear down the test
		options.testTearDown()
	}

	// let the calling test know if this upgrade was skipped or not
	options.UpgradeTestSkipped = skipped

	return result, resultErr
}

// RunTestConsistency Runs Test To check consistency between apply and re-apply, returns the output as string for further assertions
func (options *TestOptions) RunTestConsistency() (*terraform.PlanStruct, error) {
	defer func() {
		// Clear the plan file path so it is not used in the next test if testSetup is disabled
		if options.SkipTestSetup {
			options.TerraformOptions.PlanFilePath = ""
		}
	}()
	options.testSetup()

	logger.Log(options.Testing, "START: Init / Apply / Consistency Check")
	_, err := options.runTest()
	if err != nil {
		options.testTearDown()
		return nil, err
	}
	result, err := options.runTestPlan()
	if err != nil {
		options.testTearDown()
		return result, err
	}
	options.checkConsistency(result)
	logger.Log(options.Testing, "FINISHED: Init / Apply / Consistency Check")

	options.testTearDown()

	return result, err
}

// RunTestPlan Runs Test plan and returns the plan as a struct for assertions
func (options *TestOptions) RunTestPlan() (*terraform.PlanStruct, error) {
	options.testSetup()
	outputStruct, err := options.runTestPlan()
	options.testTearDown()

	return outputStruct, err
}

// runTestPlan Runs Test plan and returns the plan as a struct for assertions for internal use no setup or teardown
func (options *TestOptions) runTestPlan() (*terraform.PlanStruct, error) {

	logger.Log(options.Testing, "START: Init / Plan / Show w/Struct")

	// create a unique plan file name in terraform directory (which is already in temp location)
	tmpPlanFile, tmpPlanErr := os.CreateTemp(options.TerraformDir, "terratest-plan-file-")
	if tmpPlanErr != nil {
		return nil, tmpPlanErr
	}
	options.TerraformOptions.PlanFilePath = tmpPlanFile.Name()

	// TERRATEST uses its own internal logger.
	// The "show" command will produce a very large JSON to stdout which is printed by the logger.
	// We are temporarily turning the terratest logger OFF (discard) while running "show" to prevent large JSON stdout.
	options.TerraformOptions.Logger = logger.Discard
	outputStruct, err := terraform.InitAndPlanAndShowWithStructE(options.Testing, options.TerraformOptions)

	options.TerraformOptions.Logger = logger.Default // turn log back on

	assert.Nil(options.Testing, err, "Failed to create plan: ", err)
	logger.Log(options.Testing, "FINISHED: Init / Plan / Show w/Struct")

	return outputStruct, err
}

// RunTest Runs Test and returns the output as a string for assertions
func (options *TestOptions) RunTest() (string, error) {
	options.testSetup()
	output, err := options.runTest()
	options.testTearDown()

	return output, err
}

// runTest Runs Test and returns the output as a string for assertions for internal use no setup or teardown
func (options *TestOptions) runTest() (string, error) {

	logger.Log(options.Testing, "START: Init / Apply")
	output, err := terraform.InitAndApplyE(options.Testing, options.TerraformOptions)
	assert.Nil(options.Testing, err, "Failed", err)
	logger.Log(options.Testing, "FINISHED: Init / Apply")

	return output, err
}

// setTerraformDir helper funtion to set the terraform directory
// sets the TerraformOptions.TerraformDir, TestOptions.TerraformDir and TestOptions.WorkspacePath
func (options *TestOptions) setTerraformDir(tempDir string) {
	options.TerraformOptions.TerraformDir = tempDir
	options.TerraformDir = tempDir
	options.WorkspacePath = tempDir
}
