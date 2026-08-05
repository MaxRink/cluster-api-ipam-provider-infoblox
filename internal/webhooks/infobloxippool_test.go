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

package webhooks

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/telekom/cluster-api-ipam-provider-infoblox/api/v1alpha1"
	"github.com/telekom/cluster-api-ipam-provider-infoblox/internal/index"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const ipamAPIVersion = "ipam.cluster.x-k8s.io/v1beta2"

func TestCreatingPool(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

	namespacedPool := &v1alpha1.InfobloxIPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pool",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.InfobloxIPPoolSpec{
			InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			Subnets:     []v1alpha1.Subnet{{CIDR: "192.168.1.0/24", Gateway: "192.168.1.1"}},
		},
	}

	ips := []client.Object{
		createIP("address00", "192.168.1.2", namespacedPool),
		createIP("address01", "192.168.1.3", namespacedPool),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ips...).
		WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
		Build()

	webhook := InfobloxIPPool{
		Client: fakeClient,
	}

	oldNamespacedPool := namespacedPool.DeepCopy()
	namespacedPool.Spec.Subnets = []v1alpha1.Subnet{{CIDR: "192.168.2.0/24", Gateway: "192.168.2.1"}}

	_, err := webhook.ValidateUpdate(ctx, oldNamespacedPool, namespacedPool)
	g.Expect(err).ToNot(HaveOccurred(), "should not allow removing in use IPs from addresses field in pool")
}

func TestPoolDeletionWithExistingIPAddresses(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

	namespacedPool := &v1alpha1.InfobloxIPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pool",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.InfobloxIPPoolSpec{
			InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			Subnets:     []v1alpha1.Subnet{{CIDR: "192.168.1.0/24", Gateway: "192.168.1.1"}},
		},
	}

	ips := []client.Object{
		createIP("address00", "192.168.1.2", namespacedPool),
		createIP("address01", "192.168.1.3", namespacedPool),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ips...).
		WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
		Build()

	webhook := InfobloxIPPool{
		Client: fakeClient,
	}

	_, err := webhook.ValidateDelete(ctx, namespacedPool)
	g.Expect(err).To(HaveOccurred(), "should not allow deletion when claims exist")

	g.Expect(fakeClient.DeleteAllOf(ctx, &ipamv1.IPAddress{})).To(Succeed())

	_, err = webhook.ValidateDelete(ctx, namespacedPool)
	g.Expect(err).ToNot(HaveOccurred(), "should allow deletion when no claims exist")
}

func TestPoolDeletionWithExistingIPAddressesAndDeletionSkipAnnotation(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

	namespacedPool := &v1alpha1.InfobloxIPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pool",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				SkipValidateDeleteWebhookAnnotation: "",
			},
		},
		Spec: v1alpha1.InfobloxIPPoolSpec{
			InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			Subnets:     []v1alpha1.Subnet{{CIDR: "192.168.1.0/24", Gateway: "192.168.1.1"}},
		},
	}

	ips := []client.Object{
		createIP("address00", "192.168.1.2", namespacedPool),
		createIP("address01", "192.168.1.3", namespacedPool),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ips...).
		WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
		Build()

	webhook := InfobloxIPPool{
		Client: fakeClient,
	}

	_, err := webhook.ValidateDelete(ctx, namespacedPool)
	g.Expect(err).ToNot(HaveOccurred(), "should not allow deletion when claims exist")

	g.Expect(fakeClient.DeleteAllOf(ctx, &ipamv1.IPAddress{})).To(Succeed())
}

func TestUpdatingPool(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

	namespacedPool := &v1alpha1.InfobloxIPPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pool",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.InfobloxIPPoolSpec{
			InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			Subnets:     []v1alpha1.Subnet{{CIDR: "192.168.1.0/24", Gateway: "192.168.1.1"}},
		},
	}

	ips := []client.Object{
		createIP("address00", "192.168.1.2", namespacedPool),
		createIP("address01", "192.168.1.3", namespacedPool),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ips...).
		WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
		Build()

	webhook := InfobloxIPPool{
		Client: fakeClient,
	}

	oldNamespacedPool := namespacedPool.DeepCopy()
	namespacedPool.Spec.Subnets = []v1alpha1.Subnet{{CIDR: "192.168.2.0/24", Gateway: "192.168.2.1"}}

	_, err := webhook.ValidateUpdate(ctx, oldNamespacedPool, namespacedPool)
	g.Expect(err).ToNot(HaveOccurred(), "should not allow removing in use IPs from addresses field in pool")
}

type invalidScenarioTest struct {
	testcase      string
	spec          v1alpha1.InfobloxIPPoolSpec
	expectedError string
}

