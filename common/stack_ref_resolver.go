package common

import (
	"fmt"
	project "github.com/IBM/project-go-sdk/projectv1"
	"reflect"
	"strings"
)

// Ref represents a reference with its name, resolved value, and flags
type Ref struct {
	Name          string
	Ref           *string
	ResolvedValue *string
	isRef         bool
	Resolved      bool
}

// ConfigRefs represents the configuration references for a stack member
type ConfigRefs struct {
	Name     string
	ID       string
	Inputs   []Ref
	Outputs  []Ref
	Resolved bool
}

// StackRef represents the stack reference with inputs, outputs, and members
type StackRef struct {
	Name     string
	ID       string
	Inputs   []Ref
	Outputs  []Ref
	Members  []ConfigRefs
	Resolved bool
}

func CreateStackRefStruct(stackConfig *project.StackDefinition, members []*project.ProjectConfig) (*StackRef, error) {

	// Process the stack members
	stackMemberConfigRefs := ProcessMembers(members)

	inputs := convertStackDefInputsToMap(stackConfig.StackDefinition.Inputs)
	outputs := convertStackDefOutputsToListOfOutputValues(stackConfig.StackDefinition.Outputs)
	// Create a StackRef struct
	stackRefStruct := &StackRef{
		Name:    *stackConfig.Configuration.Definition.Name,
		ID:      *stackConfig.ID,
		Inputs:  ProcessInputs(inputs),
		Outputs: ProcessOutputs(outputs),
		Members: stackMemberConfigRefs,
	}

	unresolvedRefs := GetAllUnresolvedRefs(stackRefStruct)
	if len(unresolvedRefs) > 0 {
		stackRefStruct.Resolved = false
	} else {
		stackRefStruct.Resolved = true
	}

	return stackRefStruct, nil
}

// ProcessInputs processes the input values and returns a slice of Ref structs
func ProcessInputs(inputs map[string]interface{}) []Ref {
	var stackRefInputs []Ref
	if inputs == nil {
		return stackRefInputs
	}
	for inputName, inputValue := range inputs {
		ref, resolvedValue, resolved, isRef := processValue(inputValue)
		stackRefInputs = append(stackRefInputs, Ref{
			Name:          inputName,
			Ref:           ref,
			ResolvedValue: resolvedValue,
			isRef:         isRef,
			Resolved:      resolved,
		})
	}
	return stackRefInputs
}

func convertStackDefInputsToMap(inputs []project.StackDefinitionInputVariable) map[string]interface{} {
	inputsMap := make(map[string]interface{})
	// TODO: Check if input.Default is a always the correct value to use
	//       Do we need to also check somewhere else for the value?
	for _, input := range inputs {
		inputsMap[*input.Name] = input.Default
	}
	return inputsMap
}

// ProcessOutputs processes the output values and returns a slice of Ref structs
func ProcessOutputs(outputs []project.OutputValue) []Ref {
	var stackRefOutputs []Ref
	for _, outputValue := range outputs {
		ref, resolvedValue, resolved, isRef := processValue(outputValue.Value)
		if outputValue.Name != nil && isRef == false {
			resolved = true
		}
		stackRefOutputs = append(stackRefOutputs, Ref{
			Name:          *outputValue.Name,
			Ref:           ref,
			ResolvedValue: resolvedValue,
			isRef:         isRef,
			Resolved:      resolved,
		})
	}
	return stackRefOutputs
}

func convertStackDefOutputsToListOfOutputValues(outputs []project.StackDefinitionOutputVariable) []project.OutputValue {
	var outputValues []project.OutputValue
	for _, output := range outputs {
		outputValues = append(outputValues, project.OutputValue{
			Name:  output.Name,
			Value: output.Value,
		})
	}
	return outputValues
}

// ProcessMembers processes the stack members and returns a slice of ConfigRefs structs
func ProcessMembers(members []*project.ProjectConfig) []ConfigRefs {
	var stackMemberConfigRefs []ConfigRefs
	for _, member := range members {
		memberDef := member.Definition.(*project.ProjectConfigDefinitionResponse)
		memberRefInputs := ProcessInputs(memberDef.Inputs)
		memberRefOutputs := ProcessOutputs(member.Outputs)
		memberResolved := isStackResolved(memberRefInputs, memberRefOutputs)

		stackMemberConfigRefs = append(stackMemberConfigRefs, ConfigRefs{
			Name:     *memberDef.Name,
			ID:       *member.ID,
			Inputs:   memberRefInputs,
			Outputs:  memberRefOutputs,
			Resolved: memberResolved,
		})
	}
	return stackMemberConfigRefs
}

