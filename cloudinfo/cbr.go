package cloudinfo

import (
	"fmt"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
)

// GetCBRRuleByID gets a rule by its ID using the Context-based Restrictions service
func (infoSvc *CloudInfoService) GetCBRRuleByID(ruleID string) (*contextbasedrestrictionsv1.Rule, error) {
	// Call the GetRule method with the rule ID and the context
	getRuleOptions := &contextbasedrestrictionsv1.GetRuleOptions{
		RuleID: &ruleID,
	}
	rule, response, err := infoSvc.cbrService.GetRule(getRuleOptions)
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
	// Call the GetZone method with the zone ID and the context
	getZoneOptions := &contextbasedrestrictionsv1.GetZoneOptions{
		ZoneID: &zoneID,
	}
	zone, response, err := infoSvc.cbrService.GetZone(getZoneOptions)
	if err != nil {
		return nil, err
	}

	// Check if the response status code is 200 (success)
	if response.StatusCode == 200 {
		return zone, nil
	}

	return nil, fmt.Errorf("failed to get zone: %s", response.RawResult)
}
