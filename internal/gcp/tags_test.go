package gcp

import (
	"context"
	"testing"

	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/gcp/mocks"
	"github.com/stretchr/testify/assert"
)

func TestTagsManager_CreateKey(t *testing.T) {

	mockTagsManager := mocks.NewTagsManager(t)

	ctx := context.Background()
	projectID := "test-project"
	key := "test-key"
	expectedTagKey := &resourcemanagerpb.TagKey{Name: "projects/test-project/keys/test-key"}

	mockTagsManager.On("CreateKey", ctx, projectID, key).Return(expectedTagKey, nil)

	result, err := mockTagsManager.CreateKey(ctx, projectID, key)

	assert.NoError(t, err)
	assert.Equal(t, expectedTagKey, result)

	mockTagsManager.AssertExpectations(t)
}

func TestTagsManager_CreateValue(t *testing.T) {

	mockTagsManager := mocks.NewTagsManager(t)

	ctx := context.Background()
	projectID := "test-project"
	key := "test-key"
	value := "test-value"
	expectedTagValue := &resourcemanagerpb.TagValue{Name: "projects/test-project/keys/test-key/values/test-value"}

	mockTagsManager.On("CreateValue", ctx, projectID, key, value).Return(expectedTagValue, nil)

	result, err := mockTagsManager.CreateValue(ctx, projectID, key, value)

	assert.NoError(t, err)
	assert.Equal(t, expectedTagValue, result)

	mockTagsManager.AssertExpectations(t)
}

func TestTagsManager_LookupKey(t *testing.T) {

	mockTagsManager := mocks.NewTagsManager(t)

	ctx := context.Background()
	projectID := "test-project"
	key := "test-key"
	expectedTagKey := &resourcemanagerpb.TagKey{Name: "projects/test-project/keys/test-key"}

	mockTagsManager.On("LookupKey", ctx, projectID, key).Return(expectedTagKey, nil)

	result, err := mockTagsManager.LookupKey(ctx, projectID, key)

	assert.NoError(t, err)
	assert.Equal(t, expectedTagKey, result)

	mockTagsManager.AssertExpectations(t)
}

func TestTagsManager_LookupValue(t *testing.T) {

	mockTagsManager := mocks.NewTagsManager(t)

	ctx := context.Background()
	projectID := "test-project"
	key := "test-key"
	value := "test-value"
	expectedTagValue := &resourcemanagerpb.TagValue{Name: "projects/test-project/keys/test-key/values/test-value"}

	mockTagsManager.On("LookupValue", ctx, projectID, key, value).Return(expectedTagValue, nil)

	result, err := mockTagsManager.LookupValue(ctx, projectID, key, value)

	assert.NoError(t, err)
	assert.Equal(t, expectedTagValue, result)

	mockTagsManager.AssertExpectations(t)
}
