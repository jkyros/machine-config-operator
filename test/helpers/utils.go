package helpers

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	ign3types "github.com/coreos/ignition/v2/config/v3_2/types"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/openshift/machine-config-operator/test/framework"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// GetMcName returns the current configuration name of the machine config pool poolName
func GetMcName(t *testing.T, cs *framework.ClientSet, poolName string) string {
	// grab the initial machineconfig used by the worker pool
	// this MC is gonna be the one which is going to be reapplied once the previous MC is deleted
	mcp, err := cs.MachineConfigPools().Get(context.TODO(), poolName, metav1.GetOptions{})
	require.Nil(t, err)
	return mcp.Status.Configuration.Name
}

// WaitForConfigAndPoolComplete is a helper function that gets a renderedConfig and waits for its pool to complete.
// The return value is the final rendered config.
func WaitForConfigAndPoolComplete(t *testing.T, cs *framework.ClientSet, pool, mcName string) string {
	config, err := WaitForRenderedConfig(t, cs, pool, mcName)
	require.Nil(t, err, "failed to render machine config %s from pool %s", mcName, pool)
	err = WaitForPoolComplete(t, cs, pool, config)
	require.Nil(t, err, "pool %s did not update to config %s", pool, config)
	return config
}

// WaitForRenderedConfig polls a MachineConfigPool until it has
// included the given mcName in its config, and returns the new
// rendered config name.
func WaitForRenderedConfig(t *testing.T, cs *framework.ClientSet, pool, mcName string) (string, error) {
	var renderedConfig string
	startTime := time.Now()
	if err := wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		mcp, err := cs.MachineConfigPools().Get(context.TODO(), pool, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, mc := range mcp.Spec.Configuration.Source {
			if mc.Name == mcName {
				renderedConfig = mcp.Spec.Configuration.Name
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return "", errors.Wrapf(err, "machine config %s hasn't been picked by pool %s (waited %s)", mcName, pool, time.Since(startTime))
	}
	t.Logf("Pool %s has rendered config %s with %s (waited %v)", pool, mcName, renderedConfig, time.Since(startTime))
	return renderedConfig, nil
}

// WaitForPoolComplete polls a pool until it has completed an update to target
func WaitForPoolComplete(t *testing.T, cs *framework.ClientSet, pool, target string) error {
	startTime := time.Now()
	if err := wait.Poll(2*time.Second, 20*time.Minute, func() (bool, error) {
		mcp, err := cs.MachineConfigPools().Get(context.TODO(), pool, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if mcp.Status.Configuration.Name != target {
			return false, nil
		}
		if mcfgv1.IsMachineConfigPoolConditionTrue(mcp.Status.Conditions, mcfgv1.MachineConfigPoolUpdated) {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return errors.Wrapf(err, "pool %s didn't report %s to updated (waited %s)", pool, target, time.Since(startTime))
	}
	t.Logf("Pool %s has completed %s (waited %v)", pool, target, time.Since(startTime))
	return nil
}

// LabelRandomNodeFromPool gets all nodes in pool and chooses one at random to label
func LabelRandomNodeFromPool(t *testing.T, cs *framework.ClientSet, pool, label string) func() {
	nodes, err := GetNodesByRole(cs, pool)
	require.Nil(t, err)
	require.NotEmpty(t, nodes)

	rand.Seed(time.Now().UnixNano())
	infraNode := nodes[rand.Intn(len(nodes))]
	out, err := exec.Command("oc", "label", "node", infraNode.Name, label+"=", "--overwrite=true").CombinedOutput()
	require.Nil(t, err, "unable to label worker node %s with infra: %s", infraNode.Name, string(out))
	return func() {
		out, err = exec.Command("oc", "label", "node", infraNode.Name, label+"-").CombinedOutput()
		require.Nil(t, err, "unable to remove label from node %s: %s", infraNode.Name, string(out))
	}
}

// GetSingleNodeByRoll gets all nodes by role pool, and asserts there should only be one
func GetSingleNodeByRole(t *testing.T, cs *framework.ClientSet, role string) corev1.Node {
	nodes, err := GetNodesByRole(cs, role)
	require.Nil(t, err)
	require.Len(t, nodes, 1)
	return nodes[0]
}

// GetNodesByRole gets all nodes labeled with role role
func GetNodesByRole(cs *framework.ClientSet, role string) ([]corev1.Node, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{fmt.Sprintf("node-role.kubernetes.io/%s", role): ""}).String(),
	}
	nodes, err := cs.Nodes().List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

// CreateMCP create a machine config pool with name mcpName
// it will also use mcpName as the label selector, so any node you want to be included
// in the pool should have a label node-role.kubernetes.io/mcpName = ""
func CreateMCP(t *testing.T, cs *framework.ClientSet, mcpName string) func() {
	infraMCP := &mcfgv1.MachineConfigPool{}
	infraMCP.Name = mcpName
	nodeSelector := metav1.LabelSelector{}
	infraMCP.Spec.NodeSelector = &nodeSelector
	infraMCP.Spec.NodeSelector.MatchLabels = make(map[string]string)
	infraMCP.Spec.NodeSelector.MatchLabels[MCPNameToRole(mcpName)] = ""
	mcSelector := metav1.LabelSelector{}
	infraMCP.Spec.MachineConfigSelector = &mcSelector
	infraMCP.Spec.MachineConfigSelector.MatchExpressions = []metav1.LabelSelectorRequirement{
		{
			Key:      mcfgv1.MachineConfigRoleLabelKey,
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"worker", mcpName},
		},
	}
	infraMCP.ObjectMeta.Labels = make(map[string]string)
	infraMCP.ObjectMeta.Labels[mcpName] = ""
	_, err := cs.MachineConfigPools().Create(context.TODO(), infraMCP, metav1.CreateOptions{})
	require.Nil(t, err)
	return func() {
		err := cs.MachineConfigPools().Delete(context.TODO(), mcpName, metav1.DeleteOptions{})
		require.Nil(t, err)
	}
}

// MCPNameToRole converts a mcpName to a node role label
func MCPNameToRole(mcpName string) string {
	return fmt.Sprintf("node-role.kubernetes.io/%s", mcpName)
}

// CreateMC creates a machine config object with name and role
func CreateMC(name, role string) *mcfgv1.MachineConfig {
	return NewMachineConfig(name, MCLabelForRole(role), "", nil)
}

// ExecCmdOnNode finds a node's mcd, and oc rsh's into it to execute a command on the node
// all commands should use /rootfs as root
func ExecCmdOnNode(t *testing.T, cs *framework.ClientSet, node corev1.Node, subArgs ...string) string {
	mcd, err := mcdForNode(cs, &node)
	require.Nil(t, err)
	mcdName := mcd.ObjectMeta.Name

	entryPoint := "oc"
	args := []string{"rsh",
		"-n", "openshift-machine-config-operator",
		"-c", "machine-config-daemon",
		mcdName}
	args = append(args, subArgs...)

	cmd := exec.Command(entryPoint, args...)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	require.Nil(t, err, "failed to exec cmd %v on node %s: %s", subArgs, node.Name, string(out))
	return string(out)
}

// IsOKDCluster checks whether the Upstream field on the CV spec references OKD's update server
func IsOKDCluster(cs *framework.ClientSet) (bool, error) {
	cv, err := cs.ClusterVersions().Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if cv.Spec.Upstream == "https://origin-release.svc.ci.openshift.org/graph" {
		return true, nil
	}
	return false, nil
}

func MCLabelForRole(role string) map[string]string {
	mcLabels := make(map[string]string)
	mcLabels[mcfgv1.MachineConfigRoleLabelKey] = role
	return mcLabels
}

func MCLabelForWorkers() map[string]string {
	return MCLabelForRole("worker")
}

// TODO consider also testing for Ign2
// func createIgn2File(path, content, fs string, mode int) ign2types.File {
// 	return ign2types.File{
// 		FileEmbedded1: ign2types.FileEmbedded1{
// 			Contents: ign2types.FileContents{
// 				Source: content,
// 			},
// 			Mode: &mode,
// 		},
// 		Node: ign2types.Node{
// 			Filesystem: fs,
// 			Path:       path,
// 			User: &ign2types.NodeUser{
// 				Name: "root",
// 			},
// 		},
// 	}
// }

func CreateIgn3File(path, content string, mode int) ign3types.File {
	return ign3types.File{
		FileEmbedded1: ign3types.FileEmbedded1{
			Contents: ign3types.Resource{
				Source: &content,
			},
			Mode: &mode,
		},
		Node: ign3types.Node{
			Path: path,
			User: ign3types.NodeUser{
				Name: StrToPtr("root"),
			},
		},
	}
}

func mcdForNode(cs *framework.ClientSet, node *corev1.Node) (*corev1.Pod, error) {
	// find the MCD pod that has spec.nodeNAME = node.Name and get its name:
	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}).String(),
	}
	listOptions.LabelSelector = labels.SelectorFromSet(labels.Set{"k8s-app": "machine-config-daemon"}).String()

	mcdList, err := cs.Pods("openshift-machine-config-operator").List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}
	if len(mcdList.Items) != 1 {
		if len(mcdList.Items) == 0 {
			return nil, fmt.Errorf("failed to find MCD for node %s", node.Name)
		}
		return nil, fmt.Errorf("too many (%d) MCDs for node %s", len(mcdList.Items), node.Name)
	}
	return &mcdList.Items[0], nil
}

// IsCoreOSNode determines if the node is a CoreOS variant
func IsCoreOSNode(n *corev1.Node) bool {
	return strings.Contains(n.Status.NodeInfo.OSImage, "CoreOS")
}
