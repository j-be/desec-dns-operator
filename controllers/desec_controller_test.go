package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/j-be/desec-dns-operator/api/v1"
	"github.com/j-be/desec-dns-operator/controllers/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestDesecDnsReconciler(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		// Given
		updated := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/", r.URL.Path)
			assert.Equal(t, "Token I'm a token", r.Header.Get("Authorization"))
			assert.Equal(t, "1.2.3.4", r.URL.Query().Get("myip"))
			assert.Equal(t, "some-domain.dedyn.io", r.URL.Query().Get("hostname"))
			_, err := w.Write([]byte("good"))
			assert.NoError(t, err)
			updated = true
		}))
		defer server.Close()
		reconciler := createDesecDnsReconciler(t, server.URL, []string{"1.2.3.4"})
		desec := new(v1.DesecDns)
		assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, desec))
		assert.Empty(t, desec.Status.Conditions)

		statusGeneration := ""
		for i := 1; i < 3; i = i + 1 {
			// When
			result, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: util.NamespacedName})
			// Then
			assert.True(t, updated)
			assert.NoError(t, err)
			assert.Equal(t, 5*time.Minute, result.RequeueAfter)

			assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, desec))
			if len(statusGeneration) == 0 {
				statusGeneration = desec.ResourceVersion
			}
			assert.Equal(t, statusGeneration, desec.ResourceVersion)
			condition := meta.FindStatusCondition(desec.Status.Conditions, "IpUpdate")
			assert.NotNil(t, condition)
			assert.Equal(t, metav1.ConditionTrue, condition.Status)
			assert.Equal(t, "Updated", condition.Reason)
			assert.Equal(t, "Updated to: [1.2.3.4]", condition.Message)
		}
	})

	t.Run("No IPs, no call", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { assert.Fail(t, "Should not have been called") }))
		defer server.Close()
		reconciler := createDesecDnsReconciler(t, server.URL, []string{})
		// When
		result, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: util.NamespacedName})
		// Then
		assert.NoError(t, err)
		assert.True(t, result.IsZero())

		desec := new(v1.DesecDns)
		assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, desec))
		assert.Empty(t, desec.Status.Conditions)
	})

	t.Run("Error is registered", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
		defer server.Close()
		reconciler := createDesecDnsReconciler(t, server.URL, []string{"1.2.3.4"})
		desec := new(v1.DesecDns)

		statusGeneration := ""
		for i := 1; i < 3; i = i + 1 {
			// When
			_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: util.NamespacedName})
			// Then
			assert.EqualError(t, err, "got status code 404")

			assert.NoError(t, reconciler.Get(context.TODO(), util.NamespacedName, desec))
			if len(statusGeneration) == 0 {
				statusGeneration = desec.ResourceVersion
			}
			assert.Equal(t, statusGeneration, desec.ResourceVersion)
			assert.Equal(t, statusGeneration, desec.ResourceVersion)
			condition := meta.FindStatusCondition(desec.Status.Conditions, "IpUpdate")
			assert.NotNil(t, condition)
			assert.Equal(t, metav1.ConditionFalse, condition.Status)
			assert.Equal(t, "Error", condition.Reason)
			assert.Equal(t, "got status code 404", condition.Message)
		}
	})

	t.Run("Not doing anything if not found", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
		defer server.Close()
		reconciler := createDesecDnsReconciler(t, server.URL, []string{"1.2.3.4"})
		// When
		result, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      "IDoNotExist",
			Namespace: util.NamespacedName.Namespace,
		}})
		// Then
		assert.NoError(t, err)
		assert.True(t, result.IsZero())
	})
}

func createDesecDnsReconciler(t *testing.T, serverUrl string, ips []string) DesecDnsReconciler {
	objects := []client.Object{
		&v1.DesecDns{
			Spec:       v1.DesecDnsSpec{IPs: ips},
			ObjectMeta: metav1.ObjectMeta{Name: util.NamespacedName.Name, Namespace: util.NamespacedName.Namespace},
		},
	}

	mockScheme := runtime.NewScheme()
	assert.NoError(t, v1.AddToScheme(mockScheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(mockScheme).
		WithObjects(objects...).
		WithStatusSubresource(objects...).
		Build()

	configDir := util.CreateConfigDir(t, serverUrl)

	return DesecDnsReconciler{
		Client:    fakeClient,
		Scheme:    mockScheme,
		ConfigDir: configDir,
	}
}
