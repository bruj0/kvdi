package desktop

import (
	"context"
	"fmt"
	"net"

	"github.com/tinyzimmer/kvdi/pkg/apis/kvdi/v1alpha1"
	"github.com/tinyzimmer/kvdi/pkg/resources"
	"github.com/tinyzimmer/kvdi/pkg/util"
	"github.com/tinyzimmer/kvdi/pkg/util/errors"
	"github.com/tinyzimmer/kvdi/pkg/util/reconcile"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DesktopReconciler struct {
	resources.DesktopReconciler

	client client.Client
	scheme *runtime.Scheme
}

var _ resources.DesktopReconciler = &DesktopReconciler{}

// New returns a new Desktop reconciler
func New(c client.Client, s *runtime.Scheme) resources.DesktopReconciler {
	return &DesktopReconciler{client: c, scheme: s}
}

func (f *DesktopReconciler) Reconcile(reqLogger logr.Logger, instance *v1alpha1.Desktop) error {
	template, err := instance.GetTemplate(f.client)
	if err != nil {
		return err
	}
	cluster, err := instance.GetVDICluster(f.client)
	if err != nil {
		return err
	}
	if err := reconcile.ReconcileCertificate(reqLogger, f.client, newDesktopProxyCert(cluster, instance), true); err != nil {
		return err
	}

	if _, err := reconcile.ReconcilePod(reqLogger, f.client, newDesktopPodForCR(cluster, template, instance)); err != nil {
		return err
	}
	if err := reconcile.ReconcileService(reqLogger, f.client, newHeadlessServiceForCR(cluster, instance)); err != nil {
		return err
	}

	// Wait for the desktop to be ready
	desktopPod := &corev1.Pod{}
	nn := types.NamespacedName{Name: instance.GetName(), Namespace: instance.GetNamespace()}
	if err := f.client.Get(context.TODO(), nn, desktopPod); err != nil {
		return err
	}
	for _, status := range desktopPod.Status.ContainerStatuses {
		if status.State.Running == nil {
			return errors.NewRequeueError("Desktop instance is not yet running", 3)
		}
	}

	// make sure it's resolving
	addr := util.DesktopShortURL(instance)
	reqLogger.Info(fmt.Sprintf("Checking if %s resolves to desktop instance", addr))
	if _, err := net.LookupHost(addr); err != nil {
		reqLogger.Info(err.Error())
		return errors.NewRequeueError("DNS not yet resolving to desktop instance", 3)
	}

	if !instance.Status.Running {
		instance.Status.Running = true
		if err := f.client.Status().Update(context.TODO(), instance); err != nil {
			return err
		}
	}

	return nil
}
