// +build recover

package nsmd_integration_tests

import (
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

func TestNSCDiesSingleNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, true, 1)
}

func TestNSEDiesSingleNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, false, 1)
}

func TestNSCDiesMultiNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, true, 2)
}

func TestNSEDiesMultiNode(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDie(t, false, 2)
}

var NSENoHeal = &pods.NSMgrPodConfig{
	Variables: map[string]string{
		nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
		nsm.NsmdHealDSTWaitTimeout:   "1",    // 1 second
		nsm.NsmdHealEnabled:          "true",
	},
	DataplaneVariables: kubetest.DefaultDataplaneVariables(),
}

func testDie(t *testing.T, killSrc bool, nodesCount int) {
	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	NSENoHeal.Namespace = k8s.GetK8sNamespace()

	nodes, err := kubetest.SetupNodesConfig(k8s, nodesCount, defaultTimeout, []*pods.NSMgrPodConfig{
		NSENoHeal,
		NSENoHeal,
	}, k8s.GetK8sNamespace())
	Expect(err).To(BeNil())

	failures := InterceptGomegaFailures(func() {
		icmp := kubetest.DeployICMP(k8s, nodes[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)
		nsc := kubetest.DeployNSC(k8s, nodes[0].Node, "nsc-1", defaultTimeout)

		ipResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

		ipResponse, errOut, err = k8s.Exec(icmp, icmp.Spec.Containers[0].Name, "ip", "addr")
		Expect(err).To(BeNil())
		Expect(errOut).To(Equal(""))
		Expect(strings.Contains(ipResponse, "nsm")).To(Equal(true))

		pingResponse, errOut, err := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, "ping", "172.16.1.2", "-A", "-c", "5")
		Expect(err).To(BeNil())
		Expect(strings.Contains(pingResponse, "5 packets transmitted, 5 packets received, 0% packet loss")).To(Equal(true))
		logrus.Printf("NSC Ping is success:%s", pingResponse)

		var podToKill *v1.Pod
		var podToCheck *v1.Pod
		if killSrc {
			podToKill = nsc
			podToCheck = icmp
		} else {
			podToKill = icmp
			podToCheck = nsc
		}

		k8s.DeletePods(podToKill)
		success := false
		for attempt := 0; attempt < 20; <-time.After(300 * time.Millisecond) {
			attempt++
			ipResponse, errOut, err = k8s.Exec(podToCheck, podToCheck.Spec.Containers[0].Name, "ip", "addr")
			if !strings.Contains(ipResponse, "nsm") {
				success = true
				break
			}
		}
		Expect(success).To(Equal(true))
	})

	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		for k := 0; k < nodesCount; k++ {
			nsmdLogs, _ := k8s.GetLogs(nodes[k].Nsmd, "nsmd")
			logrus.Errorf("===================== NSMD %d output since test is failing %v\n=====================", k, nsmdLogs)

			nsmdk8sLogs, _ := k8s.GetLogs(nodes[k].Nsmd, "nsmd-k8s")
			logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdk8sLogs)

			nsmdpLogs, _ := k8s.GetLogs(nodes[k].Nsmd, "nsmdp")
			logrus.Errorf("===================== NSMD K8S %d output since test is failing %v\n=====================", k, nsmdpLogs)

			dataplaneLogs, _ := k8s.GetLogs(nodes[k].Dataplane, "")
			logrus.Errorf("===================== Dataplane %d output since test is failing %v\n=====================", k, dataplaneLogs)
		}

		t.Fail()
	}
}
