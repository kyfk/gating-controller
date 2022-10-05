/*
Copyright 2022.

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

	"github.com/fluxcd/pkg/runtime/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/gating-controller/api/v1alpha1"
	gatingv1alpha1 "github.com/fluxcd/gating-controller/api/v1alpha1"
	"github.com/fluxcd/gating-controller/internal/object"
)

// GateReconciler reconciles a Gate object
type GateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	OpenRequestedAnnotation  string = "open.gate.fluxcd.io/requestedAt"
	CloseRequestedAnnotation string = "close.gate.fluxcd.io/requestedAt"
)

//+kubebuilder:rbac:groups=gating.toolkit.fluxcd.io,resources=gates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gating.toolkit.fluxcd.io,resources=gates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gating.toolkit.fluxcd.io,resources=gates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *GateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// log := ctrl.LoggerFrom(ctx)

	var gate *gatingv1alpha1.Gate
	err := r.Get(ctx, req.NamespacedName, gate)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Set a condition if the Gate is just created, conditions is nil.
	if len(gate.Status.Conditions) == 0 {
		condition := r.defaultOpenedCondition(gate)
		conditions.Set(gate, condition)
		return ctrl.Result{RequeueAfter: gate.GetRequeueAfter()}, nil
	}

	// Patch the status when a user has overridden the annotation.
	requestedAt, resetToDefaultAt, condition, ok, err := r.newStatusFromAnnotating(gate)
	if err != nil {
		return ctrl.Result{}, err
	}
	if ok {
		r.patchStatus(gate, requestedAt, resetToDefaultAt, condition)
		return ctrl.Result{RequeueAfter: gate.GetRequeueAfter()}, nil
	}

	err = r.tryResetToDefault(gate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: gate.GetRequeueAfter()}, nil
}

func (r *GateReconciler) defaultOpenedCondition(gate *v1alpha1.Gate) *metav1.Condition {
	switch gate.Spec.Default {
	case v1alpha1.SpecDefaultOpened:
		return conditions.TrueCondition(v1alpha1.OpenedCondition, v1alpha1.ReconciliationSucceededReason, "Gate scheduled for opening at %s")
	case v1alpha1.SpecDefaultClosed:
		return conditions.FalseCondition(v1alpha1.OpenedCondition, v1alpha1.ReconciliationSucceededReason, "Gate closed by default")
	default:
		return nil
	}
}

func (r *GateReconciler) newStatusFromAnnotating(gate *v1alpha1.Gate) (string, string, *metav1.Condition, bool, error) {
	annotations := gate.GetAnnotations()

	// Reset the all annotations.
	defer func() {
		annotations[OpenRequestedAnnotation] = ""
		annotations[CloseRequestedAnnotation] = ""
		gate.SetAnnotations(annotations)
	}()

	var requestedAtStr string
	var condition *metav1.Condition
	if annotations[OpenRequestedAnnotation] != "" {
		requestedAtStr = annotations[OpenRequestedAnnotation]
		condition = conditions.TrueCondition(v1alpha1.OpenedCondition, v1alpha1.ReconciliationSucceededReason, "Gate scheduled for closing at %s", requestedAtStr)
	} else if annotations[CloseRequestedAnnotation] != "" {
		requestedAtStr = annotations[CloseRequestedAnnotation]
		condition = conditions.FalseCondition(v1alpha1.OpenedCondition, v1alpha1.ReconciliationSucceededReason, "Gate close requested")
	} else {
		return "", "", nil, false, nil
	}

	requestedAt, err := time.Parse(time.RFC3339, requestedAtStr)
	if err != nil {
		return "", "", nil, false, err
	}

	interval, err := object.GetRequeueInterval(gate)
	if err != nil {
		return "", "", nil, false, err
	}

	resetToDefaultAt := requestedAt.Add(interval)

	return requestedAtStr, resetToDefaultAt.Format(time.RFC3339), condition, true, nil
}

func (r *GateReconciler) tryResetToDefault(gate *v1alpha1.Gate) error {
	if gate.Status.ResetToDefaultAt == "" {
		return nil
	}

	at, err := time.Parse(time.RFC3339, gate.Status.ResetToDefaultAt)
	if err != nil {
		return err
	}

	// still in the window.
	if at.After(time.Now()) {
		return nil
	}

	// out of the window.
	r.patchStatus(gate, "", "", r.defaultOpenedCondition(gate))
	return nil
}

func (r *GateReconciler) patchStatus(gate *v1alpha1.Gate, requestedAt, resetToDefaultAt string, condition *metav1.Condition) {
	gate.Status.RequestedAt = requestedAt
	gate.Status.ResetToDefaultAt = resetToDefaultAt
	conditions.Set(gate, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatingv1alpha1.Gate{}).
		Complete(r)
}
