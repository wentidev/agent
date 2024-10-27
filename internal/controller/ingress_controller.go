/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"github.com/wentidev/agent/internal/utils"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Retrieve the ingress object
	ingress := &networkingv1.Ingress{}
	err := r.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		_, err := utils.DeleteHealthCheck(utils.IngressInfo{
			Name: fmt.Sprintf("%s_%s", req.Namespace, req.Name),
		})
		if err != nil {
			log.Log.Error(err, "unable to delete health check")
			return ctrl.Result{}, err
		}
		log.Log.Info("ingress is being deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	check, err := utils.CreateOrUpdateHealthCheck(utils.IngressInfo{
		Name:        fmt.Sprintf("%s_%s", ingress.Namespace, ingress.Name),
		Description: fmt.Sprintf("%s_%s", ingress.Namespace, ingress.Name),
		Target:      ingress.Spec.Rules[0].Host,
		Port:        utils.GetStringAnnotation(ingress, utils.HealthCheckPort),
		Protocol:    utils.GetStringAnnotation(ingress, utils.HealthCheckProtocol),
		Path:        utils.GetStringAnnotation(ingress, utils.HealthCheckPath),
		Method:      utils.GetStringAnnotation(ingress, utils.HealthCheckMethod),
		HTTPCode:    utils.GetStringAnnotation(ingress, utils.HealthCheckHTTPCode),
		Timeout:     utils.GetStringAnnotation(ingress, utils.HealthCheckTimeout),
		Interval:    utils.GetStringAnnotation(ingress, utils.HealthCheckInterval),
		Enabled:     true,
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Log.Info("health check response", "Status", check)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Named("ingress").
		Complete(r)
}