func TestInvalidScenarios(t *testing.T) {
	tests := []invalidScenarioTest{
		{
			testcase: "addresses must be set",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
			expectedError: "subnets is required",
		},
		{
			testcase: "InstanceRef must be set",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.0/30", Gateway: "10.0.0.1"}},
				InstanceRef: v1alpha1.InstanceReference{},
			},
			expectedError: "InstanceRef.Name is required",
		},
		{
			testcase: "invalid subnet should not be allowed",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.3/30", Gateway: "10.0.0.1"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
			expectedError: "is not a valid CIDR",
		},
		{
			testcase: "invalid gateway should not be allowed",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.3/30", Gateway: "10.0.0.999"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
			expectedError: "is not a valid IP address",
		},
		{
			testcase: "IPv4 subnet and IPv6 gateway should not be allowed",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.3/30", Gateway: "2001:db8::1"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
			expectedError: "CIDR and gateway are mixed IPv4 and IPv6 addresses",
		},
		{
			testcase: "IPv6 subnet and IPv4 gateway should not be allowed",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "2001:db8::0/64", Gateway: "10.0.0.1"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
			expectedError: "CIDR and gateway are mixed IPv4 and IPv6 addresses",
		},
	}
	for _, tt := range tests {
		namespacedPool := &v1alpha1.InfobloxIPPool{Spec: tt.spec}

		g := NewWithT(t)
		scheme := runtime.NewScheme()
		g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

		webhook := InfobloxIPPool{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
				Build(),
		}
		runInvalidScenarioTests(t, tt, namespacedPool, webhook)
	}
}

func TestUnparseableCIDRIsRejectedWithoutPanicking(t *testing.T) {
	// net.ParseCIDR returns a nil *net.IPNet when the CIDR cannot be parsed at
	// all. The cross-family check used to dereference that nil unconditionally,
	// so any pool with a syntactically broken CIDR panicked the webhook server
	// instead of returning a validation error.
	for _, cidr := range []string{"", "garbage", "10.0.0.0/33", "10.0.0.0/"} {
		t.Run(cidr, func(t *testing.T) {
			g := NewWithT(t)

			scheme := runtime.NewScheme()
			g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

			webhook := InfobloxIPPool{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
					Build(),
			}

			pool := &v1alpha1.InfobloxIPPool{
				Spec: v1alpha1.InfobloxIPPoolSpec{
					InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
					Subnets:     []v1alpha1.Subnet{{CIDR: cidr, Gateway: "10.0.0.1"}},
				},
			}

			_, err := webhook.ValidateCreate(context.Background(), pool)
			g.Expect(err).To(MatchError(ContainSubstring("is not a valid CIDR")))
		})
	}
}

func TestUpdateOfPoolMarkedForDeletionSkipsSpecValidation(t *testing.T) {
	// This webhook was inert in-cluster for the provider's entire deployed life
	// (the markers selected a nonexistent apiVersion), so pools that violate these
	// rules are already persisted in real clusters. Every pool also carries
	// ProtectPoolFinalizer, and the controller finishes a deletion by *updating* the
	// pool to drop that finalizer. If spec validation ran on those updates, the
	// rejection would deadlock deletion and strand the pool in Terminating forever.
	scheme := runtime.NewScheme()
	g := NewWithT(t)
	g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

	deletionTime := metav1.Now()

	for _, tt := range []struct {
		testcase string
		spec     v1alpha1.InfobloxIPPoolSpec
	}{
		{
			testcase: "non-canonical CIDR",
			spec: v1alpha1.InfobloxIPPoolSpec{
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.3/30", Gateway: "10.0.0.1"}},
			},
		},
		{
			testcase: "empty subnets",
			spec: v1alpha1.InfobloxIPPoolSpec{
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
				Subnets:     []v1alpha1.Subnet{},
			},
		},
		{
			testcase: "mismatched address families",
			spec: v1alpha1.InfobloxIPPoolSpec{
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "2001:db8::1"}},
			},
		},
		{
			testcase: "missing instance ref",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets: []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"}},
			},
		},
	} {
		t.Run(tt.testcase, func(t *testing.T) {
			g := NewWithT(t)

			webhook := InfobloxIPPool{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
					Build(),
			}

			pool := &v1alpha1.InfobloxIPPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "legacy-pool",
					Namespace:  "test-namespace",
					Finalizers: []string{"ipam.cluster.x-k8s.io/ProtectPool"},
				},
				Spec: tt.spec,
			}

			// Sanity check: while the pool is live, the spec really is rejected.
			_, err := webhook.ValidateUpdate(context.Background(), pool.DeepCopy(), pool.DeepCopy())
			g.Expect(err).To(HaveOccurred(), "invalid spec must still be rejected on a live pool")

			// Once marked for deletion, the controller must be able to drop the
			// finalizer so the deletion can complete.
			terminating := pool.DeepCopy()
			terminating.DeletionTimestamp = &deletionTime
			finalizerRemoved := terminating.DeepCopy()
			finalizerRemoved.Finalizers = nil

			_, err = webhook.ValidateUpdate(context.Background(), terminating, finalizerRemoved)
			g.Expect(err).ToNot(HaveOccurred(),
				"a pool marked for deletion must accept the finalizer-removing update, otherwise deletion deadlocks")
		})
	}
}

