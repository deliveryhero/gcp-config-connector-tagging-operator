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

type TagsManager struct {
	keysClient   *resourcemanager.TagKeysClient
	valuesClient *resourcemanager.TagValuesClient
	cache        *cache.Cache
}

// TODO add logging to this file

func NewTagsManager(keysClient *resourcemanager.TagKeysClient, valuesClient *resourcemanager.TagValuesClient) *TagsManager {
	return &TagsManager{
		keysClient:   keysClient,
		valuesClient: valuesClient,
		cache:        cache.New(tagCacheDuration, tagCacheDuration),
	}
}

func (m *TagsManager) LookupKey(ctx context.Context, projectID string, key string) (*resourcemanagerpb.TagKey, error) {
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
		return nil, err
	}

	m.cache.Set(cacheKey, tagKey, tagCacheDuration)
	return tagKey, nil
}

func (m *TagsManager) CreateKey(ctx context.Context, projectID string, key string) (*resourcemanagerpb.TagKey, error) {
	op, err := m.keysClient.CreateTagKey(ctx, &resourcemanagerpb.CreateTagKeyRequest{
		TagKey: &resourcemanagerpb.TagKey{
			Parent:    fmt.Sprintf("projects/%s", projectID),
			ShortName: key,
		},
	})
	if err != nil {
		return nil, err
	}
	tagKey, err := op.Wait(ctx)
	if err != nil {
		return nil, err
	}

	m.cache.Set(cacheKeyTagKey(key), tagKey, tagCacheDuration)
	return tagKey, nil
}

func (m *TagsManager) LookupValue(ctx context.Context, projectID string, key string, value string) (*resourcemanagerpb.TagValue, error) {
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
		return nil, err
	}

	m.cache.Set(cacheKey, tagValue, tagCacheDuration)
	return tagValue, nil
}

func (m *TagsManager) CreateValue(ctx context.Context, projectID string, key string, value string) (*resourcemanagerpb.TagValue, error) {
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
		return nil, err
	}
	tagValue, err := op.Wait(ctx)
	if err != nil {
		return nil, err
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
