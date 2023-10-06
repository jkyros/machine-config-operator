// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "github.com/openshift/api/machineconfiguration/v1"
	machineconfigurationv1 "github.com/openshift/client-go/machineconfiguration/applyconfigurations/machineconfiguration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeControllerConfigs implements ControllerConfigInterface
type FakeControllerConfigs struct {
	Fake *FakeMachineconfigurationV1
}

var controllerconfigsResource = v1.SchemeGroupVersion.WithResource("controllerconfigs")

var controllerconfigsKind = v1.SchemeGroupVersion.WithKind("ControllerConfig")

// Get takes name of the controllerConfig, and returns the corresponding controllerConfig object, and an error if there is any.
func (c *FakeControllerConfigs) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ControllerConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(controllerconfigsResource, name), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}

// List takes label and field selectors, and returns the list of ControllerConfigs that match those selectors.
func (c *FakeControllerConfigs) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ControllerConfigList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(controllerconfigsResource, controllerconfigsKind, opts), &v1.ControllerConfigList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.ControllerConfigList{ListMeta: obj.(*v1.ControllerConfigList).ListMeta}
	for _, item := range obj.(*v1.ControllerConfigList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested controllerConfigs.
func (c *FakeControllerConfigs) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(controllerconfigsResource, opts))
}

// Create takes the representation of a controllerConfig and creates it.  Returns the server's representation of the controllerConfig, and an error, if there is any.
func (c *FakeControllerConfigs) Create(ctx context.Context, controllerConfig *v1.ControllerConfig, opts metav1.CreateOptions) (result *v1.ControllerConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(controllerconfigsResource, controllerConfig), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}

// Update takes the representation of a controllerConfig and updates it. Returns the server's representation of the controllerConfig, and an error, if there is any.
func (c *FakeControllerConfigs) Update(ctx context.Context, controllerConfig *v1.ControllerConfig, opts metav1.UpdateOptions) (result *v1.ControllerConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(controllerconfigsResource, controllerConfig), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeControllerConfigs) UpdateStatus(ctx context.Context, controllerConfig *v1.ControllerConfig, opts metav1.UpdateOptions) (*v1.ControllerConfig, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(controllerconfigsResource, "status", controllerConfig), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}

// Delete takes name of the controllerConfig and deletes it. Returns an error if one occurs.
func (c *FakeControllerConfigs) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(controllerconfigsResource, name, opts), &v1.ControllerConfig{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeControllerConfigs) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(controllerconfigsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1.ControllerConfigList{})
	return err
}

// Patch applies the patch and returns the patched controllerConfig.
func (c *FakeControllerConfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ControllerConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(controllerconfigsResource, name, pt, data, subresources...), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied controllerConfig.
func (c *FakeControllerConfigs) Apply(ctx context.Context, controllerConfig *machineconfigurationv1.ControllerConfigApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ControllerConfig, err error) {
	if controllerConfig == nil {
		return nil, fmt.Errorf("controllerConfig provided to Apply must not be nil")
	}
	data, err := json.Marshal(controllerConfig)
	if err != nil {
		return nil, err
	}
	name := controllerConfig.Name
	if name == nil {
		return nil, fmt.Errorf("controllerConfig.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(controllerconfigsResource, *name, types.ApplyPatchType, data), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeControllerConfigs) ApplyStatus(ctx context.Context, controllerConfig *machineconfigurationv1.ControllerConfigApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ControllerConfig, err error) {
	if controllerConfig == nil {
		return nil, fmt.Errorf("controllerConfig provided to Apply must not be nil")
	}
	data, err := json.Marshal(controllerConfig)
	if err != nil {
		return nil, err
	}
	name := controllerConfig.Name
	if name == nil {
		return nil, fmt.Errorf("controllerConfig.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(controllerconfigsResource, *name, types.ApplyPatchType, data, "status"), &v1.ControllerConfig{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ControllerConfig), err
}
