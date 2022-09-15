package cloudinfo

import (
	"errors"
	"log"
	"os"
	"sort"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	"gopkg.in/yaml.v3"
)

const (
	regionStatusAvailable      = "available"
	maxPowerConnectionsPerZone = 2
)

// GetAvailableVpcRegions is a method for receiver CloudInfoService that will query the caller account
// for all regions that support the VPC resource type and are available in the account.
// Returns an array of vpcv1.Region and error.
func (infoSvc *CloudInfoService) GetAvailableVpcRegions() ([]vpcv1.Region, error) {

	// list of regions with status 'available'
	var availRegions []vpcv1.Region

	// Retrieve the list of regions for your account.
	regionCol, detailedResponse, err := infoSvc.vpcService.ListRegions(&vpcv1.ListRegionsOptions{})
	if err != nil {
		log.Println("Failed LIST REGIONS:", err, "Full Response:", detailedResponse)
		return nil, err
	}
	log.Println("Number of Regions available: ", len(regionCol.Regions))
	log.Println("*** REGIONS AVAILABLE ***")

	// loop through regions for account looking for status 'available'
	for _, region := range regionCol.Regions {
		log.Println(*region.Name, *region.Status, *region.Endpoint, *region.Href)
		if *region.Status == regionStatusAvailable {
			availRegions = append(availRegions, region)
		}
	}

	return availRegions, nil
}

// GetLeastVpcTestRegion is a method for receiver CloudInfoService that will determine a region available
// to the caller account that currently contains the least amount of deployed VPCs.
// The determination can be influenced by specifying CloudInfoService.regionsData.
// If no CloudInfoService.regionsData exists, it will simply loop through all available regions for the caller account
// and choose a region with lowest VPC count.
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastVpcTestRegion() (string, error) {

	var bestregion RegionData

	regions, err := infoSvc.GetTestRegionsByPriority()
	if err != nil {
		return "", err
	}

	for i, region := range regions {
		setErr := infoSvc.vpcService.SetServiceURL(region.Endpoint)
		if setErr != nil {
			log.Println("Failed to set a service url in vpc service")
			return "", err
		}

		vpcCol, detailedResponse, err := infoSvc.vpcService.ListVpcs(&vpcv1.ListVpcsOptions{})
		if err != nil {
			log.Println("Failed LIST VPCs for region", region.Name, ":", err, "Full Response:", detailedResponse)
			return "", err
		}
		region.ResourceCount = int(*vpcCol.TotalCount)
		log.Println("Region", region.Name, "VPC count:", region.ResourceCount)

		// region list is sorted by priority, so if vpc count is zero then short circuit and return, it is the best region
		if region.ResourceCount == 0 {
			bestregion = region
			log.Println("--- new best region is", bestregion.Name)
			break
		} else if i == 0 {
			bestregion = region // always use first region in list
			log.Println("--- new best region is", bestregion.Name)
		} else if region.ResourceCount < bestregion.ResourceCount {
			bestregion = region // use if lower count
			log.Println("--- new best region is", bestregion.Name)
		}
	}

	// after this is done need to set serviceURL back to default
	defer func() {
		err = infoSvc.vpcService.SetServiceURL(vpcv1.DefaultServiceURL)
	}()

	// if return val is still empty, then there were no regions available, send error
	if len(bestregion.Name) == 0 {
		return "", errors.New("ERROR: No region could be determined")
	}

	return bestregion.Name, nil
}

// GetTestRegionsByPriority is a method for receiver CloudInfoService that will use the service regionsData
// to determine a priorty order and region eligibility for test resources to be deployed.
// The returned array will then be used by various methods to determine best region to use for different test scenarios.
// Returns an array of RegionData struct, and error.
func (infoSvc *CloudInfoService) GetTestRegionsByPriority() ([]RegionData, error) {

	var regions []RegionData

	// check if there was region data supplied by caller
	if len(infoSvc.regionsData) == 0 {
		// if caller did not supply custom region priority list, query all avail regions
		// for account and assume all same priority
		vpcLoadErr := infoSvc.LoadRegionsFromVpcAccount()
		if vpcLoadErr != nil {
			log.Println("Failed loading regions from cloud account")
			return nil, vpcLoadErr
		}
	}

	// filter out regions not used for testing or that are unavailable
	for _, testregion := range infoSvc.regionsData {
		if testregion.UseForTest {
			regiondetail, detailedResponse, err := infoSvc.vpcService.GetRegion(infoSvc.vpcService.NewGetRegionOptions(testregion.Name))
			if err != nil {
				log.Println("Failed GET DETAILS for region", testregion.Name, ":", err, "Full Response:", detailedResponse)
				return nil, err
			}
			if *regiondetail.Status == regionStatusAvailable {
				testregion.Endpoint = *regiondetail.Endpoint + "/v1"
				regions = append(regions, testregion)
			}
		}
	}

	// sort by priority ascending
	sort.Sort(SortedRegionsDataByPriority(regions))

	return regions, nil
}

