package utils

import (
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"context"
	"google.golang.org/api/iterator"
)

func GetResourceTagValues(ctx context.Context, bindingsClient *resourcemanager.TagBindingsClient, valuesClient *resourcemanager.TagValuesClient, parent string) ([]string, error) {
	it := bindingsClient.ListTagBindings(ctx, &resourcemanagerpb.ListTagBindingsRequest{
		Parent: parent,
	})
	p := iterator.NewPager(it, 300, "")
	var tagValues []string
	for {
		var bindings []*resourcemanagerpb.TagBinding
		nextPageToken, err := p.NextPage(&bindings)
		if err != nil {
			return tagValues, err
		}
		for _, binding := range bindings {
			value, err := valuesClient.GetTagValue(ctx, &resourcemanagerpb.GetTagValueRequest{
				Name: binding.TagValue,
			})
			if err != nil {
				return tagValues, err
			}
			tagValues = append(tagValues, value.NamespacedName)
		}
		if nextPageToken == "" {
			break
		}
	}
	return tagValues, nil
}
