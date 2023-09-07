package testhelper

import (
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gruntwork-io/terratest/modules/files"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/common"
)

func skipUpgradeTest(branch string) bool {
	// Get all the commit messages from the PR branch
	// NOTE: using the "origin" of the default branch as the start point, which will exist in a fresh
	// clone even if the default branch has not been checked out or pulled.
	cmd := exec.Command("/bin/sh", "-c", "git log origin/master..", branch)
	out, _ := cmd.CombinedOutput()

	// Skip upgrade Test if BREAKING CHANGE OR SKIP UPGRADE TEST string found in commit messages
	doNotRunUpgradeTest := false
	if strings.Contains(string(out), "BREAKING CHANGE") || strings.Contains(string(out), "SKIP UPGRADE TEST") {
		doNotRunUpgradeTest = true
	}
	if !doNotRunUpgradeTest {
		// NOTE: using the "origin" of the default branch as the start point, which will exist in a fresh
		// clone even if the default branch has not been checked out or pulled.
		cmd = exec.Command("/bin/sh", "-c", "git log origin/main..", branch)
		out, _ = cmd.CombinedOutput()

		if strings.Contains(string(out), "BREAKING CHANGE") || strings.Contains(string(out), "SKIP UPGRADE TEST") {
			doNotRunUpgradeTest = true
		}
	}
	return doNotRunUpgradeTest
}

// checkConsistency Fails the test if any destroys are detected and the resource is not exempt.
// If any addresses are provided in IgnoreUpdates.List then fail on updates too unless the resource is exempt
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

		// Run plan again to output the nice human-readable plan
		//terraform.Plan(options.Testing, options.TerraformOptions)

		var resourceDetails string

		if resource.Change.Actions.Update() {
			resourceDetails = fmt.Sprintf("Name: %s Address: %s Actions: %s\nDIFF:\n%s\n\nChange Detail:\n%s", resource.Name, resource.Address, resource.Change.Actions, common.GetBeforeAfterDiff(changesJson), changesJson)
		} else {
			// Do not print changesJson because might expose secrets
			resourceDetails = fmt.Sprintf("Name: %s Address: %s Actions: %s\n", resource.Name, resource.Address, resource.Change.Actions)
		}
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
			gitRoot, _ := common.GitRootPath(".")

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
					if files.PathContainsTerraformStateOrVars(path) {
						return false
					}
					// Add a filter to ignore directories named "temp"
					if strings.Contains(strings.ToLower(path), "/temp/") || strings.HasSuffix(strings.ToLower(path), "/temp") {
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
				options.TerraformOptions.TerraformDir = path.Join(dstDir, options.TerraformDir)
				options.TerraformDir = options.TerraformOptions.TerraformDir
				options.baseTempWorkingDir = tempDir
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
	if !options.SkipTestTearDown {
		// Check if "DO_NOT_DESTROY_ON_FAILURE" is set
		envVal, _ := os.LookupEnv("DO_NOT_DESTROY_ON_FAILURE")

		// Do not destroy if tests failed and "DO_NOT_DESTROY_ON_FAILURE" is true
		if options.Testing.Failed() && strings.ToLower(envVal) == "true" {
			fmt.Println("Terratest failed. Debug the Test and delete resources manually.")
		} else {

			for _, address := range options.ImplicitDestroy {
				statefile := fmt.Sprintf("%s/terraform.tfstate", options.WorkspacePath)
				out, err := RemoveFromStateFile(statefile, address)
				if options.ImplicitRequired && err != nil {
					logger.Log(options.Testing, out)
					assert.Nil(options.Testing, err, "Could not remove from state file")
				} else {
					logger.Log(options.Testing, out)
				}
			}

			logger.Log(options.Testing, "START: Destroy")
			terraform.Destroy(options.Testing, options.TerraformOptions)
			if options.UseTerraformWorkspace {
				terraform.WorkspaceDelete(options.Testing, options.TerraformOptions, options.Prefix)
			}
			logger.Log(options.Testing, "END: Destroy")

			if options.baseTempWorkingDir != "" {
				logger.Log(options.Testing, "Deleting the temp working directory")
				os.RemoveAll(options.baseTempWorkingDir)
			}
		}
	} else {
		logger.Log(options.Testing, "Skipping automatic Test Teardown")
	}
}

