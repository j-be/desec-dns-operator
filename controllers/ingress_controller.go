/*
Copyright 2023.

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
	"time"

	"golang.org/x/exp/slices"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/j-be/desec-dns-operator/api/v1"
	"github.com/j-be/desec-dns-operator/controllers/config"
	"github.com/j-be/desec-dns-operator/controllers/desec"
	"github.com/j-be/desec-dns-operator/controllers/util"
)

// IngressReconciler reconciles a DesecDns object
type IngressReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	ConfigDir string
}

//+kubebuilder:rbac:groups=desec.owly.dedyn.io,resources=desecdns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=desec.owly.dedyn.io,resources=desecdns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=desec.owly.dedyn.io,resources=desecdns/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DesecDns object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Starting", "req", req)

	// Create deSEC client
	desecConfig, err := config.NewConfigFor(r.ConfigDir)
	if err != nil {
		log.Error(err, "Failed to read the configuration")
		return ctrl.Result{}, err
	}
	desecClient, err := desec.NewClient(desecConfig.Domain, r.ConfigDir)
	if err != nil {
		log.Error(err, "Cannot create client")
		return ctrl.Result{}, err
	}

	// Fetch or create CR
	dnsCr := new(v1.DesecDns)
	if err := r.Get(ctx, desecConfig.GetNamespacedName(), dnsCr); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to load CR", "req", req)
			return ctrl.Result{}, err
		}
		// Initialize
		dnsCr = util.InitializeDesecDns(desecConfig.GetNamespacedName())
		err := r.Create(ctx, dnsCr)
		return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
	}

	// Initialize status
	if len(dnsCr.Status.Conditions) == 0 {
		dnsCr.Status = util.InitializeDesecDnsStatus()
		err := r.Status().Update(ctx, dnsCr)
		return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
	}

	// Make sure domain exists
	domains, err := desecClient.GetDomains()
	if err != nil {
		log.Error(err, "Failed to fetch domains")
		return ctrl.Result{}, err
	}
	if !slices.ContainsFunc(domains, func(domain desec.Domain) bool { return domain.Name == desecClient.Domain }) {
		if util.UpdateDesecDnsStatus(&dnsCr.Status, "Domain", metav1.ConditionFalse, "Creating", "") {
			if err := r.Status().Update(ctx, dnsCr); err != nil {
				return ctrl.Result{}, err
			}
		}
		_, err := desecClient.CreateDomain()
		return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
	}
	if util.UpdateDesecDnsStatus(&dnsCr.Status, "Domain", metav1.ConditionTrue, "Created", "") {
		err := r.Status().Update(ctx, dnsCr)
		return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
	}

	// Fetch ingress
	ingress := networkingv1.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		log.Error(err, "Failed to load ingress", "req", req)
		return ctrl.Result{}, err
	}

	// Make sure all IPs are in Spec
	ips := util.GetIps(ingress)
	slices.Sort(ips)
	if !slices.Equal(ips, dnsCr.Spec.IPs) {
		dnsCr.Spec.IPs = ips
		err := r.Update(ctx, dnsCr)
		return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
	}

	// Add missing CNAMES
	rrsets, _ := desecClient.GetRRSets()
	for _, subname := range util.GetSubnames(ingress, desecClient.Domain) {
		if !slices.ContainsFunc(rrsets, func(rrset desec.RRSet) bool { return rrset.Type == "CNAME" && rrset.Subname == subname }) {
			log.Info("Adding CNAME", "subname", subname, "domain", desecClient.Domain)
			if util.UpdateDesecDnsStatus(&dnsCr.Status, subname, metav1.ConditionFalse, "Creating", "") {
				if err := r.Status().Update(ctx, dnsCr); err != nil {
					return ctrl.Result{}, err
				}
			}
			cname, err := desecClient.CreateCNAME(subname)
			if err == nil {
				log.Info("CNAME created", "cname", cname)
			}
			return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
		}
		if util.UpdateDesecDnsStatus(&dnsCr.Status, subname, metav1.ConditionTrue, "Created", "") {
			err := r.Status().Update(ctx, dnsCr)
			return ctrl.Result{Requeue: true, RequeueAfter: 100 * time.Millisecond}, err
		}
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ConfigDir = "./mnt"
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
