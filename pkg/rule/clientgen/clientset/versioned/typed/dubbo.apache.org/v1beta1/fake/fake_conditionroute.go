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

// FakeConditionRoutes implements ConditionRouteInterface
type FakeConditionRoutes struct {
	Fake *FakeDubboV1beta1
	ns   string
}

var conditionroutesResource = schema.GroupVersionResource{Group: "dubbo.apache.org", Version: "v1beta1", Resource: "conditionroutes"}

var conditionroutesKind = schema.GroupVersionKind{Group: "dubbo.apache.org", Version: "v1beta1", Kind: "ConditionRoute"}

// Get takes name of the conditionRoute, and returns the corresponding conditionRoute object, and an error if there is any.
func (c *FakeConditionRoutes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.ConditionRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(conditionroutesResource, c.ns, name), &v1beta1.ConditionRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ConditionRoute), err
}

// List takes label and field selectors, and returns the list of ConditionRoutes that match those selectors.
func (c *FakeConditionRoutes) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.ConditionRouteList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(conditionroutesResource, conditionroutesKind, c.ns, opts), &v1beta1.ConditionRouteList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.ConditionRouteList{ListMeta: obj.(*v1beta1.ConditionRouteList).ListMeta}
	for _, item := range obj.(*v1beta1.ConditionRouteList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested conditionRoutes.
func (c *FakeConditionRoutes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(conditionroutesResource, c.ns, opts))

}

// Create takes the representation of a conditionRoute and creates it.  Returns the server's representation of the conditionRoute, and an error, if there is any.
func (c *FakeConditionRoutes) Create(ctx context.Context, conditionRoute *v1beta1.ConditionRoute, opts v1.CreateOptions) (result *v1beta1.ConditionRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(conditionroutesResource, c.ns, conditionRoute), &v1beta1.ConditionRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ConditionRoute), err
}

// Update takes the representation of a conditionRoute and updates it. Returns the server's representation of the conditionRoute, and an error, if there is any.
func (c *FakeConditionRoutes) Update(ctx context.Context, conditionRoute *v1beta1.ConditionRoute, opts v1.UpdateOptions) (result *v1beta1.ConditionRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(conditionroutesResource, c.ns, conditionRoute), &v1beta1.ConditionRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ConditionRoute), err
}

// Delete takes name of the conditionRoute and deletes it. Returns an error if one occurs.
func (c *FakeConditionRoutes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(conditionroutesResource, c.ns, name, opts), &v1beta1.ConditionRoute{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeConditionRoutes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(conditionroutesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.ConditionRouteList{})
	return err
}

// Patch applies the patch and returns the patched conditionRoute.
func (c *FakeConditionRoutes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.ConditionRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(conditionroutesResource, c.ns, name, pt, data, subresources...), &v1beta1.ConditionRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ConditionRoute), err
}
