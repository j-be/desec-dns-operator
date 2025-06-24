package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/j-be/desec-dns-operator/api/v1"

	"github.com/j-be/desec-dns-operator/controllers/desec"
	"github.com/j-be/desec-dns-operator/controllers/util"
	"github.com/stretchr/testify/assert"
	netv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var ingressRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "some-ingress", Namespace: "some-namespace"}}

func TestIngressReconciler(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		// Given
		domains := make([]desec.Domain, 0)
		rrsets := make([]desec.RRSet, 0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/domains/":
				switch r.Method {
				case "GET":
					body, err := json.Marshal(domains)
					assert.NoError(t, err)
					_, err = w.Write(body)
					assert.NoError(t, err)
				case "POST":
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					domain := desec.Domain{}
					assert.NoError(t, json.Unmarshal(body, &domain))
					domains = append(domains, domain)
					w.WriteHeader(201)
					_, err = w.Write(body)
					assert.NoError(t, err)
				default:
					t.Fail()
				}
			case "/api/v1/domains/some-domain.dedyn.io/rrsets/":
				switch r.Method {
				case "GET":
					body, err := json.Marshal(rrsets)
					assert.NoError(t, err)
					_, err = w.Write(body)
					assert.NoError(t, err)
				case "POST":
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					rrset := desec.RRSet{}
					assert.NoError(t, json.Unmarshal(body, &rrset))
					rrsets = append(rrsets, rrset)
					w.WriteHeader(201)
					_, err = w.Write(body)
					assert.NoError(t, err)
				default:
					t.Fail()
				}
			default:
				t.Fail()
			}
		}))
		defer server.Close()
		reconciler := createIngressReconciler(t, server.URL)
		dnsCr := new(v1.DesecDns)
		assert.EqualError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr), `desecdnses.desec.owly.dedyn.io "some-domain.dedyn.io" not found`)
		// Init CR + Status
		for i := 0; i < 2; i = i + 1 {
			result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
			assert.NoError(t, err)
			assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)
			assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
			assert.Len(t, dnsCr.Status.Conditions, i*2)
		}
		for _, conditionType := range []string{"Domain", "IpUpdate"} {
			condition := meta.FindStatusCondition(dnsCr.Status.Conditions, conditionType)
			assert.NotNil(t, condition)
			assert.Equal(t, metav1.ConditionUnknown, condition.Status)
			assert.Equal(t, "Initializing", condition.Reason)
		}
		// Create the domain
		{
			assert.Empty(t, domains)
			result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
			assert.NoError(t, err)
			assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)
			assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
			domainCondition := meta.FindStatusCondition(dnsCr.Status.Conditions, "Domain")
			assert.NotNil(t, domainCondition)
			assert.Equal(t, metav1.ConditionFalse, domainCondition.Status)
			assert.Equal(t, "Creating", domainCondition.Reason)
			// Actually, it is already created here
			assert.Len(t, domains, 1)
			assert.Equal(t, "some-domain.dedyn.io", domains[0].Name)
		}
		// Update condition
		{
			result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
			assert.NoError(t, err)
			assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)
			assert.Len(t, domains, 1)
			assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
			domainCondition := meta.FindStatusCondition(dnsCr.Status.Conditions, "Domain")
			assert.NotNil(t, domainCondition)
			assert.Equal(t, metav1.ConditionTrue, domainCondition.Status)
			assert.Equal(t, "Created", domainCondition.Reason)
		}
		// Update IPs
		{
			assert.Empty(t, dnsCr.Spec.IPs)
			result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
			assert.NoError(t, err)
			assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)
			assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
			assert.Equal(t, dnsCr.Spec.IPs, []string{"1.2.3.4", "2.3.4.5"})
		}
		assert.Empty(t, rrsets)
		for i, subname := range []string{"www", "git"} {
			// Create CNAME
			{
				assert.Nil(t, meta.FindStatusCondition(dnsCr.Status.Conditions, subname))
				result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
				assert.NoError(t, err)
				assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)
				assert.Len(t, rrsets, i+1)
				cname := rrsets[i]
				assert.Equal(t, "some-domain.dedyn.io", cname.Domain)
				assert.Equal(t, "CNAME", cname.Type)
				assert.Equal(t, subname, cname.Subname)
				assert.Equal(t, subname+".some-domain.dedyn.io.", cname.Name)
				assert.Equal(t, []string{"some-domain.dedyn.io."}, cname.Records)
				assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
				condition := meta.FindStatusCondition(dnsCr.Status.Conditions, subname)
				assert.NotNil(t, condition)
				assert.Equal(t, metav1.ConditionFalse, condition.Status)
				assert.Equal(t, "Creating", condition.Reason)
			}
			// Update associated condition
			{
				result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
				assert.NoError(t, err)
				assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)
				assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
				condition := meta.FindStatusCondition(dnsCr.Status.Conditions, subname)
				assert.NotNil(t, condition)
				assert.Equal(t, metav1.ConditionTrue, condition.Status)
				assert.Equal(t, "Created", condition.Reason)
			}
		}
		// Do nothing
		{
			resourceVersion := dnsCr.ResourceVersion
			for i := 0; i < 5; i = i + 1 {
				result, err := reconciler.Reconcile(context.TODO(), ingressRequest)
				assert.NoError(t, err)
				assert.True(t, result.IsZero())
				assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, dnsCr))
				assert.Equal(t, resourceVersion, dnsCr.ResourceVersion)
			}
		}
		// Make sure IpUpdate condition wasn't touched
		{
			ipUpdateCondition := meta.FindStatusCondition(dnsCr.Status.Conditions, "IpUpdate")
			assert.NotNil(t, ipUpdateCondition)
			assert.Equal(t, metav1.ConditionUnknown, ipUpdateCondition.Status)
			assert.Equal(t, "Initializing", ipUpdateCondition.Reason)
		}
	})

	t.Run("Not doing anything if not found", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/domains/", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			body, err := json.Marshal([]desec.Domain{{Name: "some-domain.dedyn.io"}})
			assert.NoError(t, err)
			_, err = w.Write(body)
			assert.NoError(t, err)
		}))
		defer server.Close()
		reconciler := createIngressReconciler(t, server.URL)
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "IDoNotExist", Namespace: ingressRequest.Namespace}}
		for i := 0; i < 3; i = i + 1 {
			_, err := reconciler.Reconcile(context.TODO(), request)
			assert.NoError(t, err)
		}
		// When
		result, err := reconciler.Reconcile(context.TODO(), request)
		// Then
		assert.EqualError(t, err, `ingresses.networking.k8s.io "IDoNotExist" not found`)
		assert.True(t, result.IsZero())
	})
}

