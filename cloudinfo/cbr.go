package cloudinfo

import (
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/contextbasedrestrictionsv1"
)

// GetCBRRuleByID gets a rule by its ID using the Context-based Restrictions service
func (infoSvc *CloudInfoService) GetCBRRuleByID(ruleID string) (*contextbasedrestrictionsv1.Rule, *core.DetailedResponse, error) {
	// Call the GetRule method with the rule ID and the context
	getRuleOptions := &contextbasedrestrictionsv1.GetRuleOptions{
		RuleID: &ruleID,
	}
	rule, detailedResponse, err := infoSvc.cbrService.GetRule(getRuleOptions)
	if err != nil {
		return nil, detailedResponse, err
	}

	// Check if the response status code is 200 (success)
	if detailedResponse.StatusCode == 200 {
		return rule, detailedResponse, nil
	}

	return nil, detailedResponse, fmt.Errorf("failed to get rule: %s", detailedResponse.RawResult)
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

// SetCBREnforcementMode sets the enforcement mode of a rule using the Context-based Restrictions service
func (infoSvc *CloudInfoService) SetCBREnforcementMode(ruleID string, mode string) error {
	if mode != "enabled" && mode != "report" && mode != "disabled" {
		return fmt.Errorf("invalid CBR enforcement mode: %s, valid options enabled, report, or disabled", mode)
	}

	existingRule, detailedResponse, rule_err := infoSvc.GetCBRRuleByID(ruleID)
	if rule_err != nil {
		return fmt.Errorf("failed to get rule: %s", rule_err)
	}
	// Extract the eTag from the DetailedResponse struct
	eTag := detailedResponse.GetHeaders().Get("eTag")

	existingRule.EnforcementMode = &mode

	// Call the ReplaceCBRRule method to update the rule with the modified version
	_, _, err := infoSvc.ReplaceCBRRule(existingRule, &eTag)
	if err != nil {
		return fmt.Errorf("failed to update rule: %s", err)
	}

	return nil
}
