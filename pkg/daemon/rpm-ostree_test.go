package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/yaml"
)

/*
 * This file contains test code for the rpm-ostree client. It is meant to be used when
 * testing the daemon and mocking the responses that would normally be executed by the
 * client.
 */

// GetBootedOSImageURLReturn is a structure used for testing. The fields correspond with the
// return values in GetBootedOSImageURL implementations.
type GetBootedOSImageURLReturn struct {
	OsImageURL string
	Version    string
	Error      error
}

// RpmOstreeClientMock is a testing implementation of NodeUpdaterClient. Fields presented here
// hold return values that will be returned when their corresponding methods are called.
type RpmOstreeClientMock struct {
	GetBootedOSImageURLReturns []GetBootedOSImageURLReturn
}

func (r RpmOstreeClientMock) Initialize() error {
	return nil
}

// GetBootedOSImageURL implements a test version of RpmOStreeClients GetBootedOSImageURL.
// It returns an OsImageURL, Version, and Error as defined in GetBootedOSImageURLReturns in order.
func (r RpmOstreeClientMock) GetBootedOSImageURL() (string, string, error) {
	returnValues := r.GetBootedOSImageURLReturns[0]
	if len(r.GetBootedOSImageURLReturns) > 1 {
		r.GetBootedOSImageURLReturns = r.GetBootedOSImageURLReturns[1:]
	}
	return returnValues.OsImageURL, returnValues.Version, returnValues.Error
}

// PullAndRebase is a mock
func (r RpmOstreeClientMock) Rebase(string, string) (bool, error) {
	return false, nil
}

func (r RpmOstreeClientMock) GetStatus() (string, error) {
	return "rpm-ostree mock: blah blah some status here", nil
}

func (r RpmOstreeClientMock) GetBootedDeployment() (*RpmOstreeDeployment, error) {
	return &RpmOstreeDeployment{}, nil
}

func TestParseVersion(t *testing.T) {
	s := `
rpm-ostree:
  Version: '2021.14'
  Git: v2021.14
  Features:
   - bin-unit-tests
   - compose
   - rust
   - fedora-integration
   `
	var outer rpmOstreeVersionOuter
	assert.Nil(t, yaml.Unmarshal([]byte(s), &outer))
	fmt.Printf("%v", outer)
	assert.Equal(t, outer.Root.Version, "2021.14")
}

func TestValidateVersion(t *testing.T) {
	for _, old := range []string{"2019.5", "2021.6"} {
		v := rpmOstreeVersionOuter{
			Root: rpmOstreeVersionData{
				Version: old,
			},
		}
		assert.NotNil(t, validateVersion(v))
	}
	for _, newver := range []string{"2021.14", "2021.15", "2022.1"} {
		v := rpmOstreeVersionOuter{
			Root: rpmOstreeVersionData{
				Version: newver,
			},
		}
		assert.Nil(t, validateVersion(v))
	}
}

// instead of calling exec with the actual command, run exec with a command that will run our TestExecDiff function, passing through command
func fakeExecDiffCommand(command string, args ...string) *exec.Cmd {
	return exec.Command(os.Args[0], "-test.run=TestExecDiff", "--", command)
}

// this isn't actually a test, it's a helper function that is our mock run of ostree diff
func TestExecDiff(t *testing.T) {
	fmt.Fprint(os.Stdout, `A    /usr/etc/transpiled.ign
A    /usr/etc/rpm-ostree
A    /usr/etc/rpm-ostree/origin.d
A    /usr/etc/rpm-ostree/origin.d/extensions-48783c1.yaml`)
	os.Exit(0)
}

func TestDiff(t *testing.T) {
	execCommand = fakeExecDiffCommand
	defer func() { execCommand = exec.Command }()
	expectedDiffFileSet := []string{
		"/usr/etc/transpiled.ign",
		"/usr/etc/rpm-ostree/origin.d/extensions-48783c1.yaml",
	}

	diffFileSet, err := Diff("", "")
	assert.Nil(t, err)
	assert.Equal(t, expectedDiffFileSet, diffFileSet)
}
