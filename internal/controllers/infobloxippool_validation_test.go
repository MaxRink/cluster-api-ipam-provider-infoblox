/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/telekom/cluster-api-ipam-provider-infoblox/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("InfobloxIPPool CRD validation", func() {
	validPool := func(name string, subnets []v1alpha1.Subnet) *v1alpha1.InfobloxIPPool {
		return &v1alpha1.InfobloxIPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: v1alpha1.InfobloxIPPoolSpec{
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
				Subnets:     subnets,
			},
		}
	}

	Context("CIDR validation", func() {
		It("should accept a valid IPv4 CIDR", func() {
			pool := validPool("valid-v4-cidr", []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"}})
			Expect(poolCreateAndDelete(ctx, k8sClient, pool)).To(Succeed())
		})

		It("should accept a valid IPv6 CIDR", func() {
			pool := validPool("valid-v6-cidr", []v1alpha1.Subnet{{CIDR: "2001:db8::/64", Gateway: "2001:db8::1"}})
			Expect(poolCreateAndDelete(ctx, k8sClient, pool)).To(Succeed())
		})

		It("should reject a non-CIDR string", func() {
			pool := validPool("invalid-cidr", []v1alpha1.Subnet{{CIDR: "not-a-cidr"}})
			err := poolCreateAndDelete(ctx, k8sClient, pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.subnets[0].cidr"))
			Expect(err.Error()).To(ContainSubstring("Invalid value"))
		})

		It("should reject a CIDR without prefix", func() {
			pool := validPool("no-prefix-cidr", []v1alpha1.Subnet{{CIDR: "10.0.0.0"}})
			err := poolCreateAndDelete(ctx, k8sClient, pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.subnets[0].cidr"))
			Expect(err.Error()).To(ContainSubstring("Invalid value"))
		})

		It("should reject an empty CIDR", func() {
			pool := validPool("empty-cidr", []v1alpha1.Subnet{{CIDR: ""}})
			err := poolCreateAndDelete(ctx, k8sClient, pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.subnets"))
		})
	})

	Context("Gateway validation", func() {
		It("should accept a valid IPv4 gateway", func() {
			pool := validPool("valid-v4-gw", []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"}})
			Expect(poolCreateAndDelete(ctx, k8sClient, pool)).To(Succeed())
		})

		It("should accept a valid IPv6 gateway", func() {
			pool := validPool("valid-v6-gw", []v1alpha1.Subnet{{CIDR: "2001:db8::/64", Gateway: "2001:db8::1"}})
			Expect(poolCreateAndDelete(ctx, k8sClient, pool)).To(Succeed())
		})

		It("should accept an empty gateway", func() {
			pool := validPool("empty-gw", []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: ""}})
			Expect(poolCreateAndDelete(ctx, k8sClient, pool)).To(Succeed())
		})

		It("should reject a non-IP gateway", func() {
			pool := validPool("invalid-gw", []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "not-an-ip"}})
			err := poolCreateAndDelete(ctx, k8sClient, pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.subnets[0].gateway"))
			Expect(err.Error()).To(ContainSubstring("Invalid value"))
		})

		It("should reject a gateway with CIDR notation", func() {
			pool := validPool("cidr-gw", []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1/24"}})
			err := poolCreateAndDelete(ctx, k8sClient, pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.subnets[0].gateway"))
			Expect(err.Error()).To(ContainSubstring("Invalid value"))
		})
	})

	Context("Subnet uniqueness (listType=map)", func() {
		It("should accept multiple subnets with different CIDRs", func() {
			pool := validPool("unique-subnets", []v1alpha1.Subnet{
				{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"},
				{CIDR: "10.0.1.0/24", Gateway: "10.0.1.1"},
			})
			Expect(poolCreateAndDelete(ctx, k8sClient, pool)).To(Succeed())
		})

		It("should reject duplicate subnet CIDRs", func() {
			pool := validPool("dup-subnets", []v1alpha1.Subnet{
				{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"},
				{CIDR: "10.0.0.0/24", Gateway: "10.0.0.2"},
			})
			err := poolCreateAndDelete(ctx, k8sClient, pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Duplicate"))
		})
	})
})

func poolCreateAndDelete(ctx context.Context, cl client.Client, obj client.Object) error {
	defer cl.Delete(ctx, obj) //nolint:errcheck
	return cl.Create(ctx, obj)
}
