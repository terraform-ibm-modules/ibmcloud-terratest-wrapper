package testhelper

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	for _, resource := range plan.ResourceChangesMap {
		resourceDetails := fmt.Sprintf("Name: %s Address: %s Actions: %s", resource.Name, resource.Address, resource.Change.Actions)
		var errorMessage string
		if !options.IgnoreDestroys.IsExemptedResource(resource.Address) {
			errorMessage = fmt.Sprintf("Resource(s) identified to be destroyed %s", resourceDetails)
			assert.False(options.Testing, resource.Change.Actions.Delete(), errorMessage)
			assert.False(options.Testing, resource.Change.Actions.DestroyBeforeCreate(), errorMessage)
			assert.False(options.Testing, resource.Change.Actions.CreateBeforeDestroy(), errorMessage)
		}
		if !options.IgnoreUpdates.IsExemptedResource(resource.Address) {
			errorMessage = fmt.Sprintf("Resource(s) identified to be updated %s", resourceDetails)
			assert.False(options.Testing, resource.Change.Actions.Update(), errorMessage)
		}
		// We only want to check pure Adds (creates without destroy) if the consistency test is
		// NOT the result of an Upgrade, as some adds are expected when doing the Upgrade test
		// (such as new resources were added as part of the pull request)
		if !options.IsUpgradeTest {
			if !options.IgnoreAdds.IsExemptedResource(resource.Address) {
				errorMessage = fmt.Sprintf("Resource(s) identified to be created %s", resourceDetails)
				assert.False(options.Testing, resource.Change.Actions.Create(), errorMessage)
			}
		}
	}
}

// testSetup Setup test
func (options *TestOptions) testSetup() {

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

	// Ensure always running from git root
	gitRoot, _ := GitRootPath(".")

	// To avoid workspace collisions when running in parallel, ignoring any temp terraform files
	// NOTE: if it is upgrade test we need hidden .git files
	tempDirFilter := func(path string) bool {
		if !options.IsUpgradeTest && files.PathContainsHiddenFileOrFolder(path) {
			return false
		}
		if files.PathContainsTerraformStateOrVars(path) {
			return false
		}
		return true
	}
	tempDir, tempDirErr := files.CopyFolderToTemp(gitRoot, options.Prefix, tempDirFilter)
	require.Nil(options.Testing, tempDirErr, "Error setting up TEMP directory")
	logger.Log(options.Testing, "TEMP TESTING DIR CREATED: ", tempDir)

	options.TerraformDir = fmt.Sprintf("%s/%s", tempDir, options.TerraformDir)
	options.baseTempWorkingDir = tempDir

	// update existing TerraformOptions with full path of new temp location
	options.TerraformOptions.TerraformDir = options.TerraformDir

	options.WorkspacePath = options.TerraformDir
	if options.UseTerraformWorkspace {
		// Always run in a new clean workspace to avoid reusing existing state files
		options.WorkspaceName = terraform.WorkspaceSelectOrNew(options.Testing, options.TerraformOptions, options.Prefix)
		options.WorkspacePath = fmt.Sprintf("%s/terraform.tfstate.d/%s", options.WorkspacePath, options.Prefix)
	}
}

// testTearDown Tear down test
func (options *TestOptions) testTearDown() {
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

		// remove the temp directory which is one level above the working directory
		tempDirParent := filepath.Dir(options.baseTempWorkingDir)
		logger.Log(options.Testing, "Deleting the temp working directory")
		os.RemoveAll(tempDirParent)
	}
}

