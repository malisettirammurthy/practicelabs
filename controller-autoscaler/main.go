package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	server "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// ---------- Config (env) ----------

type Config struct {
	Namespace             string
	DeploymentName        string
	PodLabelSelector      string // e.g. "app=nginx"
	PromURL               string // e.g. "http://kube-prometheus-stack-prometheus.monitoring.svc:9090"
	PollInterval          time.Duration
	Cooldown              time.Duration
	MinReplicas           int32
	MaxReplicas           int32
	TargetCPUPerReplica   float64 // cores per replica you’re willing to sustain (e.g., 0.2 = 200m)
	TargetMemPerReplicaMB float64 // MiB per replica (budget)
	HysteresisPct         float64 // e.g., 10 => need 10% margin to trigger
	ScaleStepLimit        int32   // max replicas to change per decision (e.g., 5)
}

func mustEnv(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func parseDuration(s, def string) time.Duration {
	if s == "" {
		d, _ := time.ParseDuration(def)
		return d
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		d, _ = time.ParseDuration(def)
	}
	return d
}

func parseInt32(s string, def int32) int32 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return def
	}
	return int32(n)
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func loadConfig() Config {
	return Config{
		Namespace:             mustEnv("TARGET_NAMESPACE", "default"),
		DeploymentName:        mustEnv("TARGET_DEPLOYMENT", "nginx-sample-deployment"),
		PodLabelSelector:      mustEnv("POD_SELECTOR", "app=nginx"),
		PromURL:               mustEnv("PROM_URL", "http://kube-prometheus-stack-prometheus.monitoring.svc:9090"),
		PollInterval:          parseDuration(os.Getenv("POLL_INTERVAL"), "15s"),
		Cooldown:              parseDuration(os.Getenv("COOLDOWN"), "60s"),
		MinReplicas:           parseInt32(os.Getenv("MIN_REPLICAS"), 2),
		MaxReplicas:           parseInt32(os.Getenv("MAX_REPLICAS"), 50),
		TargetCPUPerReplica:   parseFloat(os.Getenv("TARGET_CPU_CORES"), 0.2), // 200m per replica
		TargetMemPerReplicaMB: parseFloat(os.Getenv("TARGET_MEM_MIB"), 300.0), // 300 MiB per replica
		HysteresisPct:         parseFloat(os.Getenv("HYSTERESIS_PCT"), 10.0),  // 10%
		ScaleStepLimit:        parseInt32(os.Getenv("SCALE_STEP_LIMIT"), 5),
	}
}

// ---------- Prometheus tiny client ----------

type promAPIResp struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			// For instant queries, "value"; for range, "values"
			Value  []interface{}   `json:"value"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func promInstantQuery(promURL, query string) (promAPIResp, error) {
	u, _ := url.Parse(promURL)
	u.Path = "/api/v1/query"
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return promAPIResp{}, err
	}
	defer resp.Body.Close()

	var out promAPIResp
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

// ---------- Reconciler ----------

type Reconciler struct {
	k8s    client.Client
	cfg    Config
	lastAt time.Time // last decision time (simple cooldown)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Only handle our target Deployment key; ignore others
	targetKey := types.NamespacedName{Namespace: r.cfg.Namespace, Name: r.cfg.DeploymentName}
	if req.NamespacedName != targetKey {
		return ctrl.Result{}, nil
	}

	// Read current scale
	var dep appsv1.Deployment
	if err := r.k8s.Get(ctx, targetKey, &dep); err != nil {
		return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, client.IgnoreNotFound(err)
	}
	if dep.Spec.Replicas == nil {
		r := int32(1)
		dep.Spec.Replicas = &r
	}

	current := *dep.Spec.Replicas

	// Query Prometheus for workload demand
	// 1) CPU total cores used by pods matching selector over last 2m
	// We rely on cAdvisor metric container_cpu_usage_seconds_total
	// NOTE: adjust job/service labels if needed; here we filter by pod label using Kubernetes relabeling from kube-state-metrics > pod labels may not be present.
	// Safer: sum by (pod) then join via label selector using kube-state-metrics.
	// For simplicity, we filter by namespace + match on pod name prefix of deployment:
	prefix := dep.Name + "-"
	cpuQ := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s.*",image!=""}[2m]))`, r.cfg.Namespace, prefix)
	memQ := fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s",pod=~"%s.*",image!=""})`, r.cfg.Namespace, prefix)

	cpuResp, err := promInstantQuery(r.cfg.PromURL, cpuQ)
	if err != nil || cpuResp.Status != "success" {
		logger.Error(err, "prometheus cpu query failed")
		return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, nil
	}
	memResp, err := promInstantQuery(r.cfg.PromURL, memQ)
	if err != nil || memResp.Status != "success" {
		logger.Error(err, "prometheus mem query failed")
		return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, nil
	}

	totalCPUcores := 0.0
	if len(cpuResp.Data.Result) > 0 && len(cpuResp.Data.Result[0].Value) == 2 {
		if s, ok := cpuResp.Data.Result[0].Value[1].(string); ok {
			totalCPUcores, _ = strconv.ParseFloat(s, 64) // cores/sec (rate), effectively cores used
		}
	}

	totalMemBytes := 0.0
	if len(memResp.Data.Result) > 0 && len(memResp.Data.Result[0].Value) == 2 {
		if s, ok := memResp.Data.Result[0].Value[1].(string); ok {
			totalMemBytes, _ = strconv.ParseFloat(s, 64)
		}
	}
	totalMemMiB := totalMemBytes / (1024.0 * 1024.0)

	// Compute desired replicas based on the stricter of CPU vs Mem demands
	// replicas_cpu = ceil(totalCPU / targetCPUPerReplica)
	// replicas_mem = ceil(totalMemMiB / targetMemPerReplicaMB)
	desiredCPU := int32(math.Ceil(totalCPUcores / r.cfg.TargetCPUPerReplica))
	desiredMem := int32(math.Ceil(totalMemMiB / r.cfg.TargetMemPerReplicaMB))
	desired := max32(desiredCPU, desiredMem)
	if desired < r.cfg.MinReplicas {
		desired = r.cfg.MinReplicas
	}
	if desired > r.cfg.MaxReplicas {
		desired = r.cfg.MaxReplicas
	}

	// Hysteresis: change only if outside ±H%
	changeNeeded := shouldScale(current, desired, r.cfg.HysteresisPct)

	// Cooldown
	if time.Since(r.lastAt) < r.cfg.Cooldown {
		logger.Info("cooldown active; skipping", "current", current, "desired", desired)
		return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, nil
	}

	if !changeNeeded {
		logger.Info("within hysteresis; no scale", "current", current, "desired", desired,
			"cpu_cores", fmt.Sprintf("%.3f", totalCPUcores), "mem_mib", fmt.Sprintf("%.1f", totalMemMiB))
		return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, nil
	}

	// Rate limit step
	diff := int32(0)
	if desired > current {
		diff = min32(desired-current, r.cfg.ScaleStepLimit)
	} else if desired < current {
		diff = -min32(current-desired, r.cfg.ScaleStepLimit)
	}
	newReplicas := current + diff
	newReplicas = clamp32(newReplicas, r.cfg.MinReplicas, r.cfg.MaxReplicas)

	dep.Spec.Replicas = &newReplicas
	if err := r.k8s.Update(ctx, &dep); err != nil {
		logger.Error(err, "failed to update replicas")
		return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, err
	}

	r.lastAt = time.Now()
	logger.Info("scaled", "from", current, "to", newReplicas,
		"desired_raw", desired, "cpu_cores", fmt.Sprintf("%.3f", totalCPUcores),
		"mem_mib", fmt.Sprintf("%.1f", totalMemMiB))

	return ctrl.Result{RequeueAfter: r.cfg.PollInterval}, nil
}