// LoadRegionPrefsFromFile is a method for receiver CloudInfoService that will populate the CloudInfoService.regionsData
// by reading a file in the YAML format.
// Returns error.
func (infoSvc *CloudInfoService) LoadRegionPrefsFromFile(filePath string) error {
	data, readErr := os.ReadFile(filePath)
	if readErr != nil {
		log.Println("ERROR reading", filePath, ":", readErr)
		return readErr
	}

	var regionsData []RegionData

	err := yaml.Unmarshal(data, &regionsData)
	if err != nil {
		log.Println("ERROR unmarshalling", filePath, ":", err)
		return err
	}

	infoSvc.regionsData = regionsData

	return nil
}

// LoadRegionsFromVpcAccount is a method for receiver CloudInfoService that will populate the CloudInfoService.regionsData
// by using an API call to retrieve all available regions for the caller account, and assuming all are for test use and same priority.
// Returns error.
func (infoSvc *CloudInfoService) LoadRegionsFromVpcAccount() error {
	var regionsData []RegionData

	availRegions, regionErr := infoSvc.GetAvailableVpcRegions()
	if regionErr != nil {
		log.Println("ERROR: could not load available regions")
		return regionErr
	}

	for _, vpcRegion := range availRegions {
		newRegion := RegionData{
			Name:         *vpcRegion.Name,
			UseForTest:   true,
			TestPriority: 100, // making larger value, in case we need to add regions prioitezed before
		}
		regionsData = append(regionsData, newRegion)
	}

	infoSvc.regionsData = regionsData

	return nil
}

// GetLeastPowerConnectionZone is a method for receiver CloudInfoService that will determine a zone (data center) available
// to the caller account that currently contains the least amount of deployed PowerCloud connections.
// This determination requires specifying CloudInfoService.regionsData with valid data centers (regions)
// that are supported by the PowerCloud service.
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastPowerConnectionZone() (string, error) {

	var bestregion RegionData

	// sort available regions/zones by priority
	// for powercloud resources the available zone list needs to be supplied, otherwise error
	if len(infoSvc.regionsData) == 0 {
		return "", errors.New("no available zones were supplied for powercloud")
	}

	regions := infoSvc.regionsData
	// sort by priority ascending
	sort.Sort(SortedRegionsDataByPriority(regions))

	// load existing powercloud connections and their datacenter for the account
	connections, connErr := infoSvc.ListPowerConnectionsForAccount()
	if connErr != nil {
		return "", connErr
	}

	for _, region := range regions {

		connCount := countPowerConnectionsInZone(region.Name, connections)
		region.ResourceCount = connCount
		log.Println("Region", region.Name, "Resource count:", region.ResourceCount)

		// region list is sorted by priority, so if resource count is zero then short circuit and return, it is the best region
		// NOTE: we will also make sure each region is not at total limit of connections, if it is we will move on to next
		if region.ResourceCount == 0 {
			bestregion = region
			log.Println("--- new best region is", bestregion.Name)
			break
		} else if region.ResourceCount < maxPowerConnectionsPerZone && len(bestregion.Name) == 0 {
			bestregion = region // always use first VALID region found in list
			log.Println("--- new best region is", bestregion.Name)
		} else if region.ResourceCount < maxPowerConnectionsPerZone && region.ResourceCount < bestregion.ResourceCount {
			bestregion = region // use if valid AND lower count than previous best
			log.Println("--- new best region is", bestregion.Name)
		}
	}

	// if return val is still empty, then there were no regions available, send error
	if len(bestregion.Name) == 0 {
		return "", errors.New("no region could be determined")
	}

	return bestregion.Name, nil
}

// countPowerConnectionsInZone is a private helper function that will return a count of occurances of
// the provided zone in a list of existing Powercloud connections.
func countPowerConnectionsInZone(zone string, connections []*PowerCloudConnectionDetail) int {
	count := 0

	for _, conn := range connections {
		if *conn.Zone == zone {
			count += 1
		}
	}

	return count
}
