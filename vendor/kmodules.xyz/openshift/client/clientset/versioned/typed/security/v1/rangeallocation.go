/*
Copyright AppsCode Inc. and Contributors

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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1 "kmodules.xyz/openshift/apis/security/v1"
	securityv1 "kmodules.xyz/openshift/client/applyconfiguration/security/v1"
	scheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// RangeAllocationsGetter has a method to return a RangeAllocationInterface.
// A group's client should implement this interface.
type RangeAllocationsGetter interface {
	RangeAllocations() RangeAllocationInterface
}

// RangeAllocationInterface has methods to work with RangeAllocation resources.
type RangeAllocationInterface interface {
	Create(ctx context.Context, rangeAllocation *v1.RangeAllocation, opts metav1.CreateOptions) (*v1.RangeAllocation, error)
	Update(ctx context.Context, rangeAllocation *v1.RangeAllocation, opts metav1.UpdateOptions) (*v1.RangeAllocation, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.RangeAllocation, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.RangeAllocationList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.RangeAllocation, err error)
	Apply(ctx context.Context, rangeAllocation *securityv1.RangeAllocationApplyConfiguration, opts metav1.ApplyOptions) (result *v1.RangeAllocation, err error)
	RangeAllocationExpansion
}

// rangeAllocations implements RangeAllocationInterface
type rangeAllocations struct {
	client rest.Interface
}

// newRangeAllocations returns a RangeAllocations
func newRangeAllocations(c *SecurityV1Client) *rangeAllocations {
	return &rangeAllocations{
		client: c.RESTClient(),
	}
}

// Get takes name of the rangeAllocation, and returns the corresponding rangeAllocation object, and an error if there is any.
func (c *rangeAllocations) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.RangeAllocation, err error) {
	result = &v1.RangeAllocation{}
	err = c.client.Get().
		Resource("rangeallocations").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of RangeAllocations that match those selectors.
func (c *rangeAllocations) List(ctx context.Context, opts metav1.ListOptions) (result *v1.RangeAllocationList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.RangeAllocationList{}
	err = c.client.Get().
		Resource("rangeallocations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested rangeAllocations.
func (c *rangeAllocations) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("rangeallocations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a rangeAllocation and creates it.  Returns the server's representation of the rangeAllocation, and an error, if there is any.
func (c *rangeAllocations) Create(ctx context.Context, rangeAllocation *v1.RangeAllocation, opts metav1.CreateOptions) (result *v1.RangeAllocation, err error) {
	result = &v1.RangeAllocation{}
	err = c.client.Post().
		Resource("rangeallocations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(rangeAllocation).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a rangeAllocation and updates it. Returns the server's representation of the rangeAllocation, and an error, if there is any.
func (c *rangeAllocations) Update(ctx context.Context, rangeAllocation *v1.RangeAllocation, opts metav1.UpdateOptions) (result *v1.RangeAllocation, err error) {
	result = &v1.RangeAllocation{}
	err = c.client.Put().
		Resource("rangeallocations").
		Name(rangeAllocation.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(rangeAllocation).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the rangeAllocation and deletes it. Returns an error if one occurs.
func (c *rangeAllocations) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("rangeallocations").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *rangeAllocations) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("rangeallocations").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched rangeAllocation.
func (c *rangeAllocations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.RangeAllocation, err error) {
	result = &v1.RangeAllocation{}
	err = c.client.Patch(pt).
		Resource("rangeallocations").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied rangeAllocation.
func (c *rangeAllocations) Apply(ctx context.Context, rangeAllocation *securityv1.RangeAllocationApplyConfiguration, opts metav1.ApplyOptions) (result *v1.RangeAllocation, err error) {
	if rangeAllocation == nil {
		return nil, fmt.Errorf("rangeAllocation provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(rangeAllocation)
	if err != nil {
		return nil, err
	}
	name := rangeAllocation.Name
	if name == nil {
		return nil, fmt.Errorf("rangeAllocation.Name must be provided to Apply")
	}
	result = &v1.RangeAllocation{}
	err = c.client.Patch(types.ApplyPatchType).
		Resource("rangeallocations").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
