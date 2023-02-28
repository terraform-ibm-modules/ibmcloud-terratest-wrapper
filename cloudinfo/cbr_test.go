package cloudinfo

import (
	"encoding/json"
	"errors"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/stretchr/testify/assert"
	"testing"
)

const ruleJson = `{
    "contexts": [
        {
            "attributes": [
                {
                    "name": "endpointType",
                    "value": "private"
                },
                {
                    "name": "networkZoneId",
                    "value": "366d56825bd272e7e7ffe0299e74f22b"
                }
            ]
        }
    ],
    "created_at": "2023-02-27T17:11:51.000Z",
    "created_by_id": "IBMid-270006EYUC",
    "crn": "crn:v1:bluemix:public:context-based-restrictions:global:a/abac0df06b644a9cabc6e44f55b3880e::rule:ca1c2bb48b40ed7c595a6ff3ed71a15c",
    "description": "test-postgres-postgres access only from vpc",
    "enforcement_mode": "enabled",
    "href": "https://cbr.cloud.ibm.com/v1/rules/ca1c2bb48b40ed7c595a6ff3ed71a15c",
    "id": "ca1c2bb48b40ed7c595a6ff3ed71a15c",
    "last_modified_at": "2023-02-27T17:11:51.000Z",
    "last_modified_by_id": "IBMid-270006EYUC",
    "operations": {
        "api_types": [
            {
                "api_type_id": "crn:v1:bluemix:public:context-based-restrictions::::api-type:data-plane"
            }
        ]
    },
    "resources": [
        {
            "attributes": [
                {
                    "name": "accountId",
                    "operator": "stringEquals",
                    "value": "abac0df06b644a9cabc6e44f55b3880e"
                },
                {
                    "name": "serviceInstance",
                    "operator": "stringEquals",
                    "value": "c2ce344a-752c-4d9b-9374-75a0b664aecf"
                },
                {
                    "name": "serviceName",
                    "operator": "stringEquals",
                    "value": "databases-for-postgresql"
                }
            ]
        }
    ]
}`
const zoneJson = `{
  "account_id": "abac0df06b644a9cabc6e44f55b3880e",
  "address_count": 1,
  "addresses": [
    {
      "type": "vpc",
      "value": "crn:v1:bluemix:public:is:us-south:a/abac0df06b644a9cabc6e44f55b3880e::vpc:r006-746aefab-fb68-4aea-bd9f-6ad069a72288"
    }
  ],
  "created_at": "2023-02-27T16:22:43.000Z",
  "created_by_id": "IBMid-270006EYUC",
  "crn": "crn:v1:bluemix:public:context-based-restrictions:global:a/abac0df06b644a9cabc6e44f55b3880e::zone:366d56825bd272e7e7ffe0299e74f22b",
  "description": "CBR Network zone representing VPC",
  "excluded": [],
  "excluded_count": 0,
  "href": "https://cbr.cloud.ibm.com/v1/zones/366d56825bd272e7e7ffe0299e74f22b",
  "id": "366d56825bd272e7e7ffe0299e74f22b",
  "last_modified_at": "2023-02-27T16:22:43.000Z",
  "last_modified_by_id": "IBMid-270006EYUC",
  "name": "test-postgres-VPC-network-zone"
}`

// TestGetCBRRuleByID tests the GetCBRRuleByID function with a valid rule ID
func TestGetCBRRuleByID(t *testing.T) {

	ruleID := "ca1c2bb48b40ed7c595a6ff3ed71a15c"

	var mockRule = contextbasedrestrictionsv1.Rule{}

	err := json.Unmarshal([]byte(ruleJson), &mockRule)
	if err != nil {
		t.Fatalf("Failed to unmarshal %s", err)
	}
	mockResponse200 := core.DetailedResponse{StatusCode: 200}
	mockResponse404 := core.DetailedResponse{StatusCode: 404, RawResult: []byte("Mock failure")}

	t.Parallel()

	t.Run("cbr get rule success", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{rule: &mockRule, detailedResponse: &mockResponse200, err: nil}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		rule, err := infoSvc.GetCBRRuleByID(ruleID)
		assert.Nil(t, err)
		assert.NotNil(t, rule)
		assert.Equal(t, ruleID, *rule.ID)
	})

	t.Run("cbr get rule fail 404", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{rule: nil, detailedResponse: &mockResponse404, err: nil}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		rule, err := infoSvc.GetCBRRuleByID(ruleID)
		assert.Nil(t, rule)
		assert.Equal(t, errors.New("failed to get rule: Mock failure"), err)
	})
	t.Run("cbr get rule fail nil response", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{rule: nil, detailedResponse: nil, err: errors.New("some failure")}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		rule, err := infoSvc.GetCBRRuleByID(ruleID)
		assert.Nil(t, rule)
		assert.Equal(t, errors.New("some failure"), err)
	})

}

// TestGetCBRZoneByID tests the GetCBRZoneByID function with a valid zone ID
func TestGetCBRZoneByID(t *testing.T) {

	// Set a valid zone ID (replace with your own)
	zoneID := "366d56825bd272e7e7ffe0299e74f22b"
	var mockZone = contextbasedrestrictionsv1.Zone{}

	err := json.Unmarshal([]byte(zoneJson), &mockZone)
	if err != nil {
		t.Fatalf("Failed to unmarshal %s", err)
	}
	mockResponse200 := core.DetailedResponse{StatusCode: 200}
	mockResponse404 := core.DetailedResponse{StatusCode: 404, RawResult: []byte("Mock failure")}

	t.Parallel()

	t.Run("cbr get zone success", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{zone: &mockZone, detailedResponse: &mockResponse200, err: nil}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		zone, err := infoSvc.GetCBRZoneByID(zoneID)
		assert.Nil(t, err)
		assert.NotNil(t, zone)
		assert.Equal(t, zoneID, *zone.ID)
	})

	t.Run("cbr get zone fail 404", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{zone: nil, detailedResponse: &mockResponse404, err: nil}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		zone, err := infoSvc.GetCBRZoneByID(zoneID)
		assert.Nil(t, zone)
		assert.Equal(t, errors.New("failed to get zone: Mock failure"), err)
	})
	t.Run("cbr get zone fail nil response", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{zone: nil, detailedResponse: nil, err: errors.New("some failure")}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		rule, err := infoSvc.GetCBRZoneByID(zoneID)
		assert.Nil(t, rule)
		assert.Equal(t, errors.New("some failure"), err)
	})

}
