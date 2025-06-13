package main

import (
	"fmt"
	"os"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// Test script to demonstrate the enhanced ref-resolver debugging
func main() {
	fmt.Println("=== Testing Enhanced Ref-Resolver Debug Logging ===")

	// Enable verbose debugging
	os.Setenv("TERRATEST_VERBOSE_REF_RESOLVER_DEBUG", "true")

	// Test the verbose debug check
	enabled := cloudinfo.IsVerboseRefResolverDebugEnabled()
	fmt.Printf("Verbose debug enabled: %t\n", enabled)

	// Disable verbose debugging
	os.Unsetenv("TERRATEST_VERBOSE_REF_RESOLVER_DEBUG")
	enabled = cloudinfo.IsVerboseRefResolverDebugEnabled()
	fmt.Printf("Verbose debug after unset: %t\n", enabled)

	// Test with different values
	os.Setenv("TERRATEST_VERBOSE_REF_RESOLVER_DEBUG", "false")
	enabled = cloudinfo.IsVerboseRefResolverDebugEnabled()
	fmt.Printf("Verbose debug with 'false': %t\n", enabled)

	os.Setenv("TERRATEST_VERBOSE_REF_RESOLVER_DEBUG", "TRUE")
	enabled = cloudinfo.IsVerboseRefResolverDebugEnabled()
	fmt.Printf("Verbose debug with 'TRUE': %t\n", enabled)

	fmt.Println("\n=== Debug Test Complete ===")
	fmt.Println("To enable detailed ref-resolver logging in your tests:")
	fmt.Println("export TERRATEST_VERBOSE_REF_RESOLVER_DEBUG=true")
	fmt.Println("\nThis will show detailed logging for:")
	fmt.Println("- API requests and responses")
	fmt.Println("- Authentication token details (masked)")
	fmt.Println("- Retry logic and failure reasons")
	fmt.Println("- Project resolution details")
	fmt.Println("- Reference transformation steps")
}
