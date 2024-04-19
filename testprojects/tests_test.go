package testprojects

//func TestProjectsFullTest(t *testing.T) {
//
//	options := TestProjectOptionsDefault(&TestProjectsOptions{
//		Testing:                t,
//		StackConfigurationOrder: []string{
//			"primary-da",
//			"secondary-da",
//		},
//	})
//	options.StackInputs = map[string]map[string]interface{}{
//		"primary-da": {
//			"prefix": fmt.Sprintf("primary%s", options.Prefix),
//		},
//		"secondary-da": {
//			"prefix": fmt.Sprintf("secondary%s", options.Prefix),
//		},
//	}
//	err := options.RunProjectsTest()
//	if assert.NoError(t, err) {
//		t.Log("TestProjectsFullTest Passed")
//	} else {
//		t.Error("TestProjectsFullTest Failed")
//	}
//}
