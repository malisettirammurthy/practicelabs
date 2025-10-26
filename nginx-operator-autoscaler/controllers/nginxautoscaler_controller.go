package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	prom "github.com/malisettirammurthy/nginx-operator-autoscaler/internal/prom"
)

var (
	autoscalerGVK = schema.GroupVersionKind{
		Group:   "autoscaler.malisetti.dev",
		Version: "v1alpha1",
		Kind:    "NginxAutoscaler",
	}
)

type reconciler struct {
	client.Client
}

func SetupNginxAutoscalerController(mgr ctrl.Manager) error {
	r := &reconciler{Client: mgr.GetClient()}
	// Watch the CRD using an unstructured object (no codegen needed)
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(autoscalerGVK)
	return ctrl.NewControllerManagedBy(mgr).
		For(u).
		Complete(r)
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("nginxautoscaler", req.NamespacedName)

	// 1) Load CR
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(autoscalerGVK)
	if err := r.Get(ctx, req.NamespacedName, u); err != nil {
		// gone? nothing to do.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	spec := u.Object["spec"].(map[string]interface{})

	// Extract spec
	getStr := func(key, def string) string {
		if v, ok := spec[key].(string); ok && v != "" {
			return v
		}
		return def
	}
	getF64 := func(key string, def float64) float64 {
		if v, ok := spec[key].(float64); ok {
			return v
		}
		return def
	}
	getI32 := func(key string, def int32) int32 {
		if v, ok := spec[key].(int64); ok {
			return int32(v)
		}
		if v, ok := spec[key].(float64); ok {
			return int32(v)
		}
		return def
	}
	parseDur := func(s string, def time.Duration) time.Duration {
		if s == "" {
			return def
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return def
		}
		return d
	}

	targetDeployment := getStr("targetDeployment", "nginx-sample-deployment-2")
	promURL := getStr("promURL", "http://kube-prometheus-stack-prometheus.monitoring.svc:9090")
	pollInterval := parseDur(getStr("pollInterval", "15s"), 15*time.Second)
	cooldown := parseDur(getStr("cooldown", "60s"), 60*time.Second)
	minReplicas := getI32("minReplicas", 2)
	maxReplicas := getI32("maxReplicas", 20)
	targetCPU := getF64("targetCPU", 0.2)   // cores per replica
	targetMem := getF64("targetMem", 300.0) // MiB per replica
	hysteresisPct := getF64("hysteresisPct", 10.0)
	stepLimit := getI32("stepLimit", 5)

	// 2) Load Deployment
	var dep appsv1.Deployment
	key := types.NamespacedName{Namespace: req.Namespace, Name: targetDeployment}
	if err := r.Get(ctx, key, &dep); err != nil {
		logger.Error(err, "failed to get target Deployment", "name", targetDeployment)
		return ctrl.Result{RequeueAfter: pollInterval}, client.IgnoreNotFound(err)
	}

	if dep.Spec.Replicas == nil {
		r1 := int32(1)
		dep.Spec.Replicas = &r1
	}
	current := *dep.Spec.Replicas

	// 3) Query Prometheus (sum across pods of this deployment – by pod prefix)
	prefix := dep.Name + "-"
	cpuQ := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s.*",image!=""}[2m]))`, dep.Namespace, prefix)
	memQ := fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s",pod=~"%s.*",image!=""})`, dep.Namespace, prefix)

	cpu, err := prom.InstantVector(promURL, cpuQ)
	if err != nil {
		logger.Error(err, "prometheus cpu query failed")
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}
	mem, err := prom.InstantVector(promURL, memQ)
	if err != nil {
		logger.Error(err, "prometheus mem query failed")
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}
	totalCPUcores := cpu // seconds/sec → cores
	totalMemMiB := mem / (1024 * 1024)

	// 4) Compute desired replicas
	cpuReplicas := int32(math.Ceil(totalCPUcores / targetCPU))
	memReplicas := int32(math.Ceil(totalMemMiB / targetMem))
	desired := max32(cpuReplicas, memReplicas)
	if desired < minReplicas {
		desired = minReplicas
	}
	if desired > maxReplicas {
		desired = maxReplicas
	}

	// 5) Hysteresis band
	if !outsideBand(current, desired, hysteresisPct) {
		logger.Info("within hysteresis; no scale",
			"current", current, "desired", desired,
			"cpu_cores", fmt.Sprintf("%.3f", totalCPUcores),
			"mem_mib", fmt.Sprintf("%.1f", totalMemMiB))
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}

	// 6) Cooldown: store lastScaleTime in status
	now := time.Now().Format(time.RFC3339)
	lastScaleStr, _, _ := unstructured.NestedString(u.Object, "status", "lastScaleTime")
	if lastScaleStr != "" {
		if t, err := time.Parse(time.RFC3339, lastScaleStr); err == nil {
			if time.Since(t) < cooldown {
				logger.Info("cooldown active; skipping", "cooldown", cooldown)
				return ctrl.Result{RequeueAfter: pollInterval}, nil
			}
		}
	}

	// 7) Rate limit step
	diff := int32(0)
	if desired > current {
		diff = min32(desired-current, stepLimit)
	} else if desired < current {
		diff = -min32(current-desired, stepLimit)
	}
	newReplicas := current + diff
	if newReplicas < minReplicas {
		newReplicas = minReplicas
	}
	if newReplicas > maxReplicas {
		newReplicas = maxReplicas
	}

	// 8) Patch Deployment
	dep.Spec.Replicas = &newReplicas
	if err := r.Update(ctx, &dep); err != nil {
		logger.Error(err, "failed to update replicas")
		return ctrl.Result{RequeueAfter: pollInterval}, err
	}

	// 9) Update CR status
	_ = unstructured.SetNestedField(u.Object, now, "status", "lastScaleTime")
	_ = unstructured.SetNestedField(u.Object, int64(newReplicas), "status", "currentReplicas")
	_ = unstructured.SetNestedField(u.Object, int64(desired), "status", "desiredReplicas")
	if err := r.Status().Update(ctx, u); err != nil {
		logger.Error(err, "failed to update status (will retry later)")
	}

	logger.Info("scaled",
		"from", current, "to", newReplicas, "desired_raw", desired,
		"cpu_cores", fmt.Sprintf("%.3f", totalCPUcores),
		"mem_mib", fmt.Sprintf("%.1f", totalMemMiB))

	return ctrl.Result{RequeueAfter: pollInterval}, nil
}

func outsideBand(current, desired int32, hysteresisPct float64) bool {
	if current == desired {
		return false
	}
	low := float64(current) * (1.0 - hysteresisPct/100.0)
	high := float64(current) * (1.0 + hysteresisPct/100.0)
	return float64(desired) < low || float64(desired) > high
}

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