// RunTestUpgrade Runs upgrade Test and asserts no resources have been destroyed in the upgrade, returns plan struct for further assertions
func (options *TestOptions) RunTestUpgrade() (*terraform.PlanStruct, error) {

	var result *terraform.PlanStruct
	var resultErr error

	skipped := true

	// Skip upgrade Test in continuous testing pipeline which runs in short mode
	if testing.Short() {
		options.Testing.Skip("Skipping upgrade Test in short mode.")
	}

	// Determine the name of the PR branch
	branchCmd := exec.Command("/bin/sh", "-c", "git rev-parse --abbrev-ref HEAD")
	branch, _ := branchCmd.CombinedOutput()

	if skipUpgradeTest(string(branch)) {
		options.Testing.Log("Detected the string \"BREAKING CHANGE\" or \"SKIP UPGRADE TEST\" used in commit message, skipping upgrade Test.")
	} else {
		skipped = false
		options.IsUpgradeTest = true

		// Setup the test including a TEMP directory to run in
		options.testSetup()

		// from here on we will operate in the temp directory
		gitRoot, _ := GitRootPath(options.TerraformDir)
		gitRepo, _ := git.PlainOpen(gitRoot)

		ref, _ := gitRepo.Head()
		prBranch := ref.Name()
		logger.Log(options.Testing, "Current Branch (PR): "+prBranch)

		// fetch to ensure all branches are present
		remote, err := gitRepo.Remote("origin")
		if err != nil {
			logger.Log(options.Testing, err)
		}

		opts := &git.FetchOptions{
			RefSpecs: []config.RefSpec{"refs/*:refs/*"},
		}

		if err := remote.Fetch(opts); err != nil {
			logger.Log(options.Testing, err)
		}

		var branches storer.ReferenceIter
		var defaultBranch plumbing.ReferenceName
		branches, resultErr = gitRepo.Branches()
		if resultErr == nil {
			logger.Log(options.Testing, "Branches: ")
			_ = branches.ForEach(func(ref *plumbing.Reference) error {
				logger.Log(options.Testing, ref.Name().String())
				if defaultBranch == "" {
					match, _ := regexp.MatchString("/[mM]ain|[mM]aster$", ref.Name().String())
					if match {
						defaultBranch = ref.Name()
					}
				}
				return nil
			})

		}

		logger.Log(options.Testing, "Default Branch: ", defaultBranch)

		w, _ := gitRepo.Worktree()
		logger.Log(options.Testing, "Attempting Git Checkout Default Branch: ", defaultBranch)
		resultErr = w.Checkout(&git.CheckoutOptions{
			Branch: defaultBranch,
			Force:  true})
		assert.Nilf(options.Testing, resultErr, "Could Not Checkout Default Branch")
		cur, _ := gitRepo.Head()
		logger.Log(options.Testing, "Current Branch (Default): "+cur.Name())

		_, resultErr = terraform.InitAndApplyE(options.Testing, options.TerraformOptions)
		assert.Nilf(options.Testing, resultErr, "Terraform Apply on MASTER branch has failed")

		// Only proceed to upgrade Test of master branch apply passed
		if resultErr == nil {
			logger.Log(options.Testing, "Attempting Git Checkout PR Branch: ", prBranch)
			resultErr = w.Checkout(&git.CheckoutOptions{
				Branch: prBranch,
				Force:  true})
			assert.Nilf(options.Testing, resultErr, "Could Not Checkout PR Branch")
			if resultErr == nil {
				cur, _ = gitRepo.Head()
				logger.Log(options.Testing, "Current Branch (PR): "+cur.Name())

				// Plan needs a temp file to store plan in
				tmpPlanFile, _ := os.CreateTemp(options.TerraformDir, "terratest-plan-file-")
				options.TerraformOptions.PlanFilePath = tmpPlanFile.Name()

				result, resultErr = terraform.InitAndPlanAndShowWithStructE(options.Testing, options.TerraformOptions)
				assert.Nilf(options.Testing, resultErr, "Terraform Plan on PR branch has failed")
				if result != nil && resultErr == nil {
					logger.Log(options.Testing, "Parsing plan output to determine if any resources identified for destroy (PR branch)..")
					options.checkConsistency(result)
				} else {
					// if there were issues running InitAndPlan, an Init needs to take place after branch change in order for downstream
					// terraform to work (like the destroy)
					terraform.Init(options.Testing, options.TerraformOptions)
				}
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

	planDir, dirErr := os.MkdirTemp("", "plan")
	if dirErr != nil {
		log.Fatal(dirErr)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			logger.Log(options.Testing, "Failed to remove path: ", err)
		}
	}(planDir) // clean up

	options.TerraformOptions.PlanFilePath = fmt.Sprintf("%splan-%s", planDir, options.Prefix)
	outputStruct, err := terraform.InitAndPlanAndShowWithStructE(options.Testing, options.TerraformOptions)
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
