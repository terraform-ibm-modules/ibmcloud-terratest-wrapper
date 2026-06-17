package cloudinfo

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	transitgatewayapisv1 "github.com/IBM/networking-go-sdk/transitgatewayapisv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"gopkg.in/yaml.v3"
)

const (
	regionStatusAvailable      = "available"
	maxPowerConnectionsPerZone = 2
)

type GetTestRegionOptions struct {
	// exclude a region if it contains an Activity Tracker
	ExcludeActivityTrackerRegions bool
}

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

	// loop through regions for account looking for status 'available'
	for _, region := range regionCol.Regions {
		if region.Status != nil && *region.Status == regionStatusAvailable {
			availRegions = append(availRegions, region)
		}
	}
	log.Printf("Found %d available VPC regions", len(availRegions))

	return availRegions, nil
}

// GetLeastVpcTestRegion is a method for receiver CloudInfoService that will determine a region available
// to the caller account that currently contains the least amount of deployed VPCs, using default options.
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastVpcTestRegion() (string, error) {
	// Set up default
	options := NewGetTestRegionOptions()
	return infoSvc.GetLeastVpcTestRegionO(*options)
}

// GetLeastVpcTestRegionWithoutActivityTracker is a method for receiver CloudInfoService that will determine a
// region available to the caller account that currently contains the least amount of deployed VPCs and does
// not currently contain an active Activity Tracker instance (can only have one per region).
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastVpcTestRegionWithoutActivityTracker() (string, error) {
	// get default options
	options := NewGetTestRegionOptions()
	// change activity tracker setting
	options.ExcludeActivityTrackerRegions = true

	return infoSvc.GetLeastVpcTestRegionO(*options)
}