// ResolveReferences traverses the StackRef structure and resolves references
func ResolveReferences(stackRef *StackRef) {
	allResolved := true

	for i := range stackRef.Inputs {
		if stackRef.Inputs[i].isRef {
			resolvedValue := getReferenceValue(stackRef, stackRef.Inputs[i].Ref)
			if resolvedValue != nil {
				stackRef.Inputs[i].ResolvedValue = resolvedValue
				stackRef.Inputs[i].Resolved = true
			} else {
				allResolved = false
			}
		}
	}

	for i := range stackRef.Outputs {
		if stackRef.Outputs[i].isRef {
			resolvedValue := getReferenceValue(stackRef, stackRef.Outputs[i].Ref)
			if resolvedValue != nil {
				stackRef.Outputs[i].ResolvedValue = resolvedValue
				stackRef.Outputs[i].Resolved = true
			} else {
				allResolved = false
			}
		}
	}

	for i := range stackRef.Members {
		resolveConfigRefs(&stackRef.Members[i], stackRef)
		if !stackRef.Members[i].Resolved {
			allResolved = false
		}
	}

	stackRef.Resolved = allResolved
}

// PrintUnresolvedRefs prints the unresolved references in a StackRef structure
func PrintUnresolvedRefs(stackRef *StackRef) {
	fmt.Print(GetAllUnresolvedRefsAsString(stackRef))
}

// PrintAllRefs prints all references in a StackRef structure
func PrintAllRefs(stackRef *StackRef) {
	fmt.Print(GetAllRefsAsString(stackRef))
}

// GetAllRefsAsString returns all references in a StackRef structure as a single string
func GetAllRefsAsString(stackRef *StackRef) string {
	var sb strings.Builder
	for _, input := range stackRef.Inputs {
		if input.Ref != nil {
			if input.ResolvedValue != nil {
				sb.WriteString(fmt.Sprintf("%s - %s(Input): %s Value: %s\n", stackRef.Name, input.Name, *input.Ref, *input.ResolvedValue))
			} else {
				sb.WriteString(fmt.Sprintf("%s - %s(Input): %s (Unresolved)\n", stackRef.Name, input.Name, *input.Ref))
			}
		}
	}
	for _, output := range stackRef.Outputs {
		if output.Ref != nil {
			if output.ResolvedValue != nil {
				sb.WriteString(fmt.Sprintf("%s - %s(Output): %s Value: %s\n", stackRef.Name, output.Name, *output.Ref, *output.ResolvedValue))
			} else {
				sb.WriteString(fmt.Sprintf("%s - %s(Output): %s (Unresolved)\n", stackRef.Name, output.Name, *output.Ref))
			}
		}
	}
	for _, member := range stackRef.Members {
		for _, input := range member.Inputs {
			if input.Ref != nil {
				if input.ResolvedValue != nil {
					sb.WriteString(fmt.Sprintf("%s - %s(Input): %s Value: %s\n", member.Name, input.Name, *input.Ref, *input.ResolvedValue))
				} else {
					sb.WriteString(fmt.Sprintf("%s - %s(Input): %s (Unresolved)\n", member.Name, input.Name, *input.Ref))
				}
			}
		}
		for _, output := range member.Outputs {
			if output.Ref != nil {
				if output.ResolvedValue != nil {
					sb.WriteString(fmt.Sprintf("%s - %s(Output): %s Value: %s\n", member.Name, output.Name, *output.Ref, *output.ResolvedValue))
				} else {
					sb.WriteString(fmt.Sprintf("%s - %s(Output): %s (Unresolved)\n", member.Name, output.Name, *output.Ref))
				}
			}
		}
	}
	return sb.String()
}

// GetAllUnresolvedRefsAsString returns all unresolved references in a StackRef structure as a single string
func GetAllUnresolvedRefsAsString(stackRef *StackRef) string {
	var sb strings.Builder
	for _, input := range stackRef.Inputs {
		if !input.Resolved && input.Ref != nil {
			sb.WriteString(fmt.Sprintf("%s - %s(Input): %s\n", stackRef.Name, input.Name, *input.Ref))
		}
	}
	for _, output := range stackRef.Outputs {
		if !output.Resolved && output.Ref != nil {
			sb.WriteString(fmt.Sprintf("%s - %s(Output): %s\n", stackRef.Name, output.Name, *output.Ref))
		}
	}
	for _, member := range stackRef.Members {
		for _, input := range member.Inputs {
			if !input.Resolved && input.Ref != nil {
				sb.WriteString(fmt.Sprintf("%s - %s(Input): %s\n", member.Name, input.Name, *input.Ref))
			}
		}
		for _, output := range member.Outputs {
			if !output.Resolved && output.Ref != nil {
				sb.WriteString(fmt.Sprintf("%s - %s(Output): %s\n", member.Name, output.Name, *output.Ref))
			}
		}
	}
	return sb.String()
}

