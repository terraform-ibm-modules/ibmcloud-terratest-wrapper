package testhelper

import "os"

// EnvironmentVariables Holds a list of environment variables and their values
// When SetEnvironmentVariables is called it will save any existing environment variables in OldVariables and set NewVariables on the environment
// When ResetEnvironmentVariables is called it will set the environment variables back to the old values
type EnvironmentVariables struct {
	NewVariables map[string]string
	OldVariables map[string]string
}

func (environment EnvironmentVariables) SetEnvironmentVariables() {
	environment.OldVariables = make(map[string]string)
	for key, value := range environment.NewVariables {
		oldValue, found := os.LookupEnv(key)
		if found {
			environment.OldVariables[key] = oldValue
		} else {
			environment.OldVariables[key] = "?!UNSET_ME!?"
		}
		err := os.Setenv(key, value)
		if err != nil {
			print("Error setting Variable ", key)
		}
	}
}

func (environment EnvironmentVariables) ResetEnvironmentVariables() {
	for key, value := range environment.OldVariables {
		if value == "?!UNSET_ME!?" {
			err := os.Unsetenv(key)
			if err != nil {
				print("Error unsetting Variable ", key)
			}
		} else {
			err := os.Setenv(key, value)
			if err != nil {
				print("Error setting Variable ", key)
			}
		}
	}
}
