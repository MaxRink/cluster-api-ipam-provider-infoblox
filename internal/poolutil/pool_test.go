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

package poolutil

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/telekom/cluster-api-ipam-provider-infoblox/internal/index"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := ipamv1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add ipamv1 to scheme: %v", err)
	}
	return s
}

// claimByCombinedPoolRef is the local equivalent of the unexported
// index.ipAddressClaimByCombinedPoolRef, used only in tests.
func claimByCombinedPoolRef(o client.Object) []string {
	claim, ok := o.(*ipamv1.IPAddressClaim)
	if !ok {
		panic(fmt.Sprintf("Expected an IPAddressClaim but got a %T", o))
	}
	return []string{index.IPPoolRefValue(claim.Spec.PoolRef)}
}

func poolRef(kind, name string) ipamv1.IPPoolReference {
	return ipamv1.IPPoolReference{Kind: kind, Name: name}
}

func makeAddress(name, ns string, ref ipamv1.IPPoolReference) *ipamv1.IPAddress {
	return &ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       ipamv1.IPAddressSpec{PoolRef: ref},
	}
}

func makeClaim(name, ns string, ref ipamv1.IPPoolReference) *ipamv1.IPAddressClaim {
	return &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       ipamv1.IPAddressClaimSpec{PoolRef: ref},
	}
}

// TestListAddressesInUse_ReturnsMatchingAddresses verifies that only addresses
// belonging to the requested pool are returned when field indexing is active.
func TestListAddressesInUse_ReturnsMatchingAddresses(t *testing.T) {
	ref := poolRef("InfobloxIPPool", "my-pool")
	otherRef := poolRef("InfobloxIPPool", "other-pool")

	addr1 := makeAddress("addr-1", "default", ref)
	addr2 := makeAddress("addr-2", "default", ref)
	addrOther := makeAddress("addr-other", "default", otherRef)

	s := newScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
		WithObjects(addr1, addr2, addrOther).
		Build()

	got, err := ListAddressesInUse(context.Background(), c, "default", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(got))
	}
}

// TestListAddressesInUse_EmptyWhenNoneMatch verifies that an empty slice is
// returned when no addresses belong to the requested pool.
func TestListAddressesInUse_EmptyWhenNoneMatch(t *testing.T) {
	ref := poolRef("InfobloxIPPool", "my-pool")
	otherRef := poolRef("InfobloxIPPool", "other-pool")

	s := newScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithIndex(&ipamv1.IPAddress{}, index.IPAddressPoolRefCombinedField, index.IPAddressByCombinedPoolRef).
		WithObjects(makeAddress("addr-other", "default", otherRef)).
		Build()

	got, err := ListAddressesInUse(context.Background(), c, "default", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 addresses, got %d", len(got))
	}
}

// TestListAddressesInUse_PropagatesClientError verifies that errors from the
// underlying client are propagated without wrapping.
func TestListAddressesInUse_PropagatesClientError(t *testing.T) {
	sentinel := errors.New("list failed")
	c := &errClient{err: sentinel}

	_, err := ListAddressesInUse(context.Background(), c, "default", poolRef("InfobloxIPPool", "pool"))
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}
}

// TestListClaimsReferencingPool_ReturnsMatchingClaims verifies that only
// claims belonging to the requested pool are returned when field indexing is active.
func TestListClaimsReferencingPool_ReturnsMatchingClaims(t *testing.T) {
	ref := poolRef("InfobloxIPPool", "my-pool")
	otherRef := poolRef("InfobloxIPPool", "other-pool")

	claim1 := makeClaim("claim-1", "default", ref)
	claim2 := makeClaim("claim-2", "default", ref)
	claimOther := makeClaim("claim-other", "default", otherRef)

	s := newScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithIndex(&ipamv1.IPAddressClaim{}, index.IPAddressClaimPoolRefCombinedField, claimByCombinedPoolRef).
		WithObjects(claim1, claim2, claimOther).
		Build()

	got, err := ListClaimsReferencingPool(context.Background(), c, "default", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 claims, got %d", len(got))
	}
}

// TestListClaimsReferencingPool_EmptyWhenNoneMatch verifies that an empty slice
// is returned when no claims belong to the requested pool.
func TestListClaimsReferencingPool_EmptyWhenNoneMatch(t *testing.T) {
	ref := poolRef("InfobloxIPPool", "my-pool")
	otherRef := poolRef("InfobloxIPPool", "other-pool")

	s := newScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithIndex(&ipamv1.IPAddressClaim{}, index.IPAddressClaimPoolRefCombinedField, claimByCombinedPoolRef).
		WithObjects(makeClaim("claim-other", "default", otherRef)).
		Build()

	got, err := ListClaimsReferencingPool(context.Background(), c, "default", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 claims, got %d", len(got))
	}
}

// TestListClaimsReferencingPool_PropagatesClientError verifies that errors from
// the underlying client are propagated without wrapping.
func TestListClaimsReferencingPool_PropagatesClientError(t *testing.T) {
	sentinel := errors.New("list failed")
	c := &errClient{err: sentinel}

	_, err := ListClaimsReferencingPool(context.Background(), c, "default", poolRef("InfobloxIPPool", "pool"))
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}
}

// errClient is a minimal client.Client that always returns err on List calls.
type errClient struct {
	client.Client
	err error
}

func (e *errClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return e.err
}
