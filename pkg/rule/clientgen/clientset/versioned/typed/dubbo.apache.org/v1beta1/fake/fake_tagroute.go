/*
Copyright The Kubernetes Authors.

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

// Code generated by clientgen-gen. DO NOT EDIT.

package fake

import (
	"context"
	v1beta1 "github.com/apache/dubbo-admin/pkg/rule/apis/dubbo.apache.org/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTagRoutes implements TagRouteInterface
type FakeTagRoutes struct {
	Fake *FakeDubboV1beta1
	ns   string
}

var tagroutesResource = schema.GroupVersionResource{Group: "dubbo.apache.org", Version: "v1beta1", Resource: "tagroutes"}

var tagroutesKind = schema.GroupVersionKind{Group: "dubbo.apache.org", Version: "v1beta1", Kind: "TagRoute"}

// Get takes name of the tagRoute, and returns the corresponding tagRoute object, and an error if there is any.
func (c *FakeTagRoutes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.TagRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(tagroutesResource, c.ns, name), &v1beta1.TagRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TagRoute), err
}

// List takes label and field selectors, and returns the list of TagRoutes that match those selectors.
func (c *FakeTagRoutes) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.TagRouteList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(tagroutesResource, tagroutesKind, c.ns, opts), &v1beta1.TagRouteList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.TagRouteList{ListMeta: obj.(*v1beta1.TagRouteList).ListMeta}
	for _, item := range obj.(*v1beta1.TagRouteList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tagRoutes.
func (c *FakeTagRoutes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(tagroutesResource, c.ns, opts))

}

// Create takes the representation of a tagRoute and creates it.  Returns the server's representation of the tagRoute, and an error, if there is any.
func (c *FakeTagRoutes) Create(ctx context.Context, tagRoute *v1beta1.TagRoute, opts v1.CreateOptions) (result *v1beta1.TagRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(tagroutesResource, c.ns, tagRoute), &v1beta1.TagRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TagRoute), err
}

// Update takes the representation of a tagRoute and updates it. Returns the server's representation of the tagRoute, and an error, if there is any.
func (c *FakeTagRoutes) Update(ctx context.Context, tagRoute *v1beta1.TagRoute, opts v1.UpdateOptions) (result *v1beta1.TagRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(tagroutesResource, c.ns, tagRoute), &v1beta1.TagRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TagRoute), err
}

// Delete takes name of the tagRoute and deletes it. Returns an error if one occurs.
func (c *FakeTagRoutes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(tagroutesResource, c.ns, name, opts), &v1beta1.TagRoute{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTagRoutes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(tagroutesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.TagRouteList{})
	return err
}

// Patch applies the patch and returns the patched tagRoute.
func (c *FakeTagRoutes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.TagRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(tagroutesResource, c.ns, name, pt, data, subresources...), &v1beta1.TagRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.TagRoute), err
}