// GetAllRefs returns all references in a StackRef structure
func GetAllRefs(stackRef *StackRef) []Ref {
	var refs []Ref
	refs = append(refs, stackRef.Inputs...)
	refs = append(refs, stackRef.Outputs...)
	for _, member := range stackRef.Members {
		refs = append(refs, member.Inputs...)
		refs = append(refs, member.Outputs...)
	}
	return refs
}

// GetAllUnresolvedRefs returns all unresolved references in a StackRef structure
func GetAllUnresolvedRefs(stackRef *StackRef) []Ref {
	var unresolvedRefs []Ref
	allRefs := GetAllRefs(stackRef)
	for _, ref := range allRefs {
		if !ref.Resolved && ref.Ref != nil {
			unresolvedRefs = append(unresolvedRefs, ref)
		}
	}
	return unresolvedRefs
}

// processValue processes a single input value and returns the reference, resolved value, and flags
func processValue(inputValue interface{}) (*string, *string, bool, bool) {
	if inputValue == nil {
		return nil, nil, false, false
	}

	var valueStr string
	value, ok := inputValue.(string)
	if !ok {
		slice, isSlice := inputValue.([]interface{})
		if isSlice {
			valueStr = fmt.Sprintf("%v", convertSliceToString(slice))
		} else {
			return nil, nil, false, false
		}
	} else {
		valueStr = value
	}

	if isReference(valueStr) {
		return &valueStr, nil, false, true
	}
	return nil, &valueStr, true, false
}

// isStackResolved checks if all inputs and outputs are resolved
func isStackResolved(inputs []Ref, outputs []Ref) bool {
	for _, input := range inputs {
		if !input.Resolved {
			return false
		}
	}
	for _, output := range outputs {
		if !output.Resolved {
			return false
		}
	}
	return true
}

// isReference checks if a value is a reference
func isReference(value string) bool {
	return strings.HasPrefix(value, "ref:")
}

// convertSliceToString converts a slice to a string representation
func convertSliceToString(input interface{}) interface{} {
	if reflect.TypeOf(input).Kind() == reflect.Slice {
		slice := reflect.ValueOf(input)
		if slice.Len() == 0 {
			return "[]"
		}
		elements := make([]string, slice.Len())
		for i := 0; i < slice.Len(); i++ {
			element := slice.Index(i).Interface()
			if reflect.TypeOf(element).Kind() == reflect.Slice {
				elements[i] = fmt.Sprintf("%v", convertSliceToString(element))
			} else {
				elements[i] = fmt.Sprintf("\"%v\"", element)
			}
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	}
	return input
}

// resolveConfigRefs traverses the ConfigRefs structure and resolves references
func resolveConfigRefs(configRef *ConfigRefs, root *StackRef) {
	allResolved := true
	for i := range configRef.Inputs {
		if configRef.Inputs[i].isRef {
			resolvedValue := getReferenceValue(root, configRef.Inputs[i].Ref)
			if resolvedValue != nil {
				configRef.Inputs[i].ResolvedValue = resolvedValue
				configRef.Inputs[i].Resolved = true
			} else {
				allResolved = false
			}
		}
	}
	for i := range configRef.Outputs {
		if configRef.Outputs[i].isRef {
			resolvedValue := getReferenceValue(root, configRef.Outputs[i].Ref)
			if resolvedValue != nil {
				configRef.Outputs[i].ResolvedValue = resolvedValue
				configRef.Outputs[i].Resolved = true
			} else {
				allResolved = false
			}
		}
	}
	configRef.Resolved = allResolved
}

// getReferenceValue retrieves the value for a given reference path
func getReferenceValue(root *StackRef, refPath *string) *string {
	if refPath == nil {
		return nil
	}
	// Remove the "ref:" prefix
	path := strings.TrimPrefix(*refPath, "ref:")
	parts := strings.Split(path, "/")
	var ref interface{} = root
	for _, part := range parts {
		if part == ".." {
			// Navigate up the tree (not implemented, I don't think this is needed)
		} else if part != "" {
			ref = navigateToPart(ref, part)
			if ref == nil {
				return nil
			}
		}
	}
	if value, ok := ref.(*string); ok {
		return value
	}
	return nil
}

// navigateToPart navigates to a specific part of the StackRef structure
func navigateToPart(ref interface{}, part string) interface{} {
	switch v := ref.(type) {
	case *StackRef:
		if part == "inputs" {
			return v.Inputs
		} else if part == "outputs" {
			return v.Outputs
		} else if part == "members" {
			return v.Members
		} else {
			// Handle member names
			for _, member := range v.Members {
				if member.Name == part {
					return &member
				}
			}
		}
	case []Ref:
		for _, r := range v {
			if r.Name == part {
				return r.ResolvedValue
			}
		}
	case []ConfigRefs:
		for _, m := range v {
			if m.Name == part {
				return &m
			}
		}
	case *ConfigRefs:
		if part == "inputs" {
			return v.Inputs
		} else if part == "outputs" {
			return v.Outputs
		}
	}
	return nil
}
