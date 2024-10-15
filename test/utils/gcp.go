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

package utils

import (
	"context"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
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
