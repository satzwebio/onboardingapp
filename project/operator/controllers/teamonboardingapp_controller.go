package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	onboardingv1alpha1 "github.com/example/team-onboarding-operator/api/v1alpha1"
)

type TeamOnboardingAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *TeamOnboardingAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	
	// Fetch the TeamOnboardingApp instance
	var app onboardingv1alpha1.TeamOnboardingApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize status if not set
	if app.Status.Phase == "" {
		app.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create or ensure namespace exists
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: app.Spec.Namespace,
		},
	}
	if err := r.Create(ctx, namespace); err != nil && !errors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}

	// Update phase to Creating
	if app.Status.Phase == "Pending" {
		app.Status.Phase = "Creating"
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile ConfigMaps
	for _, cm := range app.Spec.ConfigMaps {
		if err := r.reconcileConfigMap(ctx, &app, cm); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Secrets
	for _, secret := range app.Spec.Secrets {
		if err := r.reconcileSecret(ctx, &app, secret); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Database PVC
	if err := r.reconcileDatabasePVC(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Database Deployment
	if err := r.reconcileDatabaseDeployment(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile WebApp Deployment
	if err := r.reconcileWebAppDeployment(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	// Update status to Ready if all resources are created
	if app.Status.Phase == "Creating" {
		app.Status.Phase = "Ready"
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *TeamOnboardingAppReconciler) reconcileConfigMap(ctx context.Context, app *onboardingv1alpha1.TeamOnboardingApp, cmSpec onboardingv1alpha1.ConfigMapSpec) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmSpec.Name,
			Namespace: app.Spec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = cmSpec.Data
		return controllerutil.SetControllerReference(app, cm, r.Scheme)
	})

	return err
}

func (r *TeamOnboardingAppReconciler) reconcileSecret(ctx context.Context, app *onboardingv1alpha1.TeamOnboardingApp, secretSpec onboardingv1alpha1.SecretSpec) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretSpec.Name,
			Namespace: app.Spec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.StringData = secretSpec.StringData
		return controllerutil.SetControllerReference(app, secret, r.Scheme)
	})

	return err
}

func (r *TeamOnboardingAppReconciler) reconcileDatabasePVC(ctx context.Context, app *onboardingv1alpha1.TeamOnboardingApp) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-db-data", app.Spec.TeamName),
			Namespace: app.Spec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		pvc.Spec.StorageClassName = &app.Spec.Database.Storage.StorageClassName
		pvc.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(app.Spec.Database.Storage.Size),
		}
		return controllerutil.SetControllerReference(app, pvc, r.Scheme)
	})

	return err
}

func (r *TeamOnboardingAppReconciler) reconcileDatabaseDeployment(ctx context.Context, app *onboardingv1alpha1.TeamOnboardingApp) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-db", app.Spec.TeamName),
			Namespace: app.Spec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		deploy.Spec.Replicas = &app.Spec.Database.Replicas
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": fmt.Sprintf("%s-db", app.Spec.TeamName),
			},
		}
		deploy.Spec.Template.ObjectMeta.Labels = map[string]string{
			"app": fmt.Sprintf("%s-db", app.Spec.TeamName),
		}
		deploy.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  "database",
				Image: app.Spec.Database.Image,
				Resources: app.Spec.Database.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "data",
						MountPath: "/var/lib/postgresql/data",
					},
				},
			},
		}
		deploy.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-db-data", app.Spec.TeamName),
					},
				},
			},
		}
		return controllerutil.SetControllerReference(app, deploy, r.Scheme)
	})

	return err
}

func (r *TeamOnboardingAppReconciler) reconcileWebAppDeployment(ctx context.Context, app *onboardingv1alpha1.TeamOnboardingApp) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-webapp", app.Spec.TeamName),
			Namespace: app.Spec.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		deploy.Spec.Replicas = &app.Spec.WebApp.Replicas
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": fmt.Sprintf("%s-webapp", app.Spec.TeamName),
			},
		}
		deploy.Spec.Template.ObjectMeta.Labels = map[string]string{
			"app": fmt.Sprintf("%s-webapp", app.Spec.TeamName),
		}
		deploy.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:  "webapp",
				Image: app.Spec.WebApp.Image,
				Resources: app.Spec.WebApp.Resources,
				Env: []corev1.EnvVar{
					{
						Name:  "TEAM_NAME",
						Value: app.Spec.TeamName,
					},
					{
						Name:  "ENVIRONMENT",
						Value: app.Spec.Environment,
					},
				},
			},
		}
		return controllerutil.SetControllerReference(app, deploy, r.Scheme)
	})

	return err
}

func (r *TeamOnboardingAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&onboardingv1alpha1.TeamOnboardingApp{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}