func createIngressReconciler(t *testing.T, serverUrl string) IngressReconciler {
	objects := []client.Object{
		&netv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "some-ingress", Namespace: "some-namespace"},
			Spec: netv1.IngressSpec{Rules: []netv1.IngressRule{
				{Host: "some-domain.dedyn.io"},
				{Host: "www.some-domain.dedyn.io"},
				{Host: "wrong-domain.dedyn.io"},
				{Host: "www.wrong-domain.dedyn.io"},
				{Host: "git.some-domain.dedyn.io"},
			}},
			Status: netv1.IngressStatus{LoadBalancer: netv1.IngressLoadBalancerStatus{Ingress: []netv1.IngressLoadBalancerIngress{
				{IP: "1.2.3.4"},
				{IP: "2.3.4.5"},
			}}},
		},
	}

	mockScheme := runtime.NewScheme()
	assert.NoError(t, v1.AddToScheme(mockScheme))
	assert.NoError(t, netv1.AddToScheme(mockScheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(mockScheme).
		WithObjects(objects...).
		WithStatusSubresource(objects...).
		WithStatusSubresource(new(v1.DesecDns)).
		Build()

	configDir := util.CreateConfigDir(t, serverUrl)

	return IngressReconciler{
		Client:    fakeClient,
		Scheme:    mockScheme,
		ConfigDir: configDir,
	}
}
