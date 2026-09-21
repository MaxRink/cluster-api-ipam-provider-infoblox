package controllers

import (
	"net"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/telekom/cluster-api-ipam-provider-infoblox/api/v1alpha1"
	"github.com/telekom/cluster-api-ipam-provider-infoblox/pkg/infoblox"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func TestMarkFailedInfobloxRequestClassifiesNetworkFailure(t *testing.T) {
	g := NewWithT(t)
	obj := &v1alpha1.InfobloxInstance{}
	err := markFailedInfobloxRequest(obj, &infoblox.RequestError{
		Endpoint:  "https://infoblox.example:443",
		Operation: "check network view \"TDCN\"",
		Err:       &url.Error{Op: "dial", URL: "https://infoblox.example:443", Err: &net.DNSError{Err: "no such host", Name: "infoblox.example"}},
	}, v1alpha1.NetworkViewNotFoundReason, "default network view \"TDCN\"")

	g.Expect(err).To(MatchError(ContainSubstring("network failure during check network view \"TDCN\" at https://infoblox.example:443")))
	condition := conditions.Get(obj, clusterv1.ReadyCondition)
	g.Expect(condition).NotTo(BeNil())
	g.Expect(condition.Reason).To(Equal(v1alpha1.InfobloxConnectionFailedReason))
	g.Expect(condition.Message).To(And(
		ContainSubstring(`network failure during check network view "TDCN" at https://infoblox.example:443`),
		ContainSubstring("no such host"),
	))
}
