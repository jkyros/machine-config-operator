package resourceapply

import (
	"context"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	mcoResourceMerge "github.com/openshift/machine-config-operator/lib/resourcemerge"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcfgclientv1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplyMachineConfig applies the required machineconfig to the cluster.
func ApplyMachineConfig(ctx context.Context, client mcfgclientv1.MachineConfigsGetter, required *mcfgv1.MachineConfig) (*mcfgv1.MachineConfig, bool, error) {
	existing, err := client.MachineConfigs().Get(ctx, required.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.MachineConfigs().Create(ctx, required, metav1.CreateOptions{})
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	mcoResourceMerge.EnsureMachineConfig(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}

	actual, err := client.MachineConfigs().Update(ctx, existing, metav1.UpdateOptions{})
	return actual, true, err
}

// ApplyMachineConfigPool applies the required machineconfig to the cluster.
func ApplyMachineConfigPool(ctx context.Context, client mcfgclientv1.MachineConfigPoolsGetter, required *mcfgv1.MachineConfigPool) (*mcfgv1.MachineConfigPool, bool, error) {
	existing, err := client.MachineConfigPools().Get(ctx, required.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.MachineConfigPools().Create(ctx, required, metav1.CreateOptions{})
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	mcoResourceMerge.EnsureMachineConfigPool(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}

	actual, err := client.MachineConfigPools().Update(ctx, existing, metav1.UpdateOptions{})
	return actual, true, err
}

// ApplyControllerConfig applies the required machineconfig to the cluster.
func ApplyControllerConfig(ctx context.Context, client mcfgclientv1.ControllerConfigsGetter, required *mcfgv1.ControllerConfig) (*mcfgv1.ControllerConfig, bool, error) {
	existing, err := client.ControllerConfigs().Get(ctx, required.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.ControllerConfigs().Create(ctx, required, metav1.CreateOptions{})
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	mcoResourceMerge.EnsureControllerConfig(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}

	actual, err := client.ControllerConfigs().Update(ctx, existing, metav1.UpdateOptions{})
	return actual, true, err
}
