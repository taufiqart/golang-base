package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectNullFields_EmptyBody(t *testing.T) {
	fields := DetectNullFields([]byte{})
	assert.Nil(t, fields)
}

func TestDetectNullFields_InvalidJSON(t *testing.T) {
	fields := DetectNullFields([]byte("not json"))
	assert.Nil(t, fields)
}

func TestDetectNullFields_NoNulls(t *testing.T) {
	body := []byte(`{"name":"test","age":25,"active":true}`)
	fields := DetectNullFields(body)
	assert.Empty(t, fields)
}

func TestDetectNullFields_ExplicitNull(t *testing.T) {
	body := []byte(`{"name":"test","max_age":null,"description":null}`)
	fields := DetectNullFields(body)

	assert.Len(t, fields, 2)
	assert.Contains(t, fields, "max_age")
	assert.Contains(t, fields, "description")
}

func TestDetectNullFields_MixedValues(t *testing.T) {
	body := []byte(`{"name":"test","max_age":null,"age":25}`)
	fields := DetectNullFields(body)

	assert.Len(t, fields, 1)
	assert.Contains(t, fields, "max_age")
	assert.NotContains(t, fields, "name")
	assert.NotContains(t, fields, "age")
}

func TestDetectNullFields_ObjectValue(t *testing.T) {
	body := []byte(`{"meta":null,"items":[1,2,3]}`)
	fields := DetectNullFields(body)

	assert.Len(t, fields, 1)
	assert.Contains(t, fields, "meta")
}

func TestDetectNullFields_NestedMap(t *testing.T) {
	body := []byte(`{"outer":{"inner":null}}`)
	fields := DetectNullFields(body)

	assert.Empty(t, fields)
}

func TestDetectNullFields_UsedViaUnmarshal(t *testing.T) {
	body := []byte(`{"name":"test","expiry_date":null,"issued_date":"2024-01-01"}`)

	nullFields := DetectNullFields(body)

	var req struct {
		Name       *string `json:"name"`
		ExpiryDate *string `json:"expiry_date"`
		IssuedDate *string `json:"issued_date"`
	}
	err := json.Unmarshal(body, &req)
	assert.NoError(t, err)

	assert.Len(t, nullFields, 1)
	assert.Contains(t, nullFields, "expiry_date")
	assert.NotNil(t, req.Name)
	assert.Nil(t, req.ExpiryDate)
	assert.NotNil(t, req.IssuedDate)
}
