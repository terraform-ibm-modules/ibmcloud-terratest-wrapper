package cloudinfo

import (
	"fmt"

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

/*
TODO-1:
Add function: GetIngressState which returns (string, error)
Requires:
	REST API: https://containers.cloud.ibm.com/global/v2/alb/getStatus?cluster=<cluster_id>
	authenticator := infoSvc.authenticator

	Access token: cloud_info/service.go/CloudInfoService.authenticator
	other parameter (need to check): check for CloudInfoService reference
*/

// GetContainerStatus retrieves the ingress state of the cluster
// clusterId: the ID
// Returns the state of the cluster
func (infoSvc *CloudInfoService) GetClusterStatus(clusterId string) (struct{}, error) {

	accessToken, tokenErr := infoSvc.GetAccessToken()
	if tokenErr != nil {
		return struct, tokenErr
	}
	// TODO: Make a REST API call with return type as a Strcuture
	// 1. Discuss if a wrapper is available to make REST API call
	// 2. Otherwise find a method to make a REST API call in Go

}

// IsClusterIngressHealthy retrieves the ingress state of the cluster
// clusterId: the ID
// Returns the bool if ingress state is healthy or not
func (infoSvc *CloudInfoService) IsClusterIngressHealthy() bool {

}