// RunTestUpgrade runs the upgrade test to ensure that the Terraform configurations being tested
// do not result in any resources being destroyed during an upgrade. This is crucial to ensure that
// existing infrastructure remains intact during updates.
//
// The function performs the following steps:
// 1. Checks if the test is running in short mode and skips the upgrade test if so.
// 2. Determines the current PR branch.
// 3. Checks if the upgrade test should be skipped based on commit messages.
// 4. If not skipped:
//    a. Sets up the test environment.
//    b. Copies the current code (from PR branch) to a temporary directory.
//    c. Clones the base branch into a separate temporary directory.
//    d. Applies Terraform configurations on the base branch.
//    e. Moves the state file from the base branch directory to the PR branch directory.
//    f. Runs Terraform plan in the PR branch directory to check for any inconsistencies.
//    g. Optionally, it can also apply the Terraform configurations on the PR branch.
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

	if skipUpgradeTest(prBranch) {
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
		options.DisableTempWorkingDir = true

		// Defer a function to restore the original value
		defer func() {
			options.DisableTempWorkingDir = tempDirCreationBackup
		}()

		// Setup the test including a TEMP directory to run in
		options.testSetup()

		// Create a temporary directory for the PR code
		prTempDir, err := os.MkdirTemp("", fmt.Sprintf("terraform-pr-%s", options.Prefix))
		if err != nil {
			logger.Log(options.Testing, err)
		} else {
			logger.Log(options.Testing, "TEMP PR DIR CREATED: ", prTempDir)
		}
		defer os.RemoveAll(prTempDir) // clean up

		// Create a temporary directory for the base branch
		baseTempDir, err := os.MkdirTemp("", fmt.Sprintf("terraform-base-%s", options.Prefix))
		if err != nil {
			logger.Log(options.Testing, err)
		} else {
			logger.Log(options.Testing, "TEMP UPGRADE BASE DIR CREATED: ", baseTempDir)
		}
		defer os.RemoveAll(baseTempDir) // clean up

		// Copy the current code (from PR branch) to the PR temp directory with the filter
		errCopy := common.CopyDirectory(gitRoot, prTempDir, func(path string) bool {
			if files.PathContainsTerraformStateOrVars(path) {
				return false
			}
			// Add a filter to ignore directories named "temp"
			if strings.Contains(strings.ToLower(path), "/temp/") || strings.HasSuffix(strings.ToLower(path), "/temp") {
				return false
			}
			return true
		})
		if errCopy != nil {
			// Tear down the test
			options.testTearDown()
			return nil, fmt.Errorf("failed to copy PR directory to temp: %v", errCopy)
		} else {
			logger.Log(options.Testing, "Copied current code to PR branch dir:", prTempDir)
		}

		// TODO: Remove before merge
		logger.Log(options.Testing, "Files in PR branch temp dir")
		printFiles(options.Testing, prTempDir)

		// TODO: This is not working in GitHub Actions
		// checkout action might need to be modified
		// Another thought is to check the GHA environment variables for the details
		baseRepo, baseBranch := common.GetBaseRepoAndBranch(options.BaseTerraformRepo, options.BaseTerraformBranch)
		if baseBranch == "" || baseRepo == "" {
			return nil, fmt.Errorf("failed to get default repo and branch: %s %s", baseRepo, baseBranch)
		} else {
			logger.Log(options.Testing, "Base Repo:", baseRepo)
			logger.Log(options.Testing, "Base Branch:", baseBranch)
		}

		authMethod, err := common.DetermineAuthMethod(baseRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to determine authentication method: %v", err)
		}

		// Try to clone with authentication first
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
		// TODO: Remove before merge
		logger.Log(options.Testing, "Files in default branch temp dir after clone")
		printFiles(options.Testing, baseTempDir)

		// Set TerraformDir to the appropriate directory within baseTempDir
		options.TerraformOptions.TerraformDir = path.Join(baseTempDir, relativeTestSampleDir)
		options.TerraformDir = options.TerraformOptions.TerraformDir
		logger.Log(options.Testing, "Init / Apply on Base repo:", baseRepo)
		logger.Log(options.Testing, "Init / Apply on Base branch:", baseBranch)
		logger.Log(options.Testing, "Init / Apply on Base branch dir:", options.TerraformOptions.TerraformDir)

		// TODO: Remove before merge
		printFiles(options.Testing, options.TerraformOptions.TerraformDir)
		_, resultErr = terraform.InitAndApplyE(options.Testing, options.TerraformOptions)
		assert.Nilf(options.Testing, resultErr, "Terraform Apply on Base branch has failed")

		// Get the path to the state file in baseTempDir
		baseStatePath := path.Join(options.TerraformOptions.TerraformDir, "terraform.tfstate")

		// Set TerraformDir to the appropriate directory within prTempDir
		options.TerraformOptions.TerraformDir = path.Join(prTempDir, relativeTestSampleDir)
		options.TerraformDir = options.TerraformOptions.TerraformDir
		// TODO: Note this worked for Tekton

		// ensure terraform working files/folders are removed before copying state file ie .terraform, .terraform.lock.hcl, terraform.tfstate, terraform.tfstate.backup
		CleanTerraformDir(options.TerraformOptions.TerraformDir)

		logger.Log(options.Testing, "Files before copying state file to PR branch dir")
		// TODO: Remove before merge
		printFiles(options.Testing, options.TerraformOptions.TerraformDir)

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

		// TODO: Remove before merge
		printFiles(options.Testing, options.TerraformOptions.TerraformDir)

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

func printFiles(t *testing.T, dir string) {
	// TODO: Debug details do not merge
	// print files in terraform dir with permisions and details including hidden files
	cmd := exec.Command("/bin/sh", "-c", "ls -la")
	cmd.Dir = dir
	fileDetails, err := cmd.CombinedOutput()
	if err != nil {
		logger.Log(t, "Error during ls -la  in ", dir, "\n", err)
	} else {
		logger.Log(t, "ls -laR in ", dir, ":\n", string(fileDetails))
	}

	// TODO: Debug details do not merge
}

// RunTestConsistency Runs Test To check consistency between apply and re-apply, returns the output as string for further assertions
func (options *TestOptions) RunTestConsistency() (*terraform.PlanStruct, error) {
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
	// TODO: Clear plan file from test options
	// Add description why we are doing this
	options.TerraformOptions.PlanFilePath = ""
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
