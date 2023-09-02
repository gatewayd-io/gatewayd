package plugin

import (
	"testing"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/stretchr/testify/assert"
)

// Test_Verify tests the Verify function.
func Test_Verify(t *testing.T) {
	params, err := v1.NewStruct(
		map[string]interface{}{
			"test": "test",
		},
	)
	assert.Nil(t, err)

	returnVal, err := v1.NewStruct(
		map[string]interface{}{
			"test": "test",
		},
	)
	assert.Nil(t, err)

	assert.True(t, Verify(params, returnVal))
}

// Test_Verify_fail tests the Verify function with different parameters to
// ensure it returns false on verification errors.
func Test_Verify_fail(t *testing.T) {
	data := [][]map[string]interface{}{
		{
			{
				"test": "test",
			},
			{
				"test":  "test",
				"test2": "test2",
			},
		},
		{
			{
				"test":  "test",
				"test2": "test2",
			},
			{
				"test": "test",
			},
		},
		{
			{
				"test":  "test",
				"test2": "test2",
			},
			{
				"test":  "test",
				"test3": "test3",
			},
		},
	}

	for _, d := range data {
		params, err := v1.NewStruct(d[0])
		assert.Nil(t, err)
		returnVal, err := v1.NewStruct(d[1])
		assert.Nil(t, err)
		assert.False(t, Verify(params, returnVal))
	}
}

// Test_Verify_nil tests the Verify function with nil parameters.
func Test_Verify_nil(t *testing.T) {
	assert.True(t, Verify(nil, nil))
}
