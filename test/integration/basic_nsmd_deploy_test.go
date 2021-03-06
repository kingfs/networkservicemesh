// +build basic

package nsmd_integration_tests

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMgrDdataplaneDeploy(t *testing.T) {
	testNSMgrDdataplaneDeploy(t, pods.NSMgrPod, pods.VPPDataplanePod)
}

func TestNSMgrDdataplaneDeployLiveCheck(t *testing.T) {
	testNSMgrDdataplaneDeploy(t, pods.NSMgrPodLiveCheck, pods.VPPDataplanePodLiveCheck)
}

func testNSMgrDdataplaneDeploy(t *testing.T, nsmdPodFactory func(string, *v1.Node, string) *v1.Pod, dataplanePodFactory func(string, *v1.Node) *v1.Pod) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	nodes := k8s.GetNodesWait(2, defaultTimeout)

	if len(nodes) < 2 {
		logrus.Printf("At least two Kubernetes nodes are required for this test")
		Expect(len(nodes)).To(Equal(2))
		return
	}

	_, err = kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())
	k8s.Cleanup()
	var count int = 0
	for _, lpod := range k8s.ListPods() {
		logrus.Printf("Found pod %s %+v", lpod.Name, lpod.Status)
		if strings.Contains(lpod.Name, "nsmgr") {
			count += 1
		}
	}
	Expect(count).To(Equal(int(0)))
}
