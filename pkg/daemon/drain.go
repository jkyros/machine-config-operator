package daemon

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	"github.com/golang/glog"
	ctrlcommon "github.com/openshift/machine-config-operator/pkg/controller/common"
	"github.com/openshift/machine-config-operator/pkg/daemon/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (dn *Daemon) drainRequired() bool {
	// Drain operation is not useful on a single node cluster as there
	// is no other node in the cluster where workload with PDB set
	// can be rescheduled. It can lead to node being stuck at drain indefinitely.
	// These clusters can take advantage of graceful node shutdown feature.
	// https://kubernetes.io/docs/concepts/architecture/nodes/#graceful-node-shutdown
	return !isSingleNodeTopology(dn.getControlPlaneTopology())
}

func (dn *Daemon) performDrain() error {
	// Skip drain process when we're not cluster driven
	if dn.kubeClient == nil {
		return nil
	}

	if !dn.drainRequired() {
		dn.logSystem("Drain not required, skipping")
		dn.nodeWriter.Eventf(corev1.EventTypeNormal, "Drain", "Drain not required, skipping")
		return nil
	}

	desiredConfigName, err := getNodeAnnotation(dn.node, constants.DesiredMachineConfigAnnotationKey)
	if err != nil {
		return err
	}
	desiredDrainAnnotationValue := fmt.Sprintf("%s-%s", "drain", desiredConfigName)
	if dn.node.Annotations[constants.LastAppliedDrainerAnnotationKey] == desiredDrainAnnotationValue {
		// We should only enter this in one of three scenarios:
		// A previous drain timed out, and the controller succeeded while we waited for the next sync
		// Or the MCD restarted for some reason
		// Or we are forcing an update
		// In either case, we just assume the drain we want has been completed
		// A third case we can hit this is that the node fails validation upon reboot, and then we use
		// the forcefile. But in that case, since we never uncordoned, skipping the drain should be fine
		dn.logSystem("drain is already completed on this node")
		MCDDrainErr.Set(0)
		return nil
	}

	// We are here, that means we need to cordon and drain node
	dn.logSystem("Update prepared; requesting cordon and drain via annotation to controller")
	startTime := time.Now()

	// We probably don't need to separate out cordon and drain, but we sort of do it today for various scenarios
	// TODO (jerzhang): revisit
	dn.nodeWriter.Eventf(corev1.EventTypeNormal, "Cordon", "Cordoned node to apply update")
	dn.nodeWriter.Eventf(corev1.EventTypeNormal, "Drain", "Draining node to update config.")

	// TODO (jerzhang): definitely don't have to block here, but as an initial PoC, this is easier
	if err := dn.nodeWriter.SetDesiredDrainer(desiredDrainAnnotationValue); err != nil {
		return fmt.Errorf("Could not set drain annotation: %w", err)
	}

	if err := wait.Poll(10*time.Second, 1*time.Hour, func() (bool, error) {
		node, err := dn.kubeClient.CoreV1().Nodes().Get(context.TODO(), dn.name, metav1.GetOptions{})
		if err != nil {
			glog.Warningf("Failed to get node: %v", err)
			return false, nil
		}
		if node.Annotations[constants.DesiredDrainerAnnotationKey] != node.Annotations[constants.LastAppliedDrainerAnnotationKey] {
			return false, nil
		}
		return true, nil
	}); err != nil {
		if err == wait.ErrWaitTimeout {
			failMsg := fmt.Sprintf("failed to drain node: %s after 1 hour. Please see machine-config-controller logs for more information", dn.node.Name)
			dn.nodeWriter.Eventf(corev1.EventTypeWarning, "FailedToDrain", failMsg)
			MCDDrainErr.Set(1)
			return fmt.Errorf(failMsg)
		}
		return fmt.Errorf("Something went wrong while attempting to drain node: %v", err)
	}

	dn.logSystem("drain complete")
	t := time.Since(startTime).Seconds()
	glog.Infof("Successful drain took %v seconds", t)
	MCDDrainErr.Set(0)

	return nil
}

