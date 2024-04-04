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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/example/team-operator/api/v1"
	// "github.com/example/team-operator/internal/database"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=platform.example.com,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.example.com,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.example.com,resources=teams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Team object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	l.Info("Reconciling Team")

	var team platformv1.Team

	if err := r.Get(ctx, req.NamespacedName, &team); err != nil {
		l.Error(err, "unable to fetch Team")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// db := database.GetDBConnection()

	// err := database.CreateTeam(db, team.Spec)

	// if err != nil {
	// 	log.Error(err, "unable to create team")
	// 	return ctrl.Result{}, err
	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Team{}).
		Complete(r)
}