// GetLeastVpcTestRegionO is a method for receiver CloudInfoService that will determine a region available
// to the caller account that currently contains the least amount of deployed VPCs.
// The determination can be influenced by specifying CloudInfoService.regionsData and supplying appropriate options.
// If no CloudInfoService.regionsData exists, it will simply loop through all available regions for the caller account
// and choose a region with lowest VPC count.
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastVpcTestRegionO(options GetTestRegionOptions) (string, error) {

	var bestregion RegionData

	regions, err := infoSvc.GetTestRegionsByPriority()
	if err != nil {
		return "", err
	}

	// if we need to filter out regions by activity tracker existence, prepare a list of those regions
	// NOTE: we only want to do this once at beginning and then use results below
	var atInstanceList []resourcecontrollerv2.ResourceInstance
	var atListErr error
	if options.ExcludeActivityTrackerRegions {
		atInstanceList, atListErr = infoSvc.ListResourcesByCrnServiceName("logdnaat")
		if atListErr != nil {
			log.Println("WARNING: Error retrieving Activity Tracker instances! Ignoring when selecting.")
			atInstanceList = []resourcecontrollerv2.ResourceInstance{}
		}
	}

	for _, region := range regions {
		// if option is set, ignore region if there is existing activity tracker
		if options.ExcludeActivityTrackerRegions {
			if regionHasActivityTracker(region.Name, atInstanceList) {
				log.Println("Region", region.Name, "skipped due to Activity Tracker present")
				continue // ignore and move to next region
			}
		}

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

		// region list is sorted by priority, so if vpc count is zero then short circuit and return, it is the best region
		if region.ResourceCount == 0 {
			bestregion = region
			log.Printf("Selected region %s (0 VPCs)", bestregion.Name)
			break
		} else if len(bestregion.Name) == 0 {
			bestregion = region // always use first valid region in list
		} else if region.ResourceCount < bestregion.ResourceCount {
			bestregion = region // use if lower count
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

	log.Printf("Selected region %s with %d VPCs", bestregion.Name, bestregion.ResourceCount)
	return bestregion.Name, nil
}

// GetLeastSdnlbTestRegion is a method for receiver CloudInfoService that will determine a region available
// to the caller account that currently contains the least amount of deployed SDN load balancers.
// This helps avoid hitting quota limits when deploying resources like PAG that require load balancers.
// If no region can be determined, returns the provided defaultRegion.
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastSdnlbTestRegion(defaultRegion string) (string, error) {
	// get default options
	options := NewGetTestRegionOptions()

	return infoSvc.GetLeastSdnlbTestRegionO(defaultRegion, *options)
}

// GetLeastSdnlbTestRegionO is a method for receiver CloudInfoService that will determine a region available
// to the caller account that currently contains the least amount of deployed SDN load balancers.
// The determination can be influenced by specifying CloudInfoService.regionsData and supplying appropriate options.
// If no CloudInfoService.regionsData exists, it will simply loop through all available regions for the caller account
// and choose a region with lowest SDN load balancer count.
// If no region can be determined, returns the provided defaultRegion.
// Returns a string representing an IBM Cloud region name, and error.
func (infoSvc *CloudInfoService) GetLeastSdnlbTestRegionO(defaultRegion string, options GetTestRegionOptions) (string, error) {
	var bestregion RegionData

	regions, err := infoSvc.GetTestRegionsByPriority()
	if err != nil {
		return "", err
	}

	// if we need to filter out regions by activity tracker existence, prepare a list of those regions
	// NOTE: we only want to do this once at beginning and then use results below
	var atInstanceList []resourcecontrollerv2.ResourceInstance
	var atListErr error
	if options.ExcludeActivityTrackerRegions {
		atInstanceList, atListErr = infoSvc.ListResourcesByCrnServiceName("logdnaat")
		if atListErr != nil {
			log.Println("WARNING: Error retrieving Activity Tracker instances! Ignoring when selecting.")
			atInstanceList = []resourcecontrollerv2.ResourceInstance{}
		}
	}

	for _, region := range regions {
		// if option is set, ignore region if there is existing activity tracker
		if options.ExcludeActivityTrackerRegions {
			if regionHasActivityTracker(region.Name, atInstanceList) {
				continue // ignore and move to next region
			}
		}

		setErr := infoSvc.vpcService.SetServiceURL(region.Endpoint)
		if setErr != nil {
			return "", setErr
		}

		lbCol, detailedResponse, err := infoSvc.vpcService.ListLoadBalancers(&vpcv1.ListLoadBalancersOptions{})
		if err != nil {
			log.Println("Failed LIST Load Balancers for region", region.Name, ":", err, "Full Response:", detailedResponse)
			return "", err
		}
		region.ResourceCount = int(*lbCol.TotalCount)

		// region list is sorted by priority, so if load balancer count is zero then short circuit and return, it is the best region
		if region.ResourceCount == 0 {
			bestregion = region
			break
		} else if len(bestregion.Name) == 0 {
			bestregion = region // always use first valid region in list
		} else if region.ResourceCount < bestregion.ResourceCount {
			bestregion = region // use if lower count
		}
	}

	// after this is done need to set serviceURL back to default
	defer func() {
		if resetErr := infoSvc.vpcService.SetServiceURL(vpcv1.DefaultServiceURL); resetErr != nil {
			log.Println("Warning: failed to reset service URL to default:", resetErr)
		}
	}()

	// if return val is still empty, then there were no regions available, return default region
	if len(bestregion.Name) == 0 {
		return defaultRegion, nil
	}

	return bestregion.Name, nil
}

// GetRegionWithoutService finds a region with ZERO instances of the specified service.
func (infoSvc *CloudInfoService) GetRegionWithoutService(serviceName string) (string, error) {
	log.Printf("Searching for regions without '%s' instances...", serviceName)

	// Get all instances of this service using Resource Controller
	instances, err := infoSvc.ListResourcesByCrnServiceName(serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to list '%s' instances: %w", serviceName, err)
	}

	log.Printf("Found %d '%s' instances total", len(instances), serviceName)

	// Build set of regions that have this service
	occupiedRegions := make(map[string]bool)
	for _, instance := range instances {
		if instance.RegionID != nil && *instance.RegionID != "" {
			occupiedRegions[*instance.RegionID] = true
			instanceName := "<unnamed>"
			if instance.Name != nil {
				instanceName = *instance.Name
			}
			log.Printf("Found '%s' instance '%s' in region: %s", serviceName, instanceName, *instance.RegionID)
		}
	}

	log.Printf("Total regions with '%s': %d", serviceName, len(occupiedRegions))

	// Get priority-ordered available regions
	regions, err := infoSvc.GetTestRegionsByPriority()
	if err != nil {
		return "", fmt.Errorf("failed to get test regions: %w", err)
	}

	// Return first priority region WITHOUT this service
	for _, region := range regions {
		if !occupiedRegions[region.Name] {
			log.Printf("✓ Selected region %s (no '%s' instances)", region.Name, serviceName)
			return region.Name, nil
		}
	}

	return "", fmt.Errorf("no region available without '%s' - all test regions have instances", serviceName)
}

// GetRegionWithLeastResources finds the region with the MINIMUM number of instances for the specified service.
func (infoSvc *CloudInfoService) GetRegionWithLeastResources(serviceName string) (string, error) {
	log.Printf("Searching for region with least '%s' instances...", serviceName)

	// Get all instances of this service using Resource Controller
	instances, err := infoSvc.ListResourcesByCrnServiceName(serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to list '%s' instances: %w", serviceName, err)
	}

	log.Printf("Found %d '%s' instances total", len(instances), serviceName)

	// Count instances per region
	regionCounts := make(map[string]int)
	for _, instance := range instances {
		if instance.RegionID != nil && *instance.RegionID != "" {
			regionCounts[*instance.RegionID]++
		}
	}

	// Get priority-ordered available regions
	regions, err := infoSvc.GetTestRegionsByPriority()
	if err != nil {
		return "", fmt.Errorf("failed to get test regions: %w", err)
	}

	// Find region with lowest count
	var bestRegion string
	minCount := math.MaxInt

	for _, region := range regions {
		count := regionCounts[region.Name]

		if count < minCount {
			minCount = count
			bestRegion = region.Name
		}

		// Short-circuit if we find empty region (optimal case)
		if count == 0 {
			log.Printf("✓ Selected region %s (zero '%s' instances)", region.Name, serviceName)
			return region.Name, nil
		}
	}

	if bestRegion == "" {
		return "", fmt.Errorf("no suitable region found for '%s'", serviceName)
	}

	log.Printf("✓ Selected region %s (least '%s' instances: %d)", bestRegion, serviceName, minCount)
	return bestRegion, nil
}

// GetRegionWithoutWatsonXGovernance
func (infoSvc *CloudInfoService) GetRegionWithoutWatsonXGovernance() (string, error) {
	return infoSvc.GetRegionWithoutService("aiopenscale")
}

// GetRegionWithLeastTransitGateways returns the region with the minimum number of transit gateways.
func (infoSvc *CloudInfoService) GetRegionWithLeastTransitGateways() (string, error) {
	// Get all transit gateways using Transit Gateway SDK
	listOptions := &transitgatewayapisv1.ListTransitGatewaysOptions{}
	result, _, err := infoSvc.transitGatewayService.ListTransitGateways(listOptions)
	if err != nil {
		return "", fmt.Errorf("failed to list transit gateways: %w", err)
	}

	// Count transit gateways per location (region)
	regionCounts := make(map[string]int)
	for _, gateway := range result.TransitGateways {
		// Check both Location and Name for nil
		if gateway.Location != nil && *gateway.Location != "" {
			if gateway.Name != nil {
				regionCounts[*gateway.Location]++
			}
		}
	}

	// Get priority-ordered available regions
	regions, err := infoSvc.GetTestRegionsByPriority()
	if err != nil {
		return "", fmt.Errorf("failed to get test regions: %w", err)
	}

	// Find region with lowest count
	var bestRegion string
	minCount := math.MaxInt

	for _, region := range regions {
		count := regionCounts[region.Name]

		if count < minCount {
			minCount = count
			bestRegion = region.Name
		}

		// if we find empty region
		if count == 0 {
			log.Printf("Selected region %s with %d transit gateways", region.Name, count)
			return region.Name, nil
		}
	}

	if bestRegion == "" {
		return "", fmt.Errorf("no suitable region found for transit gateways")
	}

	log.Printf("Selected region %s with %d transit gateways", bestRegion, minCount)
	return bestRegion, nil
}

// regionHasActivityTracker is a helper function to determine if a given region is represented in an array
// of existing ActivityTracker resource instances.
// Returns boolean true if region found
func regionHasActivityTracker(region string, activityTrackerList []resourcecontrollerv2.ResourceInstance) bool {

	// don't bother looping if empty
	if len(activityTrackerList) == 0 {
		return false
	}

	for _, at := range activityTrackerList {
		if at.RegionID != nil && *at.RegionID == region {
			return true
		}
	}

	return false
}

// GetTestRegionsByPriority is a method for receiver CloudInfoService that will use the service regionsData
// to determine a priority order and region eligibility for test resources to be deployed.
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
			if regiondetail.Status != nil && *regiondetail.Status == regionStatusAvailable {
				if regiondetail.Endpoint != nil {
					testregion.Endpoint = *regiondetail.Endpoint + "/v1"
				}
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
		if vpcRegion.Name != nil {
			newRegion := RegionData{
				Name:         *vpcRegion.Name,
				UseForTest:   true,
				TestPriority: 100, // making larger value, in case we need to add regions prioitezed before
			}
			regionsData = append(regionsData, newRegion)
		}
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
		return "", errors.New("no available zones were supplied for power systems")
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

		if region.UseForTest {
			connCount := countPowerConnectionsInZone(region.Name, connections)
			region.ResourceCount = connCount

			// region list is sorted by priority, so if resource count is zero then short circuit and return, it is the best region
			// NOTE: we will also make sure each region is not at total limit of connections, if it is we will move on to next
			if region.ResourceCount == 0 {
				bestregion = region
				log.Printf("Selected Power zone %s (0 connections)", bestregion.Name)
				break
			} else if region.ResourceCount < maxPowerConnectionsPerZone && len(bestregion.Name) == 0 {
				bestregion = region // always use first VALID region found in list
			} else if region.ResourceCount < maxPowerConnectionsPerZone && region.ResourceCount < bestregion.ResourceCount {
				bestregion = region // use if valid AND lower count than previous best
			}
		}
	}

	// if return val is still empty, then there were no regions available, send error
	if len(bestregion.Name) == 0 {
		return "", errors.New("no region could be determined")
	}

	log.Printf("Selected Power zone %s with %d connections", bestregion.Name, bestregion.ResourceCount)
	return bestregion.Name, nil
}

// HasRegionData is a method for receiver CloudInfoService that will respond with a boolean to verify that the service instance
// has region data loaded. You can use this method to determine if you need to load preference data.
func (infoSvc *CloudInfoService) HasRegionData() bool {
	if len(infoSvc.regionsData) > 0 {
		return true
	} else {
		return false
	}
}

// RemoveRegionForTest is a method for receiver CloudInfoService  that will remove a given region for use in test considerations
// by setting the UseForTest property for the region to false
func (infoSvc *CloudInfoService) RemoveRegionForTest(regionID string) {
	// loop through region data looking for given region
	for i, regionData := range infoSvc.regionsData {
		if regionData.Name == regionID {
			infoSvc.regionsData[i].UseForTest = false
			break
		}
	}
}

// countPowerConnectionsInZone is a private helper function that will return a count of occurrences of
// the provided zone in a list of existing Powercloud connections.
func countPowerConnectionsInZone(zone string, connections []*PowerCloudConnectionDetail) int {
	count := 0

	for _, conn := range connections {
		if conn.Zone != nil && *conn.Zone == zone {
			count += 1
		}
	}

	return count
}

// NewGetTestRegionOptions will return the option struct with defaults
func NewGetTestRegionOptions() *GetTestRegionOptions {
	return &GetTestRegionOptions{
		ExcludeActivityTrackerRegions: false,
	}
}
