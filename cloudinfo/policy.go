package cloudinfo

import (
	"errors"
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

// DeleteIamPolicyByID will delete an IAM policy by ID
func (infoSvc *CloudInfoService) DeleteIamPolicyByID(policyId string) error {
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
		return errors.New(ErrPolicyInvalidToDelete)
	case 401:
		return errors.New(ErrTokenInvalid)
	case 403:
		return errors.New(ErrNoAccessToDeletePolicy)
	case 404:
		return errors.New(ErrPolicyNotFound)
	case 429:
		return errors.New(ErrTooManyRequests)
	default:
		return fmt.Errorf("unknown response code %d", response.StatusCode)
	}
}
