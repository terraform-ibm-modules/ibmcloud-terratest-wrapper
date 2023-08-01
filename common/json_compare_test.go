package common

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSortMap(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantStr string
	}{
		{
			name:    "simple map",
			jsonStr: `{"b": 2, "a": 1}`,
			wantStr: `{"a": 1, "b": 2}`,
		},
		{
			name:    "nested map",
			jsonStr: `{"b": {"b": 2, "a": 1}, "a": 1}`,
			wantStr: `{"a": 1, "b": {"a": 1, "b": 2}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse jsonStr into map
			data := make(map[string]interface{})
			err := json.Unmarshal([]byte(tt.jsonStr), &data)
			if err != nil {
				t.Fatalf("Failed to unmarshal jsonStr: %v", err)
			}

			// Parse wantStr into map
			want := make(map[string]interface{})
			err = json.Unmarshal([]byte(tt.wantStr), &want)
			if err != nil {
				t.Fatalf("Failed to unmarshal wantStr: %v", err)
			}

			SortMap(data)
			if diff := cmp.Diff(want, data); diff != "" {
				t.Errorf("SortMap() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSortSlice(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantStr string
	}{
		{
			name:    "simple slice",
			jsonStr: `[2, 1]`,
			wantStr: `[1, 2]`,
		},
		{
			name:    "slice of maps",
			jsonStr: `[{"b": 2, "a": 1}, {"d": 4, "c": 3}]`,
			wantStr: `[{"a": 1, "b": 2}, {"c": 3, "d": 4}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse jsonStr into slice
			data := make([]interface{}, 0)
			err := json.Unmarshal([]byte(tt.jsonStr), &data)
			if err != nil {
				t.Fatalf("Failed to unmarshal jsonStr: %v", err)
			}

			// Parse wantStr into slice
			want := make([]interface{}, 0)
			err = json.Unmarshal([]byte(tt.wantStr), &want)
			if err != nil {
				t.Fatalf("Failed to unmarshal wantStr: %v", err)
			}

			SortSlice(data)
			if diff := cmp.Diff(want, data, cmpopts.EquateEmpty(), cmpopts.EquateApprox(0.0, 0.00001)); diff != "" {
				t.Errorf("SortSlice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompareJSONs(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr1 string
		jsonStr2 string
	}{
		{
			name: "Test case 1",
			jsonStr1: `{
				"foo": "bar",
				"stuff": [
					{
						"this": "that",
						"another": "value",
						"myindex": 1
					},
					{
						"this": "that2",
						"another": "value2",
						"myindex": 2
					}
				]
			}`,
			jsonStr2: `{
				"stuff": [
					{
						"this": "that2",
						"another": "value2",
						"myindex": 2
					},
					{
						"this": "that",
						"another": "value",
						"myindex": 1
					}
				],
				"foo": "bar"
			}`,
		},
		{
			name: "Test case 2",
			jsonStr1: `{
				"foo": "bar",
				"stuff": [
					{
						"this": "that",
						"another": "value",
						"myindex": 1,
						"deepnest": [
							{
								"key1": "value1",
								"key2": "value2"
							},
							{
								"key3": "value3",
								"key4": "value4"
							}
						]
					},
					{
						"this": "that2",
						"another": "value2",
						"myindex": 2
					}
				]
			}`,
			jsonStr2: `{
				"stuff": [
					{
						"this": "that2",
						"another": "value2",
						"myindex": 2
					},
					{
						"deepnest": [
							{
								"key4": "value4",
								"key3": "value3"
							},
							{
								"key2": "value2",
								"key1": "value1"
							}
						],						
						"this": "that",
						"another": "value",
						"myindex": 1
					}
				],
				"foo": "bar"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse jsonStr1 into data1
			data1 := make(map[string]interface{})
			err := json.Unmarshal([]byte(tt.jsonStr1), &data1)
			if err != nil {
				t.Fatalf("Failed to unmarshal jsonStr1: %v", err)
			}

			// Parse jsonStr2 into data2
			data2 := make(map[string]interface{})
			err = json.Unmarshal([]byte(tt.jsonStr2), &data2)
			if err != nil {
				t.Fatalf("Failed to unmarshal jsonStr2: %v", err)
			}

			SortMap(data1)
			SortMap(data2)

			if diff := cmp.Diff(data1, data2, cmpopts.EquateEmpty(), cmpopts.EquateApprox(0.0, 0.00001)); diff != "" {
				t.Errorf("SortMap() mismatch (-jsonStr1 +jsonStr2):\n%s", diff)
				println(diff)
			}
		})
	}
}
