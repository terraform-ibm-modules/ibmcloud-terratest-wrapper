package cloudinfo

import (
	"fmt"
	"log"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv2"
)

// GetClusterConfigConfigPath retrieves the path to the cluster's Admin configuration file
// Config for the current API keys user, uses public endpoint, and does not create a Calico configuration file
// clusterId: the ID or name of the cluster
// Returns the path to the configuration file
func (infoSvc *CloudInfoService) GetClusterConfigConfigPath(clusterId string) (string, error) {
	return infoSvc.GetClusterConfigPath(clusterId, ".", false, false, "public")
}

// GetClusterAdminConfigPath retrieves the path to the cluster's Admin configuration file
// Uses public endpoint, and does not create a Calico configuration file
// clusterId: the ID or name of the cluster
// Returns the path to the configuration file
func (infoSvc *CloudInfoService) GetClusterAdminConfigPath(clusterId string) (string, error) {
	return infoSvc.GetClusterConfigPath(clusterId, ".", true, false, "public")
}

// GetClusterConfigPathWithEndpoint retrieves the path to the cluster's configuration file
// Config for the current API keys user, and does not create a Calico configuration file
// clusterId: the ID or name of the cluster
// endpoint: the endpoint type to use
func (infoSvc *CloudInfoService) GetClusterConfigPathWithEndpoint(clusterId string, endpoint string) (string, error) {
	return infoSvc.GetClusterConfigPath(clusterId, ".", false, false, endpoint)
}

// GetClusterConfigPath retrieves the path to the cluster's configuration file
// clusterId: the ID or name of the cluster
// basePath: the base directory path where the config file will be stored
// admin: whether to retrieve admin config
// createCalicoConfig: whether to create a Calico configuration file
// endpoint: the endpoint type to use
// Returns the path to the configuration file
func (infoSvc *CloudInfoService) GetClusterConfigPath(clusterId string, basePath string, admin bool, createCalicoConfig bool, endpoint string) (string, error) {

	containerClient := infoSvc.containerClient

	_, configDetails, err := containerClient.Clusters().StoreConfigDetail(clusterId, basePath, admin, createCalicoConfig, containerv2.ClusterTargetHeader{}, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster config details: %v", err)
	}

	return configDetails.FilePath, nil
}

// GetAlbInfo retrieves the details of an ALB
// albId: the ID
// Returns the detailed information for an ALB in a cluster.
func (infoSvc *CloudInfoService) GetAlbInfo(albId string) (status string, err error) {
	// some logic
	albConfig, detailedResponse, err := infoSvc.albService.GetClusterALB(infoSvc.albService.NewGetClusterALBOptions(albId))
	if err != nil {
		log.Println("Failed to egt Cluster ALB details for ", albId, ":", err, "Full Response:", detailedResponse)
		return "", err
	}

	// If any specific operation to perform for a state(healthy, critical, pending) is requried.
	/*	if *albConfig.State == "healthy" {
		} else if *albConfig.State == "critical" {
		} else {
		}
	*/
	return *albConfig.State, nil
}
