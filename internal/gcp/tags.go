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

package gcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/patrickmn/go-cache"
	"google.golang.org/grpc/codes"
)

const (
	tagCacheDuration = 5 * time.Minute
)

type TagsManager interface {
	LookupKey(ctx context.Context, projectID string, key string) (*resourcemanagerpb.TagKey, error)
	CreateKey(ctx context.Context, projectID string, key string) (*resourcemanagerpb.TagKey, error)
	LookupValue(ctx context.Context, projectID string, key string, value string) (*resourcemanagerpb.TagValue, error)
	CreateValue(ctx context.Context, projectID string, key string, value string) (*resourcemanagerpb.TagValue, error)
}

type tagsManager struct {
	keysClient   *resourcemanager.TagKeysClient
	valuesClient *resourcemanager.TagValuesClient
	cache        *cache.Cache
}

// TODO add logging to this file

func NewTagsManager(keysClient *resourcemanager.TagKeysClient, valuesClient *resourcemanager.TagValuesClient) TagsManager {
	return &tagsManager{
		keysClient:   keysClient,
		valuesClient: valuesClient,
		cache:        cache.New(tagCacheDuration, tagCacheDuration),
	}
}

func (m *tagsManager) LookupKey(ctx context.Context, projectID string, key string) (*resourcemanagerpb.TagKey, error) {
	cacheKey := cacheKeyTagKey(key)
	cachedKey, found := m.cache.Get(cacheKey)
	if found {
		return cachedKey.(*resourcemanagerpb.TagKey), nil
	}

	tagKey, err := m.keysClient.GetNamespacedTagKey(ctx, &resourcemanagerpb.GetNamespacedTagKeyRequest{
		Name: fmt.Sprintf("%s/%s", projectID, key),
	})
	if err != nil {
		var ae *apierror.APIError
		if errors.As(err, &ae) && ae.GRPCStatus().Code() == codes.PermissionDenied {
			return m.CreateKey(ctx, projectID, key)
		}
		return nil, fmt.Errorf("failed to lookup tag key: %w", err)
	}

	m.cache.Set(cacheKey, tagKey, tagCacheDuration)
	return tagKey, nil
}

func (m *tagsManager) CreateKey(ctx context.Context, projectID string, key string) (*resourcemanagerpb.TagKey, error) {
	op, err := m.keysClient.CreateTagKey(ctx, &resourcemanagerpb.CreateTagKeyRequest{
		TagKey: &resourcemanagerpb.TagKey{
			Parent:    fmt.Sprintf("projects/%s", projectID),
			ShortName: key,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tag key: %w", err)
	}
	tagKey, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for tag key creation: %w", err)
	}

	m.cache.Set(cacheKeyTagKey(key), tagKey, tagCacheDuration)
	return tagKey, nil
}

func (m *tagsManager) LookupValue(ctx context.Context, projectID string, key string, value string) (*resourcemanagerpb.TagValue, error) {
	cacheKey := cacheKeyTagValue(key, value)
	cachedValue, found := m.cache.Get(cacheKey)
	if found {
		return cachedValue.(*resourcemanagerpb.TagValue), nil
	}

	tagValue, err := m.valuesClient.GetNamespacedTagValue(ctx, &resourcemanagerpb.GetNamespacedTagValueRequest{
		Name: fmt.Sprintf("%s/%s/%s", projectID, key, value),
	})
	if err != nil {
		var ae *apierror.APIError
		if errors.As(err, &ae) && ae.GRPCStatus().Code() == codes.PermissionDenied {
			return m.CreateValue(ctx, projectID, key, value)
		}
		return nil, fmt.Errorf("failed to lookup tag value: %w", err)
	}

	m.cache.Set(cacheKey, tagValue, tagCacheDuration)
	return tagValue, nil
}

func (m *tagsManager) CreateValue(ctx context.Context, projectID string, key string, value string) (*resourcemanagerpb.TagValue, error) {
	tagKey, err := m.LookupKey(ctx, projectID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup tag key: %w", err)
	}

	op, err := m.valuesClient.CreateTagValue(ctx, &resourcemanagerpb.CreateTagValueRequest{
		TagValue: &resourcemanagerpb.TagValue{
			Parent:    tagKey.Name,
			ShortName: value,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tag value: %w", err)
	}
	tagValue, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for tag value creation: %w", err)
	}

	m.cache.Set(cacheKeyTagValue(key, value), tagValue, tagCacheDuration)
	return tagValue, nil
}

func cacheKeyTagKey(key string) string {
	return fmt.Sprintf("key:%s", key)
}

func cacheKeyTagValue(key string, value string) string {
	return fmt.Sprintf("value:%s:%s", key, value)
}
