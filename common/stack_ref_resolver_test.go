package common

import (
	"github.com/IBM/go-sdk-core/v5/core"
	project "github.com/IBM/project-go-sdk/projectv1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcessInputs(t *testing.T) {
	tests := []struct {
		Name     string
		Inputs   map[string]interface{}
		Expected []Ref
	}{
		{
			Name:     "Nil Inputs",
			Inputs:   nil,
			Expected: nil,
		},
		{
			Name:     "Empty Inputs",
			Inputs:   map[string]interface{}{},
			Expected: nil,
		},
		{
			Name: "Single Input",
			Inputs: map[string]interface{}{
				"input1": "value1",
			},
			Expected: []Ref{
				{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
			},
		},
		{
			Name: "Reference Input",
			Inputs: map[string]interface{}{
				"input1": "ref:../outputs/output1",
			},
			Expected: []Ref{
				{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
			},
		},
		{
			Name: "Mixed Inputs",
			Inputs: map[string]interface{}{
				"input1": "value1",
				"input2": "ref:../outputs/output1",
			},
			Expected: []Ref{
				{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				{Name: "input2", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := ProcessInputs(test.Inputs)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestProcessOutputs(t *testing.T) {
	tests := []struct {
		Name     string
		Outputs  []project.OutputValue
		Expected []Ref
	}{
		{
			Name:     "Nil Outputs",
			Outputs:  nil,
			Expected: nil,
		},
		{
			Name:     "Empty Outputs",
			Outputs:  []project.OutputValue{},
			Expected: nil,
		},
		{
			Name: "Single Output",
			Outputs: []project.OutputValue{
				{Name: core.StringPtr("output1"), Value: "value1"},
			},
			Expected: []Ref{
				{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
			},
		},
		{
			Name: "Reference Output",
			Outputs: []project.OutputValue{
				{Name: core.StringPtr("output1"), Value: "ref:../inputs/input1"},
			},
			Expected: []Ref{
				{Name: "output1", Ref: core.StringPtr("ref:../inputs/input1"), isRef: true},
			},
		},
		{
			Name: "Mixed Outputs",
			Outputs: []project.OutputValue{
				{Name: core.StringPtr("output1"), Value: "value1"},
				{Name: core.StringPtr("output2"), Value: "ref:../inputs/input1"},
			},
			Expected: []Ref{
				{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				{Name: "output2", Ref: core.StringPtr("ref:../inputs/input1"), isRef: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := ProcessOutputs(test.Outputs)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestProcessMembers(t *testing.T) {
	tests := []struct {
		Name     string
		Members  []*project.ProjectConfig
		Expected []ConfigRefs
	}{
		{
			Name:     "Nil Members",
			Members:  nil,
			Expected: nil,
		},
		{
			Name:     "Empty Members",
			Members:  []*project.ProjectConfig{},
			Expected: nil,
		},
		{
			Name: "Single Member",
			Members: []*project.ProjectConfig{
				{
					ID: core.StringPtr("12345"),
					Definition: &project.ProjectConfigDefinitionResponse{
						Name:   core.StringPtr("member1"),
						Inputs: map[string]interface{}{"input1": "value1"},
					},
					Outputs: []project.OutputValue{
						{Name: core.StringPtr("output1"), Value: "value1"},
					},
				},
			},
			Expected: []ConfigRefs{
				{
					Name:     "member1",
					ID:       "12345",
					Inputs:   []Ref{{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true}},
					Outputs:  []Ref{{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true}},
					Resolved: true,
				},
			},
		},
		{
			Name: "Multiple Members",
			Members: []*project.ProjectConfig{
				{
					ID: core.StringPtr("12345"),
					Definition: &project.ProjectConfigDefinitionResponse{
						Name:   core.StringPtr("member1"),
						Inputs: map[string]interface{}{"input1": "value1"},
					},
					Outputs: []project.OutputValue{
						{Name: core.StringPtr("output1"), Value: "value1"},
					},
				},
				{
					ID: core.StringPtr("123456"),
					Definition: &project.ProjectConfigDefinitionResponse{
						Name:   core.StringPtr("member2"),
						Inputs: map[string]interface{}{"input2": "value2"},
					},
					Outputs: []project.OutputValue{
						{Name: core.StringPtr("output2"), Value: "value2"},
					},
				},
			},
			Expected: []ConfigRefs{
				{
					Name:     "member1",
					ID:       "12345",
					Inputs:   []Ref{{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true}},
					Outputs:  []Ref{{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true}},
					Resolved: true,
				},
				{
					Name:     "member2",
					ID:       "123456",
					Inputs:   []Ref{{Name: "input2", ResolvedValue: core.StringPtr("value2"), Resolved: true}},
					Outputs:  []Ref{{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true}},
					Resolved: true,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := ProcessMembers(test.Members)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestResolveReferences(t *testing.T) {
	tests := []struct {
		Name     string
		StackRef *StackRef
		Expected *StackRef
	}{
		{
			Name: "Single Unresolved Input",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Resolved: true,
			},
		},
		{
			Name: "Multiple Unresolved Inputs",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
					{Name: "input2", Ref: core.StringPtr("ref:../outputs/output2"), isRef: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
					{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
				},
			},
			Expected: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
					{Name: "input2", Ref: core.StringPtr("ref:../outputs/output2"), isRef: true, ResolvedValue: core.StringPtr("value2"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), isRef: false, Resolved: true},
					{Name: "output2", ResolvedValue: core.StringPtr("value2"), isRef: false, Resolved: true},
				},
				Resolved: true,
			},
		},
		{
			Name: "Unresolved Member Input",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "stackInput1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../inputs/stackInput1"), isRef: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
					},
					{
						Name: "member2",
						Inputs: []Ref{
							{Name: "input2", Ref: core.StringPtr("ref:../members/member1/outputs/output1"), isRef: true},
						},
						Outputs: []Ref{
							{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
					},
				},
			},
			Expected: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "stackInput1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../inputs/stackInput1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Resolved: true,
					},
					{
						Name: "member2",
						Inputs: []Ref{
							{Name: "input2", Ref: core.StringPtr("ref:../members/member1/outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Resolved: true,
					},
				},
				Resolved: true,
			},
		},
		{
			Name: "Unresolved Member Output",
			StackRef: &StackRef{
				Name: "stack1",
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("stack value1"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
							{Name: "output2", isRef: false, ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Resolved: true,
					},
					{
						Name: "member2",
						Inputs: []Ref{
							{Name: "input2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output2", Ref: core.StringPtr("ref:../member1/outputs/output2"), isRef: true},
						},
						Resolved: true,
					},
				},
				Resolved: true,
			},
			Expected: &StackRef{
				Name: "stack1",
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("stack value1"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("stack value1"), Resolved: true},
							{Name: "output2", isRef: false, ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Resolved: true,
					},
					{
						Name: "member2",
						Inputs: []Ref{
							{Name: "input2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output2", Ref: core.StringPtr("ref:../member1/outputs/output2"), isRef: true, ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Resolved: true,
					},
				},
				Resolved: true,
			},
		},
		{
			Name: "Unresolved Member Input and Output",
			StackRef: &StackRef{
				Name: "stack1",
				Outputs: []Ref{
					{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: nil, ResolvedValue: core.StringPtr("value"), isRef: false, Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Resolved: true,
					},
					{
						Name: "member2",
						Inputs: []Ref{
							{Name: "input2", Ref: core.StringPtr("ref:../members/member1/outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output2", Ref: core.StringPtr("ref:../outputs/output2"), isRef: true, ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Resolved: true,
					},
				},
				Resolved: true,
			},
			Expected: &StackRef{
				Name: "stack1",
				Outputs: []Ref{
					{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: nil, isRef: false, ResolvedValue: core.StringPtr("value"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Resolved: true,
					},
					{
						Name: "member2",
						Inputs: []Ref{
							{Name: "input2", Ref: core.StringPtr("ref:../members/member1/outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output2", Ref: core.StringPtr("ref:../outputs/output2"), isRef: true, ResolvedValue: core.StringPtr("value2"), Resolved: true},
						},
						Resolved: true,
					},
				},
				Resolved: true,
			},
		},
		{
			Name: "Multiple Unresolvable Inputs",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/missing_output1"), isRef: true},
					{Name: "input2", Ref: core.StringPtr("ref:../outputs/missing_output2"), isRef: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
					{Name: "output2", ResolvedValue: core.StringPtr("value2"), Resolved: true},
				},
			},
			Expected: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/missing_output1"), isRef: true, ResolvedValue: nil, Resolved: false},
					{Name: "input2", Ref: core.StringPtr("ref:../outputs/missing_output2"), isRef: true, ResolvedValue: nil, Resolved: false},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), isRef: false, Resolved: true},
					{Name: "output2", ResolvedValue: core.StringPtr("value2"), isRef: false, Resolved: true},
				},
				Resolved: false,
			},
		},
		{
			Name: "Unresolvable Member Input",
			StackRef: &StackRef{
				Name: "stack1",
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true, Ref: nil, isRef: false},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../outputs/output_missing"), isRef: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
					},
				},
			},
			Expected: &StackRef{
				Name: "stack1",
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true, Ref: nil, isRef: false},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../outputs/output_missing"), isRef: true, ResolvedValue: nil, Resolved: false},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), isRef: false, Ref: nil, Resolved: true},
						},
						Resolved: false,
					},
				},
				Resolved: false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ResolveReferences(test.StackRef)
			assert.Equal(t, test.Expected, test.StackRef)
		})
	}
}

func TestGetAllRefsAsString(t *testing.T) {
	tests := []struct {
		Name     string
		StackRef *StackRef
		Expected string
	}{
		{
			Name: "Single Resolved Input",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: "stack1 - input1(Input): ref:../outputs/output1 Value: value1\n",
		},
		{
			Name: "No Refs",
			StackRef: &StackRef{
				Name: "stack1",
			},
			Expected: "",
		},
		{
			Name: "Unresolved Member Input",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
						},
					},
				},
			},
			Expected: "member1 - input1(Input): ref:../outputs/output1 (Unresolved)\n",
		},
		{
			Name: "Resolved Member Input and Output",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", ResolvedValue: core.StringPtr("value1"), Ref: core.StringPtr("ref:../sample"), isRef: true, Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Ref: core.StringPtr("ref:../sample"), isRef: true, Resolved: true},
						},
					},
				},
			},
			Expected: "member1 - input1(Input): ref:../sample Value: value1\nmember1 - output1(Output): ref:../sample Value: value1\n",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := GetAllRefsAsString(test.StackRef)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestGetAllUnresolvedRefsAsString(t *testing.T) {
	tests := []struct {
		Name     string
		StackRef *StackRef
		Expected string
	}{
		{
			Name: "Single Unresolved Input",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: "stack1 - input1(Input): ref:../outputs/output1\n",
		},
		{
			Name: "No Unresolved Refs",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: "",
		},
		{
			Name: "Unresolved Member Input",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
						},
					},
				},
			},
			Expected: "member1 - input1(Input): ref:../outputs/output1\n",
		},
		{
			Name: "No Unresolved Member Refs",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
					},
				},
			},
			Expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := GetAllUnresolvedRefsAsString(test.StackRef)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestGetAllRefs(t *testing.T) {
	tests := []struct {
		Name     string
		StackRef *StackRef
		Expected []Ref
	}{
		{
			Name: "Single Resolved Input",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: []Ref{
				{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true, ResolvedValue: core.StringPtr("value1"), Resolved: true},
				{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
			},
		},
		{
			Name: "No Refs",
			StackRef: &StackRef{
				Name: "stack1",
			},
			Expected: nil,
		},
		{
			Name: "Unresolved Member Input",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
						},
					},
				},
			},
			Expected: []Ref{
				{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
			},
		},
		{
			Name: "Resolved Member Input and Output",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
					},
				},
			},
			Expected: []Ref{
				{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := GetAllRefs(test.StackRef)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestGetAllUnresolvedRefs(t *testing.T) {
	tests := []struct {
		Name     string
		StackRef *StackRef
		Expected []Ref
	}{
		{
			Name: "Single Unresolved Input",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: []Ref{
				{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
			},
		},
		{
			Name: "No Unresolved Refs",
			StackRef: &StackRef{
				Name: "stack1",
				Inputs: []Ref{
					{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
				},
			},
			Expected: nil,
		},
		{
			Name: "Unresolved Member Input",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
						},
					},
				},
			},
			Expected: []Ref{
				{Name: "input1", Ref: core.StringPtr("ref:../outputs/output1"), isRef: true},
			},
		},
		{
			Name: "No Unresolved Member Refs",
			StackRef: &StackRef{
				Name: "stack1",
				Members: []ConfigRefs{
					{
						Name: "member1",
						Inputs: []Ref{
							{Name: "input1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "output1", ResolvedValue: core.StringPtr("value1"), Resolved: true},
						},
					},
				},
			},
			Expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := GetAllUnresolvedRefs(test.StackRef)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestCreateStackRef(t *testing.T) {
	tests := []struct {
		Name        string
		StackConfig *project.StackDefinition
		Members     []*project.ProjectConfig
		Expected    *StackRef
		HasError    bool
	}{
		{
			Name: "Valid StackRef",
			StackConfig: &project.StackDefinition{
				ID: core.StringPtr("12345"),
				StackDefinition: &project.StackDefinitionBlock{
					Inputs: []project.StackDefinitionInputVariable{
						{
							Name:        core.StringPtr("stackInput1"),
							Type:        nil,
							Description: nil,
							Default:     "stackValue1",
							Required:    nil,
							Hidden:      nil,
						},
					},
					Outputs: []project.StackDefinitionOutputVariable{
						{Name: core.StringPtr("stackOutput1"), Value: "stackValue1"},
					},
				},
				Configuration: &project.StackDefinitionMetadataConfiguration{
					ID: core.StringPtr("12345"),
					Definition: &project.ConfigDefinitionReference{
						Name: core.StringPtr("stack1"),
					},
					Href: core.StringPtr("href"),
				},
			},
			Members: []*project.ProjectConfig{
				{
					ID: core.StringPtr("12345678"),
					Definition: &project.ProjectConfigDefinitionResponse{
						Name:   core.StringPtr("member1"),
						Inputs: map[string]interface{}{"memberInput1": "memberValue1"},
					},
					Outputs: []project.OutputValue{
						{Name: core.StringPtr("memberOutput1"), Value: "memberValue1"},
					},
				},
			},
			Expected: &StackRef{
				Name: "stack1",
				ID:   "12345",
				Inputs: []Ref{
					{Name: "stackInput1", ResolvedValue: core.StringPtr("stackValue1"), Resolved: true},
				},
				Outputs: []Ref{
					{Name: "stackOutput1", ResolvedValue: core.StringPtr("stackValue1"), Resolved: true},
				},
				Members: []ConfigRefs{
					{
						Name: "member1",
						ID:   "12345678",
						Inputs: []Ref{
							{Name: "memberInput1", ResolvedValue: core.StringPtr("memberValue1"), Resolved: true},
						},
						Outputs: []Ref{
							{Name: "memberOutput1", ResolvedValue: core.StringPtr("memberValue1"), Resolved: true},
						},
						Resolved: true,
					},
				},
				Resolved: true,
			},
			HasError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result, err := CreateStackRefStruct(test.StackConfig, test.Members)
			if test.HasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.Expected, result)
			}
		})
	}
}
