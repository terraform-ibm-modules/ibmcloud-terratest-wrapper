package testsetup

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// given a list of services and list of available regions, pare down the region list by looking for
// region limits for services in the DB
func (svc *TestSetupService) determineAllowedRegionList(serviceList []string, regions []string) ([]string, error) {
	// start with provided regions, if empty (no specific restrictions) then start with service default list from yaml
	var allowedRegions []string
	if len(regions) > 0 {
		allowedRegions = regions
	} else {
		allowedRegions = svc.TestRegions
	}

	// loop through services to see if there are limitations
	for _, svcName := range serviceList {
		limit, limitErr := svc.GetServiceRegionLimitation(svcName)
		if limitErr != nil {
			return []string{}, fmt.Errorf("error getting service limitation record from DB: %w", limitErr)
		}
		// if limit was nil, means there was no limitation
		if limit != nil {

		} else {
			if isDebugSet() {
				fmt.Printf("TEST SETUP DEBUG: no limitations were set for service %s\n", svcName)
			}
		}
	}

	return allowedRegions, nil
}

// LoadRegionSelectorData will load up the memdb with data taken from CloudInfoSvc (all currently deployed services),
// and also service limits from a test config file (YAML).
func (svc *TestSetupService) LoadRegionSelectorData() error {

	// create DB if needed
	if svc.ResourceDB == nil {
		createErr := svc.CreateResourceDB()
		if createErr != nil {
			return fmt.Errorf("error creating memdb: %w", createErr)
		}
	}

	// load cloud resources
	loadResourceErr := svc.loadAccountResourcesData()
	if loadResourceErr != nil {
		return fmt.Errorf("error loading resource data: %w", loadResourceErr)
	}

	// load region limits for services into DB
	limitsErr := svc.LoadTestConfigurationFromFile(svc.getTestConfigFileLocation())
	if limitsErr != nil {
		return limitsErr
	}
	for _, limitRecord := range svc.ServiceRegionLimits {
		insertErr := svc.insertCloudServiceLimit(&limitRecord)
		if insertErr != nil {
			return insertErr
		}
	}

	svc.dataLoaded = true

	if isDebugSet() {
		svc.PrintResourceTableData()
		svc.PrintLimitsTableData()
	}

	return nil
}

// Get all resources currently active in account, load them into DB table
// grouped by region with a count.
func (svc *TestSetupService) loadAccountResourcesData() error {

	// get a cloud info service and retrieve all account resources
	cloudInfoSvc, cloudInfoSvcErr := svc.GetCloudInfoService()
	if cloudInfoSvcErr != nil {
		return cloudInfoSvcErr
	}

	resourceList, resourceInfoErr := cloudInfoSvc.ListAllAccountResources()
	if resourceInfoErr != nil {
		return fmt.Errorf("error getting account resource list: %w", resourceInfoErr)
	}

	for _, resourceInfo := range resourceList {
		// create a new record struct to insert
		// NOTE: if this returns a nil it means we should skip this record
		record, recordErr := newCloudResourceRecord(resourceInfo)
		if recordErr != nil {
			return fmt.Errorf("error reading resource info record: %w", recordErr)
		}

		// add record if not nil (skipped)
		if record != nil {
			updateErr := svc.updateResourceData(record)
			if updateErr != nil {
				return updateErr
			}
		}
	}

	return nil
}

// LoadTestConfigurationFromFile will load configuration settings used for the test.
// The YAML file provided is split up into seperate documents. These documents contain
// the following data:
// 1. General Test Settings
// 2. Service Limitations for Region Selection
// Returns error if parsing YAML fails
func (svc *TestSetupService) LoadTestConfigurationFromFile(filePath string) error {
	reader, readErr := os.Open(filePath)
	if readErr != nil {
		return fmt.Errorf("error reading region prefs file at %s: %w", filePath, readErr)
	}
	defer reader.Close()

	decoder := yaml.NewDecoder(reader)

	var regionDoc yaml.Node
	var limitDoc yaml.Node

	// get first doc
	regionDocErr := decoder.Decode(&regionDoc)
	if regionDocErr != nil {
		return fmt.Errorf("error reading all region doc of yaml: %w", regionDocErr)
	}

	// get second doc
	limitDocErr := decoder.Decode(&limitDoc)
	if limitDocErr != nil {
		return fmt.Errorf("error reading all service limits doc of yaml: %w", limitDocErr)
	}

	// decode the docs
	var regionSettings TestRegionSettings
	decodeAllRegionsErr := regionDoc.Decode(&regionSettings)
	if decodeAllRegionsErr != nil {
		return fmt.Errorf("error decoding test regions of yaml: %w", decodeAllRegionsErr)
	}
	// store regions in service if needed
	if len(svc.TestRegions) == 0 {
		svc.TestRegions = regionSettings.AllowedTestRegions
	}

	// decode the service limits
	decodeLimitsErr := limitDoc.Decode(&svc.ServiceRegionLimits)
	if decodeLimitsErr != nil {
		return fmt.Errorf("error decoding service limits of yaml: %w", decodeLimitsErr)
	}

	return nil
}

/*
	func (svc *TestSetupService) LoadTestConfigurationFromFile(filePath string) error {
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return fmt.Errorf("error reading region prefs file at %s: %w", filePath, readErr)
		}

		var limitsData []CloudServiceLimit

		err := yaml.Unmarshal(data, &limitsData)
		if err != nil {
			return fmt.Errorf("error unmarshalling yaml file at %s: %w", filePath, err)
		}

		// insert limits records
		if limitsData != nil {
			for _, limitRecord := range limitsData {
				insertErr := svc.insertCloudServiceLimit(&limitRecord)
				if insertErr != nil {
					return insertErr
				}
			}
		}

		return nil
	}
*/