type ReadFileFunc func(string) ([]byte, error)

// isDrainRequired determines whether node drain is required or not to apply config changes.
func isDrainRequired(actions, diffFileSet []string, readOldFile, readNewFile ReadFileFunc) (bool, error) {
	if ctrlcommon.InSlice(postConfigChangeActionReboot, actions) {
		// Node is going to reboot, we definitely want to perform drain
		return true, nil
	} else if ctrlcommon.InSlice(postConfigChangeActionReloadCrio, actions) {
		// Drain may or may not be necessary in case of container registry config changes.
		if ctrlcommon.InSlice(constants.ContainerRegistryConfPath, diffFileSet) {
			isSafe, err := isSafeContainerRegistryConfChanges(readOldFile, readNewFile)
			if err != nil {
				return false, err
			}
			return !isSafe, nil
		}
		return false, nil
	} else if ctrlcommon.InSlice(postConfigChangeActionNone, actions) {
		return false, nil
	}
	// For any unhandled cases, default to drain
	return true, nil
}

func (dn *Daemon) drainIfRequired(actions, diffFileSet []string, readOldFile, readNewFile ReadFileFunc) error {
	drain, err := isDrainRequired(actions, diffFileSet, readOldFile, readNewFile)
	if err != nil {
		return err
	}

	if drain {
		return dn.performDrain()
	}

	glog.Info("Changes do not require drain, skipping.")
	return nil
}

