package cloudinfo

import (
	"github.com/IBM/go-sdk-core/v5/core"
	"testing"

	"github.com/IBM/platform-services-go-sdk/iampolicymanagementv1"
	"github.com/stretchr/testify/assert"
)

func TestDeletePolicyByID(t *testing.T) {
	testCases := []struct {
		name        string
		setupMock   func(mock *iamPolicyServiceMock)
		expectedErr string
	}{
		{
			name: "DeletePolicySuccess",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 204}, nil)
			},
			expectedErr: "",
		},
		{
			name: "DeletePolicyInvalid",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 400}, nil)
			},
			expectedErr: ErrPolicyInvalidToDelete,
		},
		{
			name: "DeletePolicyTokenInvalid",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 401}, nil)
			},
			expectedErr: ErrTokenInvalid,
		},
		{
			name: "DeletePolicyNoAccess",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 403}, nil)
			},
			expectedErr: ErrNoAccessToDeletePolicy,
		},
		{
			name: "DeletePolicyNotFound",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 404}, nil)
			},
			expectedErr: ErrPolicyNotFound,
		},
		{
			name: "DeletePolicyTooManyRequests",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 429}, nil)
			},
			expectedErr: ErrTooManyRequests,
		},
		{
			name: "DeletePolicyUnknownResponseCode",
			setupMock: func(mock *iamPolicyServiceMock) {
				policyID := "mock-policy-id"
				mock.On("DeletePolicy", &iampolicymanagementv1.DeletePolicyOptions{PolicyID: &policyID}).Return(&core.DetailedResponse{StatusCode: 999}, nil)
			},
			expectedErr: "unknown response code 999",
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &iamPolicyServiceMock{}
			infoSvc := CloudInfoService{
				iamPolicyService: mock,
			}

			// Setup mock
			tc.setupMock(mock)

			// Execute function under test
			err := infoSvc.DeletePolicyByID("mock-policy-id")

			// Assertions
			if tc.expectedErr != "" {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			// Assert that the DeletePolicy function was called with the correct arguments
			mock.AssertExpectations(t)
		})
	}
}
