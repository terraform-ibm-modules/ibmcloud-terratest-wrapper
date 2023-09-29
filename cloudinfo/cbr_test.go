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
		rule, _, err := infoSvc.GetCBRRuleByID(ruleID)
		assert.Nil(t, err)
		assert.NotNil(t, rule)
		assert.Equal(t, ruleID, *rule.ID)
	})

	t.Run("cbr get rule fail 404", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{rule: nil, detailedResponse: &mockResponse404, err: nil}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		rule, _, err := infoSvc.GetCBRRuleByID(ruleID)
		assert.Nil(t, rule)
		assert.Equal(t, errors.New("failed to get rule: Mock failure"), err)
	})
	t.Run("cbr get rule fail nil response", func(t *testing.T) {
		// Create a new instance of the mock
		mockCBR := &cbrServiceMock{rule: nil, detailedResponse: nil, err: errors.New("some failure")}
		infoSvc := CloudInfoService{cbrService: mockCBR}
		// Call the GetCBRRuleByID function with the service instance and the rule ID
		rule, _, err := infoSvc.GetCBRRuleByID(ruleID)
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

// TestSetCBREnforcementMode tests the SetCBREnforcementMode function.
func TestSetCBREnforcementMode(t *testing.T) {
	// Create a mock CBR service with an existing rule
	existingRule := &contextbasedrestrictionsv1.Rule{
		ID:              core.StringPtr("mock-rule-id"),
		EnforcementMode: core.StringPtr("disabled"),
	}
	eTag := "mock-etag"

	mockCBRService := &cbrServiceMock{
		rule:             existingRule,
		detailedResponse: &core.DetailedResponse{StatusCode: 200, Headers: map[string][]string{"eTag": []string{eTag}}},
		err:              nil,
	}

	// Create a CloudInfoService with the mock CBR service
	infoSvc := CloudInfoService{cbrService: mockCBRService}

	// Define test cases using subtests
	testCases := []struct {
		name           string
		mode           string
		expectedMode   string
		expectedErrStr string
	}{
		{
			name:           "Set to enabled",
			mode:           "enabled",
			expectedMode:   "enabled",
			expectedErrStr: "",
		},
		{
			name:           "Set to report",
			mode:           "report",
			expectedMode:   "report",
			expectedErrStr: "",
		},
		{
			name:           "Set to disabled",
			mode:           "disabled",
			expectedMode:   "disabled",
			expectedErrStr: "",
		},
		{
			name:           "Set to invalid mode",
			mode:           "invalid_mode",
			expectedMode:   "disabled", // No mode change expected
			expectedErrStr: "invalid CBR enforcement mode: invalid_mode, valid options enabled, report, or disabled",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			// Set the enforcement mode
			err := infoSvc.SetCBREnforcementMode("mock-rule-id", testCase.mode)

			if testCase.expectedErrStr == "" {
				assert.Nil(t, err)
			} else {
				if assert.NotNil(t, err, "eTag mismatch should have returned an error") {
					assert.Contains(t, err.Error(), testCase.expectedErrStr)
				}
			}

			assert.Equal(t, testCase.expectedMode, *existingRule.EnforcementMode)

		})
	}
}
