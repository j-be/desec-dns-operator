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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1 "github.com/j-be/desec-dns-operator/api/v1"
	"github.com/j-be/desec-dns-operator/controllers/desec"
	"github.com/j-be/desec-dns-operator/controllers/util"
)

// DesecDnsReconciler reconciles a DesecDns object
type DesecDnsReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	ConfigDir string
}

//+kubebuilder:rbac:groups=desec.owly.dedyn.io,resources=desecdns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=desec.owly.dedyn.io,resources=desecdns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=desec.owly.dedyn.io,resources=desecdns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DesecDns object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DesecDnsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Starting", "req", req)

	// Fetch CR
	dnsCr := v1.DesecDns{}
	if err := r.Client.Get(ctx, req.NamespacedName, &dnsCr); err != nil {
		if errors.IsNotFound(err) {
			log.Info("CR not found, not doing anything", "req", req)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	// Get and check IPs
	ips := dnsCr.Spec.IPs
	if len(ips) == 0 {
		log.Info("Np IPs, not doing anything", "req", req)
		return ctrl.Result{}, nil
	}

	// Create deSEC client
	desecClient, err := desec.NewClient(req.Name, r.ConfigDir)
	if err != nil {
		log.Error(err, "Cannot create client")
		return ctrl.Result{}, err
	}

	// Update IPs
	log.Info("Updating IPs")
	statusUpdate := false
	if err = desecClient.UpdateIp(ips); err != nil {
		statusUpdate = util.UpdateDesecDnsStatus(&dnsCr.Status, "IpUpdate", metav1.ConditionFalse, "Error", err.Error())
	} else {
		message := fmt.Sprintf("Updated to: [%s]", strings.Join(ips, ", "))
		statusUpdate = util.UpdateDesecDnsStatus(&dnsCr.Status, "IpUpdate", metav1.ConditionTrue, "Updated", message)
	}

	if statusUpdate {
		if err := r.Client.Status().Update(ctx, &dnsCr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DesecDnsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ConfigDir = "./mnt"
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.DesecDns{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
