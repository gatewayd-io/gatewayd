package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

// Test_sha256sum tests the sha256sum function.
func Test_sha256sum(t *testing.T) {
	checksum, err := sha256sum("../LICENSE")
	assert.Nil(t, err)
	assert.Equal(t,
		"8486a10c4393cee1c25392769ddd3b2d6c242d6ec7928e1414efff7dfb2f07ef",
		checksum,
	)
}

// Test_sha256sum_fail tests the sha256sum function with a file that does not exist.
func Test_sha256sum_fail(t *testing.T) {
	_, err := sha256sum("not_a_file")
	assert.NotNil(t, err)
}

// Test_Verify tests the Verify function.
func Test_Verify(t *testing.T) {
	params, err := structpb.NewStruct(
		map[string]interface{}{
			"test": "test",
		},
	)
	assert.Nil(t, err)

	returnVal, err := structpb.NewStruct(
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
		params, err := structpb.NewStruct(d[0])
		assert.Nil(t, err)
		returnVal, err := structpb.NewStruct(d[1])
		assert.Nil(t, err)
		assert.False(t, Verify(params, returnVal))
	}
}

// Test_Verify_nil tests the Verify function with nil parameters.
func Test_Verify_nil(t *testing.T) {
	assert.True(t, Verify(nil, nil))
}
