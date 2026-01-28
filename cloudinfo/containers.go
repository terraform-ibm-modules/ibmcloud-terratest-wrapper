package cloudinfo

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
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

// GetKubeVersions retrieves the available Kubernetes or OpenShift versions
// for a given platform and returns them as a slice of "major.minor" version strings.
//
// KubeVersions().ListV1 returns a map like:
// map[
//
//	kubernetes:[{1 31 14 false} {1 32 11 false} {1 33 7 true} {1 34 3 false}]
//	openshift:[{4 16 54 false} {4 17 45 false} {4 18 30 false} {4 19 21 true}]
//
// ]
// The function preprocesses this output to return only the versions
// corresponding to the platform passed to GetKubeVersions.
//
// The platform parameter must match a key returned by the API (e.g., "kubernetes" or "openshift").
// This works for both classic and VPC clusters.
func (infoSvc *CloudInfoService) GetKubeVersions(platform string) ([]string, error) {
	// Get the container V1 client from the service
	containerV1Client := infoSvc.containerV1Client

	// Fetch all available cluster versions (kubernetes and openShift) using the V1 API
	stableVersions, err := containerV1Client.KubeVersions().ListV1(containerv1.ClusterTargetHeader{})
	if err != nil {
		return nil, fmt.Errorf("error listing cluster versions: %w", err)
	}

	if len(stableVersions) == 0 {
		return nil, fmt.Errorf("no kubernetes or openShift versions returned")
	}

	// Get the versions for the requested platform (e.g., "kubernetes" or "openshift")
	platformVersions, ok := stableVersions[platform]
	if !ok || len(platformVersions) == 0 {
		return nil, fmt.Errorf("no versions available for platform: %s", platform)
	}

	// Convert each KubeVersion struct into a "major.minor" string e.g "4.16"
	versions := make([]string, 0, len(platformVersions))
	for _, v := range platformVersions {
		versions = append(versions, fmt.Sprintf("%d.%d", v.Major, v.Minor))
	}

	return versions, nil
}
