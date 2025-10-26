# Get Metrics (via kubectl top):
    Note: Ensure metric-server deployment is running in kube-system namespace.
    kubectl top pod nginx-sample-deployment-96b9d695-n8msg
    k top pods  --all-namespaces

# Get Metrics (via API's):
    kubectl get --raw /apis/metrics.k8s.io/ | jq .
    kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/default/pods | jq .
    kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes | jq .

# Prometheus Installation Output:
    kube-prometheus-stack has been installed. Check its status by running:
    kubectl --namespace monitoring get pods -l "release=kube-prometheus-stack"

    Get Grafana 'admin' user password by running:

    kubectl --namespace monitoring get secrets kube-prometheus-stack-grafana -o jsonpath="{.data.admin-password}" | base64 -d ; echo

    Access Grafana local instance:

    export POD_NAME=$(kubectl --namespace monitoring get pod -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=kube-prometheus-stack" -oname)
    kubectl --namespace monitoring port-forward $POD_NAME 3000

    Visit https://github.com/prometheus-operator/kube-prometheus for instructions on how to create & configure Alertmanager and Prometheus instances using the Operator.
    ram@ram-thinkPad-e14:~/github.com/practicelabs/operators$ 

# Port Forwarding:
    Prometheus:
        k  port-forward svc/kube-prometheus-stack-prometheus -n monitoring 9090:9090

    Grafana:
        export POD_NAME=$(k --namespace monitoring get pod -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=kube-prometheus-stack" -oname)
        k --namespace monitoring port-forward $POD_NAME 3000

# Port Forwarding Summary:
    [1]   Running                 microk8s kubectl port-forward svc/sample-metrics-svc 8080:8080 &
    [2]-  Running                 microk8s kubectl port-forward svc/kube-prometheus-stack-prometheus -n monitoring 9090:9090 &
    [3]+  Running                 microk8s kubectl --namespace monitoring port-forward $POD_NAME 3000 &

# Operator Flow:

flowchart TD
    subgraph Manager
        A[main.go] --> B[Controller Setup]
        B --> C[Reconciler]
    end

    C -->|Fetch CR| D[NginxAutoscaler CRD]
    C -->|Fetch| E[Target Deployment]
    C -->|Query| F[Prometheus API]
    F -->|Return Metrics| C
    C -->|Scale| E
    C -->|Update| D



