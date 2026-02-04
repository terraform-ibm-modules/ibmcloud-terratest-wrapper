package cloudinfo

import (
	"fmt"
	"time"

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

// CheckClusterIngressHealthyDefaultTimeout checks the ingress status of the specified cluster using default clusterCheckTimeoutMinutes and clusterCheckDelayMinutes values of 10 minutes and a delay of 1 minute between status checks respectively.
// This method is a convenience wrapper around the `CheckClusterIngressHealthy` method.
// Parameters:
// - clusterId: The ID or name of the cluster whose ingress status is to be checked.
// - logf: A logging function to report status updates.
func (infoSvc *CloudInfoService) CheckClusterIngressHealthyDefaultTimeout(clusterId string, logf func(...any)) bool {
	return infoSvc.CheckClusterIngressHealthy(clusterId, 10, 1, logf)
}

// CheckClusterIngressHealthy checks the ingress status of the specified cluster and asserts that it becomes healthy within a specified timeout period.
// This method performs the following steps:
// 1. Continuously checks the ingress status of the cluster identified by `clusterId`.
// 2. If the ingress status is "healthy", the method sets the result as healthy and exits the loop.
// 3. If the ingress status is "critical" or an error occurs, the method retries the check after a delay, continuing until either the status becomes "healthy" or the specified timeout is reached.
// 4. If the timeout is reached and the status is still "critical" or an error persists, the method exits the loop.
// Parameters:
// - clusterId: The ID or name of the cluster whose ingress status is to be checked.
// - clusterCheckTimeoutMinutes: The maximum time allowed for checking the ingress status, in minutes.
// - clusterCheckDelayMinutes: The duration to wait between status checks, in minutes.
// - logf: A logging function to report status updates.
// Returns:
// A bool value indicating if cluster is healthy or not.
func (infoSvc *CloudInfoService) CheckClusterIngressHealthy(clusterId string, clusterCheckTimeoutMinutes int, clusterCheckDelayMinutes int, logf func(...any)) bool {
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
	} else {
		logf("Cluster Ingress is healthy")
	}
	return healthy
