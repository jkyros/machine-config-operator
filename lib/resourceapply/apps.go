package resourceapply

import (
	"context"

	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	mcoResourceMerge "github.com/openshift/machine-config-operator/lib/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsclientv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// ApplyDaemonSet applies the required daemonset to the cluster.
func ApplyDaemonSet(ctx context.Context, client appsclientv1.DaemonSetsGetter, required *appsv1.DaemonSet) (*appsv1.DaemonSet, bool, error) {
	existing, err := client.DaemonSets(required.Namespace).Get(ctx, required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.DaemonSets(required.Namespace).Create(ctx, required, metav1.CreateOptions{})
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	mcoResourceMerge.EnsureDaemonSet(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}

	actual, err := client.DaemonSets(required.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return actual, true, err
}

// ApplyDeployment applies the required deployment to the cluster.
func ApplyDeployment(ctx context.Context, client appsclientv1.DeploymentsGetter, required *appsv1.Deployment) (*appsv1.Deployment, bool, error) {
	existing, err := client.Deployments(required.Namespace).Get(ctx, required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.Deployments(required.Namespace).Create(ctx, required, metav1.CreateOptions{})
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	mcoResourceMerge.EnsureDeployment(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}

	actual, err := client.Deployments(required.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return actual, true, err
}
