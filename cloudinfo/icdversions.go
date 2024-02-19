package cloudinfo

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/go-sdk-core/v5/core"
)

type Deployable struct {
	Type     string    `json:"type"`
	Versions []Version `json:"versions"`
}

type Version struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

type Data struct {
	Deployables []Deployable `json:"deployables"`
}

func (infoSvc *CloudInfoService) GetAvailableIcdVersions(icdType string) ([]string, error) {

	authenticator := &core.IamAuthenticator{
		ApiKey: infoSvc.authenticator.ApiKey,
	}
	newOptions := &clouddatabasesv5.CloudDatabasesV5Options{
		Authenticator: authenticator,
	}

	// Create the service client
	service, err := clouddatabasesv5.NewCloudDatabasesV5(newOptions)
	if err != nil {
		log.Fatalf("Failed to create Cloud Databases service client: %v", err)
	}

	// List deployables
	listDeployablesOptions := service.NewListDeployablesOptions() // Hypothetical method
	listDeployablesResponse, _, err := service.ListDeployables(listDeployablesOptions)
	if err != nil {
		panic(err)
	}
	response, _ := json.MarshalIndent(listDeployablesResponse, "", "  ")

	// Parse the response body
	jsonData := string(response)
	var data Data
	err2 := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		fmt.Println(err2)
		return nil, err
	}
	versions := []string{}
	for _, deployable := range data.Deployables {
		if deployable.Type == icdType {
			for _, version := range deployable.Versions {
				if version.Status == "stable" {
					versions = append(versions, version.Version)
				}
			}
		}
	}
	return versions, nil
}
