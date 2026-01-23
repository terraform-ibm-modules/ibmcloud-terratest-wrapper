package cloudinfo

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
)

// GetKubeVersions retrieves the available Kubernetes or OpenShift versions
// from the IBM Kubernetes Service (IKS) and returns them as a slice of
// "major.minor" version strings for the requested platform.
//
// The platform parameter must match a key returned by the IKS API (e.g., "kubernetes" or "openshift").
func (infoSvc *CloudInfoService) GetKubeVersions(platform string) ([]string, error) {
	// Get the container V1 client from the service
	containerV1Client := infoSvc.containerV1Client

	// Fetch Kubernetes and OpenShift versions using the V1 API
	iksVersions, err := containerV1Client.KubeVersions().ListV1(containerv1.ClusterTargetHeader{})
	if err != nil {
		return nil, fmt.Errorf("error listing cluster versions: %w", err)
	}

	if len(iksVersions) == 0 {
		return nil, fmt.Errorf("no kube versions returned")
	}

	// Look up versions for the requested platform
	versions, ok := iksVersions[platform]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("no versions available for platform: %s", platform)
	}

	// Convert version structs into "major.minor" string format
	validVersions := make([]string, 0, len(versions))
	for _, v := range versions {
		validVersions = append(validVersions, fmt.Sprintf("%d.%d", v.Major, v.Minor))
	}

	return validVersions, nil
}
