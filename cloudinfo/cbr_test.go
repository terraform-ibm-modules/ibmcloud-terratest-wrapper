package cloudinfo

import (
	"errors"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetCBRRuleByID tests the GetCBRRuleByID function with a valid rule ID
func TestGetCBRRuleByID(t *testing.T) {

	ruleID := "ca1c2bb48b40ed7c595a6ff3ed71a15c"
	description := "my-rule-description"
	mockRule := &contextbasedrestrictionsv1.Rule{
		ID:          &ruleID,
		Description: &description,
	}

	mockResponse200 := core.DetailedResponse{StatusCode: 200}
	mockResponse404 := core.DetailedResponse{StatusCode: 404, RawResult: []byte("Mock failure")}

	t.Parallel()

	t.Run("cbr get rule success", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{rule: mockRule, detailedResponse: &mockResponse200, err: nil}
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

	zoneID := "366d56825bd272e7e7ffe0299e74f22b"
	description := "my-zone-description"
	mockZone := &contextbasedrestrictionsv1.Zone{
		ID:          &zoneID,
		Description: &description,
	}

	mockResponse200 := core.DetailedResponse{StatusCode: 200}
	mockResponse404 := core.DetailedResponse{StatusCode: 404, RawResult: []byte("Mock failure")}

	t.Parallel()

	t.Run("cbr get zone success", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{zone: mockZone, detailedResponse: &mockResponse200, err: nil}
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