/*
// Simple version of reconcile loop:
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 1️⃣ Get the target Deployment
	var dep appsv1.Deployment
	if err := r.k8s.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2️⃣ Query Prometheus for current CPU and memory usage
	cpuUsage := r.queryPrometheus("sum(rate(container_cpu_usage_seconds_total{pod=~...}[2m]))")
	memUsage := r.queryPrometheus("sum(container_memory_working_set_bytes{pod=~...})")

	// 3️⃣ Calculate desired replicas based on thresholds
	cpuReplicas := ceil(cpuUsage / cfg.TargetCPUPerReplica)
	memReplicas := ceil(memUsage / cfg.TargetMemPerReplica)
	desired := max(cpuReplicas, memReplicas)

	// 4️⃣ Apply hysteresis (don’t scale if within 10% band)
	if abs(desired-currentReplicas)/currentReplicas < HysteresisPct {
		return ctrl.Result{RequeueAfter: PollInterval}, nil
	}

	// 5️⃣ Apply cooldown (avoid frequent scaling)
	if time.Since(r.lastScaleTime) < cfg.Cooldown {
		return ctrl.Result{RequeueAfter: PollInterval}, nil
	}

	// 6️⃣ Clamp between min and max replicas
	desired = clamp(desired, MinReplicas, MaxReplicas)

	// 7️⃣ Update the Deployment spec
	dep.Spec.Replicas = &desired
	if err := r.k8s.Update(ctx, &dep); err != nil {
		return ctrl.Result{}, err
	}

	// 8️⃣ Log and requeue
	r.lastScaleTime = time.Now()
	return ctrl.Result{RequeueAfter: PollInterval}, nil
}
*/

func shouldScale(current, desired int32, hysteresisPct float64) bool {
	if current == desired {
		return false
	}
	low := float64(current) * (1.0 - hysteresisPct/100.0)
	high := float64(current) * (1.0 + hysteresisPct/100.0)
	return float64(desired) < low || float64(desired) > high
}

func clamp32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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

// Setup controller: watch only the target Deployment key.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// We’ll use an Index + Named watch; simplest is to Requeue our single key on a ticker.
	// But controller-runtime wants a source; we’ll watch Deployments and filter in Reconcile.
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	cfg := loadConfig()
	ctrl.SetLogger(zap.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: ":8081",
	})

	if err != nil {
		panic(err)
	}

	r := &Reconciler{k8s: mgr.GetClient(), cfg: cfg}
	if err := r.SetupWithManager(mgr); err != nil {
		panic(err)
	}

	_ = mgr.AddHealthzCheck("ping", healthz.Ping)
	_ = mgr.AddReadyzCheck("ping", healthz.Ping)

	fmt.Println("db-autoscaler starting for:",
		cfg.Namespace+"/"+cfg.DeploymentName,
		"selector="+cfg.PodLabelSelector,
		"prom="+cfg.PromURL,
		"poll="+cfg.PollInterval.String())

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(err)
	}
}
