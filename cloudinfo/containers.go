package cloudinfo

import (
	"fmt"
	"time"

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

// GetClusterIngressStatus retrieves ingress status of the cluster
// clusterId: the ID or name of the cluster
// Returns the ingress status of the cluster
func (infoSvc *CloudInfoService) GetClusterIngressStatus(clusterId string) (string, error) {

	containerClient := infoSvc.containerClient
	ingressDetails, err := containerClient.Albs().GetIngressStatus(clusterId, containerv2.ClusterTargetHeader{})
	if err != nil {
		return "", fmt.Errorf("failed to get cluster ingress status: %v", err)
	}
	return ingressDetails.Status, nil
}

// 
func (infoSvc *CloudInfoService) CheckClusterIngressHealthy(clusterId string, clusterCheckTimeoutMinutes int, clusterCheckDelayMinutes int, logf func(string)) bool {
	startTime := time.Now()
	healthy := false
	for {
		ingressStatus, err := infoSvc.GetClusterIngressStatus(clusterId)
		if ingressStatus == "healthy" {
			healthy = true
			break
		} else if ingressStatus == "critical" || err != nil {
			if time.Since(startTime) > time.Duration(clusterCheckTimeoutMinutes)*time.Minute {
				break
			}
			logf("Cluster Ingress is critical, retrying after delay...")
			time.Sleep(time.Duration(clusterCheckDelayMinutes) * time.Minute)
		}
	}
	if !healthy {
		logf("Cluster Ingress failed to become healthy")
	}
	return healthy
}

func (infoSvc *CloudInfoService) CheckClusterIngressHealthyDefaultTimeout(clusterId string, logf func(string)){
	infoSvc.CheckClusterIngressHealthy(clusterId, 10, 1, logf)
}
