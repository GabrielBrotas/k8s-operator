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

	domainv1alpha1 "github.com/gabriel-brotas/domain-operator/api/v1alpha1"
	"github.com/gabriel-brotas/domain-operator/internal/database"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5/pgxpool"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DomainReconciler reconciles a Domain object
type DomainReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	DBConn *pgxpool.Pool
}

const domainFinalizer = "domain.platform.com/controller_finalizer"

// +kubebuilder:rbac:groups=domain.platform.com,resources=domains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=domain.platform.com,resources=domains/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=domain.platform.com,resources=domains/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=get;list;create;update


func (r *DomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("domain", req.NamespacedName)

	domain := domainv1alpha1.Domain{}

	if err := r.Get(ctx, req.NamespacedName, &domain); err != nil {
		if errors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Domain")
		return ctrl.Result{}, err
	}

	log.Info("Processing domain", "domainID", domain.Spec.DomainID, "environments", domain.Spec.Environments)

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(&domain, domainFinalizer) {
		log.Info("Adding Finalizer for the Domain")
		controllerutil.AddFinalizer(&domain, domainFinalizer)
		return ctrl.Result{}, r.Update(ctx, &domain) // the Update will trigger Reconcile function again
	}

	// Check if the domain is marked to be deleted, which is indicated by the deletion timestamp being set.
	if !domain.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&domain, domainFinalizer) {
			// Run finalization logic for domainFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDomain(ctx, log, &domain); err != nil {
				return ctrl.Result{}, err
			}

			// Remove domainFinalizer. Once all finalizers have been
			// removed, the object can be deleted.
			controllerutil.RemoveFinalizer(&domain, domainFinalizer)
			err := r.Update(ctx, &domain)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error removing finalizer %v", err)
			}
		}
		return ctrl.Result{}, nil
	}

	if req.Name != domain.Spec.DomainID {
		log.Info("Resource name mismatch", "expected", domain.Spec.DomainID, "found", req.Name)
		return r.updateDomainStatus(ctx, log, &domain, false)
	}

	if err := domain.Validate(); err != nil {
		log.Info("Domain validation failed", "error", err.Error())
		return r.updateDomainStatus(ctx, log, &domain, false)
	}

	if err := r.syncDomainWithDatabase(ctx, &domain, log); err != nil {
		return ctrl.Result{}, err
	}

	return r.updateDomainStatus(ctx, log, &domain, true)
}

func (r *DomainReconciler) finalizeDomain(ctx context.Context, log logr.Logger, domain *domainv1alpha1.Domain) error {
	log.Info("finalizing domain", "domain", domain.Name)
	err := r.deleteDomainFromDB(log, domain.Name)

	if err != nil {
		log.Error(err, "Error deleting domain from database", "domain", domain.Name)
		return err
	}

	if err := r.deleteNamespaceForDomain(ctx, log, domain.Name); err != nil {
		log.Error(err, "Error deleting namespace for domain", "namespace", domain.Name)
		return err
	}

	log.Info("Successfully finalized domain")
	return nil
}

func (r *DomainReconciler) deleteDomainFromDB(log logr.Logger, domainName string) error {
	if err := database.DeleteDomain(r.DBConn, domainName); err != nil {
		log.Error(err, "Failed to delete domain from database")
		return err
	}

	log.Info("Domain deleted from database")
	return nil
}

func (r *DomainReconciler) updateDomainStatus(ctx context.Context, log logr.Logger, domain *domainv1alpha1.Domain, isValid bool) (ctrl.Result, error) {
	domain.Status.Valid = isValid
	if err := r.Status().Update(ctx, domain); err != nil {
		log.Error(err, "Failed to update domain status")
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("Domain status updated to %t", isValid))

	return ctrl.Result{}, nil
	// return ctrl.Result{RequeueAfter: 3 * time.Minute}, nil // to run the Reconcile function again after 3 minutes
}

func (r *DomainReconciler) syncDomainWithDatabase(ctx context.Context, domain *domainv1alpha1.Domain, log logr.Logger) error {
	exists, err := database.GetDomain(r.DBConn, domain.Spec.DomainID)
	if err != nil {
		log.Error(err, "Failed to query domain in database")
		return err
	}

	if exists.DomainID == "" {
		return r.createOrUpdateDomain(ctx, domain, log, "create")
	}
	return r.createOrUpdateDomain(ctx, domain, log, "update")
}

func (r *DomainReconciler) createOrUpdateDomain(ctx context.Context, domain *domainv1alpha1.Domain, log logr.Logger, operation string) error {
	var err error
	log.Info(fmt.Sprintf("Attempting to %s domain in database", operation))
	if operation == "create" {
		err = database.CreateDomain(r.DBConn, domain.Spec)
	} else {
		err = database.UpdateDomain(r.DBConn, domain.Spec)
	}

	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to %s domain in database", operation))
		return err
	}

	log.Info(fmt.Sprintf("Domain %sd in database", operation))

	err = r.createNamespaceForDomain(ctx, log, domain)

	if err != nil {
		return err
	}

	return nil
}

func (r *DomainReconciler) createNamespaceForDomain(ctx context.Context, log logr.Logger, domain *domainv1alpha1.Domain) error {
	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: domain.Spec.DomainID}, ns)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating namespace for domain", "namespace", domain.Spec.DomainID)

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: domain.Spec.DomainID,
				Labels: map[string]string{
					"platform.com/domain":     domain.Spec.DomainID,
					"platform.com/managed-by": "domain-operator",
				},
			},
		}

		err = r.Create(ctx, ns)
		if err != nil {
			return fmt.Errorf("failed to create namespace for domain %s: %w", domain.Spec.DomainID, err)
		}

		// Set the ownerRef for the Deployment
		// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
		// fix: error cluster-scoped resource must not have a namespace-scoped owner, owner's namespace default
		// if err := ctrl.SetControllerReference(domain, ns, r.Scheme); err != nil {
		// 	return err
		// }

		log.Info("Namespace created", "namespace", domain.Spec.DomainID)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", domain.Spec.DomainID, err)
	}

	log.Info("Namespace already exists", "namespace", domain.Spec.DomainID)

	return nil
}

func (r *DomainReconciler) deleteNamespaceForDomain(ctx context.Context, log logr.Logger, domainName string) error {
	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: domainName}, ns)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Namespace does not exist", "namespace", domainName)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get namespace %s for deletion: %w", domainName, err)
	}

	err = r.Delete(ctx, ns)
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", domainName, err)
	}

	log.Info("Namespace deleted", "namespace", domainName)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&domainv1alpha1.Domain{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