func TestValidScenarios(t *testing.T) {
	tests := []struct {
		testcase string
		spec     v1alpha1.InfobloxIPPoolSpec
	}{
		{
			// Subnet.Gateway is marked Optional, so an unset gateway must be
			// accepted. It previously produced two spurious errors: an unparseable
			// address and a bogus mixed-address-family complaint.
			testcase: "subnet without a gateway",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.0/24"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
		},
		{
			testcase: "IPv4 subnet with matching gateway",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
		},
		{
			testcase: "IPv6 subnet with matching gateway",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets:     []v1alpha1.Subnet{{CIDR: "2001:db8::/64", Gateway: "2001:db8::1"}},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
		},
		{
			testcase: "dual-stack pool with one subnet per family",
			spec: v1alpha1.InfobloxIPPoolSpec{
				Subnets: []v1alpha1.Subnet{
					{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1"},
					{CIDR: "2001:db8::/64", Gateway: "2001:db8::1"},
				},
				InstanceRef: v1alpha1.InstanceReference{Name: "test-instance"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testcase, func(t *testing.T) {
			g := NewWithT(t)

			scheme := runtime.NewScheme()
			g.Expect(ipamv1.AddToScheme(scheme)).To(Succeed())

			webhook := InfobloxIPPool{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
					Build(),
			}

			pool := &v1alpha1.InfobloxIPPool{Spec: tt.spec}

			g.Expect(testCreate(context.Background(), pool, &webhook)).To(Succeed())
			g.Expect(testUpdate(context.Background(), pool, &webhook)).To(Succeed())
		})
	}
}

func runInvalidScenarioTests(t *testing.T, tt invalidScenarioTest, pool *v1alpha1.InfobloxIPPool, webhook InfobloxIPPool) {
	t.Helper()
	t.Run(tt.testcase, func(t *testing.T) {
		t.Run("create", func(t *testing.T) {
			t.Helper()

			g := NewWithT(t)
			g.Expect(testCreate(context.Background(), pool, &webhook)).
				To(MatchError(ContainSubstring(tt.expectedError)))
		})
		t.Run("update", func(t *testing.T) {
			t.Helper()

			g := NewWithT(t)
			g.Expect(testUpdate(context.Background(), pool, &webhook)).
				To(MatchError(ContainSubstring(tt.expectedError)))
		})
		t.Run("delete", func(t *testing.T) {
			t.Helper()

			g := NewWithT(t)
			g.Expect(testDelete(context.Background(), pool, &webhook)).
				To(Succeed())
		})
	})
}

func testCreate(ctx context.Context, obj *v1alpha1.InfobloxIPPool, webhook customDefaulterValidator[*v1alpha1.InfobloxIPPool]) error {
	createCopy := obj.DeepCopy()
	if err := webhook.Default(ctx, createCopy); err != nil {
		return err
	}
	_, err := webhook.ValidateCreate(ctx, createCopy)
	return err
}

func testDelete(ctx context.Context, obj *v1alpha1.InfobloxIPPool, webhook customDefaulterValidator[*v1alpha1.InfobloxIPPool]) error {
	deleteCopy := obj.DeepCopy()
	if err := webhook.Default(ctx, deleteCopy); err != nil {
		return err
	}
	_, err := webhook.ValidateDelete(ctx, deleteCopy)
	return err
}

func testUpdate(ctx context.Context, obj *v1alpha1.InfobloxIPPool, webhook customDefaulterValidator[*v1alpha1.InfobloxIPPool]) error {
	updateCopy := obj.DeepCopy()
	updatedCopy := obj.DeepCopy()
	err := webhook.Default(ctx, updateCopy)
	if err != nil {
		return err
	}
	err = webhook.Default(ctx, updatedCopy)
	if err != nil {
		return err
	}
	_, err = webhook.ValidateUpdate(ctx, updateCopy, updatedCopy)
	return err
}

func createIP(name string, ip string, pool *v1alpha1.InfobloxIPPool) *ipamv1.IPAddress {
	return &ipamv1.IPAddress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IPAddress",
			APIVersion: ipamAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pool.Namespace,
		},
		Spec: ipamv1.IPAddressSpec{
			PoolRef: ipamv1.IPPoolReference{
				APIGroup: pool.GetObjectKind().GroupVersionKind().Group,
				Kind:     pool.GetObjectKind().GroupVersionKind().Kind,
				Name:     pool.GetName(),
			},
			Address: ip,
		},
	}
}
