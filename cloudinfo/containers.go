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

// GetAlbState retrieves the details of State corresponding to ALB ID
// albId: the ID of the ALB
// Returns the State for an ALB ID.
func (infoSvc *CloudInfoService) GetAlbState(albId string) (status string, err error) {
	albConfig, detailedResponse, err := infoSvc.albService.GetClusterALB(infoSvc.albService.NewGetClusterALBOptions(albId))
	if err != nil {
		log.Println("Failed to get Cluster ALB details for ", albId, ":", err, "Full Response:", detailedResponse)
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

// GetAlbIds retrieves the list of all ALBs in a cluster
// clusterId: the ID or name of the cluster
// Returns a list all ALB IDs in a cluster. If no ALB IDs are returned, then the cluster does not have a portable subnet.
func (infoSvc *CloudInfoService) GetAlbIds(clusterId string) (ids []string, err error) {
	clusterAlbs, detailedResponse, err := infoSvc.albService.GetClusterALBs(infoSvc.albService.NewGetClusterALBsOptions(clusterId))
	if err != nil {
		log.Println("Failed to get ALB IDs for ", clusterId, ":", err, "Full Response:", detailedResponse)
		return []string{}, err
	}

	// don't bother looping if empty
	if len(clusterAlbs) == 0 {
		return
	}

	// loop through clusterAlbs to get albIds
	for _, clusterAlb := range clusterAlbs {
		ids = append(ids, *clusterAlb.ID)
	}
	return ids, nil
}

// GetIngressState retrieves the state of each albIds in a cluster
// clusterId: the ID or name of the cluster
// Returns a map with albId as key and corresponding state as value
func (infoSvc *CloudInfoService) GetIngressState(clusterId string) (state map[string]string, err error) {
	albIds, err := infoSvc.GetAlbIds(clusterId)
	if err != nil {
		return state, fmt.Errorf("failed to get ALB IDS for cluster: %v", clusterId)
	}

	// don't bother looping if empty
	if len(albIds) == 0 {
		return
	}

	for _, albId := range albIds {
		albState, err := infoSvc.GetAlbState(albId)
		if err != nil {
			state[albId] = ""
		}
		state[albId] = albState
	}
	return state, nil
}
