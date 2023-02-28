package cloudinfo

import (
	"context"
	"fmt"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
	"time"
)

// GetCBRRuleByID gets a rule by its ID using the Context-based Restrictions service
func (infoSvc *CloudInfoService) GetCBRRuleByID(ruleID string) (*contextbasedrestrictionsv1.Rule, error) {
	// Create a context with a timeout of 10 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Call the GetRule method with the rule ID and the context
	getRuleOptions := &contextbasedrestrictionsv1.GetRuleOptions{
		RuleID: &ruleID,
	}
	rule, response, err := infoSvc.cbrService.GetRuleWithContext(ctx, getRuleOptions)
	if err != nil {
		return nil, err
	}

	// Check if the response status code is 200 (success)
	if response.StatusCode == 200 {
		return rule, nil
	}

	return nil, fmt.Errorf("failed to get rule: %s", response.RawResult)
}

// GetCBRZoneByID gets a zone by its ID using the Context-based Restrictions service
func (infoSvc *CloudInfoService) GetCBRZoneByID(zoneID string) (*contextbasedrestrictionsv1.Zone, error) {
	// Create a context with a timeout of 10 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Call the GetZone method with the zone ID and the context
	getZoneOptions := &contextbasedrestrictionsv1.GetZoneOptions{
		ZoneID: &zoneID,
	}
	zone, response, err := infoSvc.cbrService.GetZoneWithContext(ctx, getZoneOptions)
	if err != nil {
		return nil, err
	}

	// Check if the response status code is 200 (success)
	if response.StatusCode == 200 {
		return zone, nil
	}

	return nil, fmt.Errorf("failed to get zone: %s", response.RawResult)
}
