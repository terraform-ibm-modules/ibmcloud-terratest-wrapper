package testprojects

//func TestProjectsFullTest(t *testing.T) {
//
//	cloudInfoSvc, cloudInfoErr := cloudinfo.NewCloudInfoServiceFromEnv("TF_VAR_ibmcloud_api_key", cloudinfo.CloudInfoServiceOptions{})
//	if !assert.NoError(t, cloudInfoErr) {
//		t.Error("TestProjectsFullTest Failed")
//		return
//	}
//	options := TestProjectOptionsDefault(&TestProjectsOptions{
//		Testing:                t,
//		StackConfigurationOrder: []string{
//			"primary-da",
//			"secondary-da",
//		},
//	})
//
//	options.StackInputs = map[string]interface{}{
//		"resource_group_name": "Default",
//		"prefix":              strings.TrimLeft(options.Prefix, "-"),
//		"ibmcloud_api_key":    cloudInfoSvc.ApiKey,
//	}
//
//	err := options.RunProjectsTest()
//	if assert.NoError(t, err) {
//		t.Log("TestProjectsFullTest Passed")
//	} else {
//		t.Error("TestProjectsFullTest Failed")
//	}
//}
