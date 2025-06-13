package testprojects

import (
	"encoding/json"
	"fmt"

	//	Import stack struct form cloudinfo
	"log"
	"os"

	"github.com/terraform-ibm-modules/ibmcloud-terratest-wrapper/cloudinfo"
)

func GetVersionLocatorFromStackDefinitionForMemberName(pathToStackDefinition string, memberName string) (string, error) {
	// load the stack definition
	// Read the config JSON file
	jsonFile, err := os.ReadFile(pathToStackDefinition)
	if err != nil {
		log.Println("Error reading config JSON file:", err)
		return "", err
	}

	// Create a new variable of type Struct
	var config cloudinfo.Stack

	// Unmarshal the JSON data into the config variable
	err = json.Unmarshal(jsonFile, &config)
	if err != nil {
		log.Println("Error unmarshalling JSON:", err)
		return "", err
	}
	// find the member with the name
	for _, member := range config.Members {
		if member.Name == memberName {
			return member.VersionLocator, nil
		}
	}
	return "", fmt.Errorf("member not found")
}
