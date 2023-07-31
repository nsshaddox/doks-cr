package controller

import (
	"context"
	"fmt"

	// "runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myappv1 "my.domain/chatgpt/api/v1"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.my.domain,resources=apps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.my.domain,resources=apps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.my.domain,resources=apps/finalizers,verbs=update

func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the App instance
	app := &myappv1.App{}
	err := r.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Define a new Pod object for frontend
	pod := newPodForCR(app, "frontend")

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new Pod", "Namespace", pod.Namespace, "Name", pod.Name)
			err = r.Create(ctx, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// Error reading the object - requeue the request.
			return ctrl.Result{}, err
		}
	}

	// Repeat the process for the backend
	pod = newPodForCR(app, "backend")

	found = &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new Pod", "Namespace", pod.Namespace, "Name", pod.Name)
			err = r.Create(ctx, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// Error reading the object - requeue the request.
			return ctrl.Result{}, err
		}
	}

	// Create frontend service
	service := newServiceForCR(app, "frontend")
	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new Service", "Namespace", service.Namespace, "Name", service.Name)
			err = r.Create(ctx, service)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// Error reading the object - requeue the request.
			return ctrl.Result{}, err
		}
	}

	// // Repeat the process for the backend
	// service = newServiceForCR(app, "backend")
	// foundService = &corev1.Service{}
	// err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	// if err != nil {
	// 	if errors.IsNotFound(err) {
	// 		log.Info("Creating a new Service", "Namespace", service.Namespace, "Name", service.Name)
	// 		err = r.Create(ctx, service)
	// 		if err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	} else {
	// 		// Error reading the object - requeue the request.
	// 		return ctrl.Result{}, err
	// 	}
	// }

	// No need to requeue, we're done
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myappv1.App{}).
		Complete(r)
}

func newPodForCR(cr *myappv1.App, component string) *corev1.Pod {
	labels := map[string]string{
		"app":       cr.Name,
		"component": component,
	}

	image := ""
	if component == "frontend" {
		image = cr.Spec.Frontend.Image
	} else if component == "backend" {
		image = cr.Spec.Backend.Image
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", cr.Name, component),
			Namespace: cr.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, myappv1.GroupVersion.WithKind("App")),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  fmt.Sprintf("%s-container", component),
					Image: image,
					Ports: []corev1.ContainerPort{{
						ContainerPort: 80,
					}},
				},
			},
		},
	}
}

func newServiceForCR(cr *myappv1.App, component string) *corev1.Service {
	labels := map[string]string{
		"app":       cr.Name,
		"component": component,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-service", cr.Name, component),
			Namespace: cr.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, myappv1.GroupVersion.WithKind("App")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}