// isSafeContainerRegistryConfChanges looks inside old and new versions of registries.conf file.
// It compares the content and determines whether changes made are safe or not. This will
// help MCD to decide whether we can skip node drain for applied changes into container
// registry.
// Currently, we consider following container registry config changes as safe to skip node drain:
// 1. A new mirror is added to an existing registry that has `mirror-by-digest-only=true`
// 2. A new registry has been added that has `mirror-by-digest-only=true`
// See https://bugzilla.redhat.com/show_bug.cgi?id=1943315
//nolint:gocyclo
func isSafeContainerRegistryConfChanges(readOldFile, readNewFile ReadFileFunc) (bool, error) {
	// /etc/containers/registries.conf contains config in toml format. Parse the file
	oldData, err := readOldFile(constants.ContainerRegistryConfPath)
	if err != nil {
		return false, fmt.Errorf("failed to get old registries.conf content: %w", err)
	}

	newData, err := readNewFile(constants.ContainerRegistryConfPath)
	if err != nil {
		return false, fmt.Errorf("failed to get new registries.conf content: %w", err)
	}

	tomlConfOldReg := sysregistriesv2.V2RegistriesConf{}
	if _, err := toml.Decode(string(oldData), &tomlConfOldReg); err != nil {
		return false, fmt.Errorf("failed decoding TOML content from file %s: %w", constants.ContainerRegistryConfPath, err)
	}

	tomlConfNewReg := sysregistriesv2.V2RegistriesConf{}
	if _, err := toml.Decode(string(newData), &tomlConfNewReg); err != nil {
		return false, fmt.Errorf("failed decoding TOML content from file %s: %w", constants.ContainerRegistryConfPath, err)
	}

	// Ensure that any unqualified-search-registries has not been deleted
	if len(tomlConfOldReg.UnqualifiedSearchRegistries) > len(tomlConfNewReg.UnqualifiedSearchRegistries) {
		return false, nil
	}
	for i, regURL := range tomlConfOldReg.UnqualifiedSearchRegistries {
		// Order of UnqualifiedSearchRegistries matters since image lookup occurs in order
		if tomlConfNewReg.UnqualifiedSearchRegistries[i] != regURL {
			return false, nil
		}
	}

	oldRegHashMap := make(map[string]sysregistriesv2.Registry)
	for _, reg := range tomlConfOldReg.Registries {
		scope := reg.Location
		if reg.Prefix != "" {
			scope = reg.Prefix
		}
		oldRegHashMap[scope] = reg
	}

	newRegHashMap := make(map[string]sysregistriesv2.Registry)
	for _, reg := range tomlConfNewReg.Registries {
		scope := reg.Location
		if reg.Prefix != "" {
			scope = reg.Prefix
		}
		newRegHashMap[scope] = reg
	}

	// Check for removed registry
	for regLoc := range oldRegHashMap {
		_, ok := newRegHashMap[regLoc]
		if !ok {
			glog.Infof("%s: registry %s has been removed", constants.ContainerRegistryConfPath, regLoc)
			return false, nil
		}
	}

	// Check for modified registry
	for regLoc, newReg := range newRegHashMap {
		oldReg, ok := oldRegHashMap[regLoc]
		if ok {
			// Registry is available in both old and new config.
			if !reflect.DeepEqual(oldReg, newReg) {
				// Registry has been changed in the new config.
				// Check that changes made are safe or not.
				if oldReg.Prefix != newReg.Prefix {
					glog.Infof("%s: prefix value for registry %s has changed from %s to %s",
						constants.ContainerRegistryConfPath, regLoc, oldReg.Prefix, newReg.Prefix)
					return false, nil
				}
				if oldReg.Location != newReg.Location {
					glog.Infof("%s: location value for registry %s has changed from %s to %s",
						constants.ContainerRegistryConfPath, regLoc, oldReg.Location, newReg.Location)
					return false, nil
				}
				if oldReg.Blocked != newReg.Blocked {
					glog.Infof("%s: blocked value for registry %s has changed from %t to %t",
						constants.ContainerRegistryConfPath, regLoc, oldReg.Blocked, newReg.Blocked)
					return false, nil
				}
				if oldReg.Insecure != newReg.Insecure {
					glog.Infof("%s: insecure value for registry %s has changed from %t to %t",
						constants.ContainerRegistryConfPath, regLoc, oldReg.Insecure, newReg.Insecure)
					return false, nil
				}
				if oldReg.MirrorByDigestOnly == newReg.MirrorByDigestOnly {
					// Ensure that all the old mirrors are present
					for _, m := range oldReg.Mirrors {
						if !searchRegistryMirror(m.Location, newReg.Mirrors) {
							glog.Infof("%s: mirror %s has been removed in registry %s",
								constants.ContainerRegistryConfPath, m.Location, regLoc)
							return false, nil
						}
					}
					// Ensure that any added mirror has mirror-by-digest-only set to true
					for _, m := range newReg.Mirrors {
						if !searchRegistryMirror(m.Location, oldReg.Mirrors) && !newReg.MirrorByDigestOnly {
							glog.Infof("%s: mirror %s has been added in registry %s that has mirror-by-digest-only set to %t ",
								constants.ContainerRegistryConfPath, m.Location, regLoc, newReg.MirrorByDigestOnly)
							return false, nil
						}
					}
				} else {
					glog.Infof("%s: mirror-by-digest-only value for registry %s has changed from %t to %t",
						constants.ContainerRegistryConfPath, regLoc, oldReg.MirrorByDigestOnly, newReg.MirrorByDigestOnly)
					return false, nil
				}
			}
		} else if !newReg.MirrorByDigestOnly {
			// New mirrors added into registry but mirror-by-digest-only has been set to false
			glog.Infof("%s: registry %s has been added with mirror-by-digest-only set to %t",
				constants.ContainerRegistryConfPath, regLoc, newReg.MirrorByDigestOnly)
			return false, nil
		}
	}

	glog.Infof("%s: changes made are safe to skip drain", constants.ContainerRegistryConfPath)
	return true, nil
}

// searchRegistryMirror does lookup of a mirror in the mirroList specified for a registry
// Returns true if found
func searchRegistryMirror(loc string, mirrors []sysregistriesv2.Endpoint) bool {
	found := false
	for _, m := range mirrors {
		if m.Location == loc {
			found = true
			break
		}
	}
	return found
}
