/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gcp_test

import (
	"context"
	"testing"

	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/gcp/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setup() (*mocks.TagsManager, context.Context) {
	mockTagsManager := new(mocks.TagsManager)
	ctx := context.Background()
	return mockTagsManager, ctx
}

func TestTagsManager(t *testing.T) {
	mockTagsManager, ctx := setup()

	tests := []struct {
		name          string
		method        string
		projectID     string
		key           string
		value         string
		expectedKey   *resourcemanagerpb.TagKey
		expectedValue *resourcemanagerpb.TagValue
		expectedError error
		mockSetupFunc func()
		testFunc      func(*testing.T)
	}{
		{
			name:      "CreateKey success",
			method:    "CreateKey",
			projectID: "test-project",
			key:       "test-key",
			expectedKey: &resourcemanagerpb.TagKey{
				Name: "projects/test-project/keys/12345",
			},
			expectedError: nil,
			mockSetupFunc: func() {
				mockTagsManager.On("CreateKey", mock.Anything, "test-project", "test-key").
					Return(&resourcemanagerpb.TagKey{Name: "projects/test-project/keys/12345"}, nil)
			},
			testFunc: func(t *testing.T) {
				tagKey, err := mockTagsManager.CreateKey(ctx, "test-project", "test-key")
				assert.NoError(t, err)
				assert.NotNil(t, tagKey)
				assert.Equal(t, "projects/test-project/keys/12345", tagKey.Name)
			},
		},
		{
			name:      "CreateValue success",
			method:    "CreateValue",
			projectID: "test-project",
			key:       "test-key",
			value:     "test-value",
			expectedValue: &resourcemanagerpb.TagValue{
				Name: "projects/test-project/keys/12345y/values/t12345",
			},
			expectedError: nil,
			mockSetupFunc: func() {
				mockTagsManager.On("CreateValue", mock.Anything, "test-project", "test-key", "test-value").
					Return(&resourcemanagerpb.TagValue{Name: "projects/test-project/keys/12345y/values/t12345"}, nil)
			},
			testFunc: func(t *testing.T) {
				tagValue, err := mockTagsManager.CreateValue(ctx, "test-project", "test-key", "test-value")
				assert.NoError(t, err)
				assert.NotNil(t, tagValue)
				assert.Equal(t, "projects/test-project/keys/12345y/values/t12345", tagValue.Name)
			},
		},
		{
			name:      "LookupKey success",
			method:    "LookupKey",
			projectID: "test-project",
			key:       "test-key",
			expectedKey: &resourcemanagerpb.TagKey{
				Name: "projects/test-project/keys/12345",
			},
			expectedError: nil,
			mockSetupFunc: func() {
				mockTagsManager.On("LookupKey", mock.Anything, "test-project", "test-key").
					Return(&resourcemanagerpb.TagKey{Name: "projects/test-project/keys/12345"}, nil)
			},
			testFunc: func(t *testing.T) {
				tagKey, err := mockTagsManager.LookupKey(ctx, "test-project", "test-key")
				assert.NoError(t, err)
				assert.NotNil(t, tagKey)
				assert.Equal(t, "projects/test-project/keys/12345", tagKey.Name)
			},
		},
		{
			name:      "LookupValue success",
			method:    "LookupValue",
			projectID: "test-project",
			key:       "test-key",
			value:     "test-value",
			expectedValue: &resourcemanagerpb.TagValue{
				Name: "projects/test-project/keys/12345/values/12345",
			},
			expectedError: nil,
			mockSetupFunc: func() {
				mockTagsManager.On("LookupValue", mock.Anything, "test-project", "test-key", "test-value").
					Return(&resourcemanagerpb.TagValue{Name: "projects/test-project/keys/12345/values/12345"}, nil)
			},
			testFunc: func(t *testing.T) {
				tagValue, err := mockTagsManager.LookupValue(ctx, "test-project", "test-key", "test-value")
				assert.NoError(t, err)
				assert.NotNil(t, tagValue)
				assert.Equal(t, "projects/test-project/keys/12345/values/12345", tagValue.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetupFunc()
			tt.testFunc(t)
			mockTagsManager.AssertExpectations(t)
		})
	}
}
