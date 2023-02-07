package operator

import (
	"fmt"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	fakeconfigclientset "github.com/openshift/client-go/config/clientset/versioned/fake"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/openshift/machine-config-operator/test/helpers"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestMetrics(t *testing.T) {
	optr := &Operator{
		eventRecorder: &record.FakeRecorder{},
	}
	optr.vStore = newVersionStore()

	p1, p2 := helpers.NewMachineConfigPool("master", nil, helpers.MasterSelector, "v0"), helpers.NewMachineConfigPool("workers", nil, helpers.WorkerSelector, "v0")
	p2.Status.MachineCount = 2
	p2.Status.UpdatedMachineCount = 1
	p2.Status.DegradedMachineCount = 1
	optr.mcpLister = &mockMCPLister{
		pools: []*mcfgv1.MachineConfigPool{p1, p2},
	}

	nodeIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	optr.nodeLister = corelisterv1.NewNodeLister(nodeIndexer)
	nodeIndexer.Add(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "first-node", Labels: map[string]string{"node-role/worker": ""}},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.21",
			},
		},
	})

	coName := fmt.Sprintf("test-%s", uuid.NewUUID())
	co := &configv1.ClusterOperator{ObjectMeta: metav1.ObjectMeta{Name: coName}}
	//cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorAvailable, Status: configv1.ConditionFalse})
	//cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse})
	//cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorDegraded, Status: configv1.ConditionFalse})
	//cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorUpgradeable, Status: configv1.ConditionUnknown})
	//co.Status.Versions = append(co.Status.Versions, configv1.OperandVersion{Name: "operator", Version: "test-version"})
	optr.name = coName
	kasOperator := &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver"},
		Status: configv1.ClusterOperatorStatus{
			Versions: []configv1.OperandVersion{
				{Name: "kube-apiserver", Version: "1.21"},
			},
		},
	}

	optr.configClient = fakeconfigclientset.NewSimpleClientset(co, kasOperator)
	err := optr.syncAll([]syncFunc{
		{name: "fn1",
			fn: func(config *renderConfig) error { return nil },
		},
	})
	require.Nil(t, err)

	metric := testutil.ToFloat64(mcoMachineCount)
	assert.Equal(t, metric, float64(2))

	metric = testutil.ToFloat64(mcoUpdatedMachineCount)
	assert.Equal(t, metric, float64(1))

	metric = testutil.ToFloat64(mcoDegradedMachineCount)
	assert.Equal(t, metric, float64(1))

}
