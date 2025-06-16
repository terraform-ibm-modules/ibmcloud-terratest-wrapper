package testaddons

import (
	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

// BuildActuallyDeployedResult contains the results of building the actually deployed list
type BuildActuallyDeployedResult struct {
	ActuallyDeployedList []cloudinfo.OfferingReferenceDetail
	Warnings             []string
	Errors               []string
}

// ValidationResult contains the results of dependency validation
type ValidationResult struct {
	IsValid           bool
	DependencyErrors  []cloudinfo.DependencyError
	UnexpectedConfigs []cloudinfo.OfferingReferenceDetail
	MissingConfigs    []cloudinfo.OfferingReferenceDetail
	Messages          []string
}

// DependencyGraphResult contains the results of building a dependency graph
type DependencyGraphResult struct {
	Graph                map[string][]cloudinfo.OfferingReferenceDetail // Using string key for offering identity
	ExpectedDeployedList []cloudinfo.OfferingReferenceDetail
	Visited              map[string]bool
}
