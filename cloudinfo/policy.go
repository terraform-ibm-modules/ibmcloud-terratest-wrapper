package cloudinfo

import (
	"fmt"
	"github.com/IBM/platform-services-go-sdk/iampolicymanagementv1"
)

const (
	ErrPolicyInvalidToDelete  = "policy was not valid to delete"
	ErrTokenInvalid           = "the token provided is not valid"
	ErrNoAccessToDeletePolicy = "access to delete the policy is not granted"
	ErrPolicyNotFound         = "policy was not found"
	ErrTooManyRequests        = "too many requests have been made within a given time window"
)

// DeletePolicyByID will delete an IAM policy by ID
func (infoSvc *CloudInfoService) DeletePolicyByID(policyId string) error {
	response, err := infoSvc.iamPolicyService.DeletePolicy(&iampolicymanagementv1.DeletePolicyOptions{
		PolicyID: &policyId,
	})

	if err != nil {
		return err
	}

	switch response.StatusCode {
	case 204:
		return nil
	case 400:
		return fmt.Errorf(ErrPolicyInvalidToDelete)
	case 401:
		return fmt.Errorf(ErrTokenInvalid)
	case 403:
		return fmt.Errorf(ErrNoAccessToDeletePolicy)
	case 404:
		return fmt.Errorf(ErrPolicyNotFound)
	case 429:
		return fmt.Errorf(ErrTooManyRequests)
	default:
		return fmt.Errorf("unknown response code %d", response.StatusCode)
	}
}
