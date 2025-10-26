You’ll be building the project under your personal GitHub path (so future Docker image tags and module paths are consistent).

🧱 Step-by-step setup
# 1️⃣ Create and enter project directory
mkdir -p ~/github.com/malisettirammurthy/db-autoscaler
cd ~/github.com/malisettirammurthy/db-autoscaler

# 2️⃣ Initialize a Go module with your GitHub path
go mod init github.com/malisettirammurthy/db-autoscaler

# 3️⃣ Add controller-runtime and Kubernetes client libraries
go get sigs.k8s.io/controller-runtime@v0.17.3
go get k8s.io/api@v0.29.2
go get k8s.io/apimachinery@v0.29.2
go get k8s.io/client-go@v0.29.2




# This will download and record the missing dependencies
go get go.uber.org/zap@v1.26.0
go get github.com/go-logr/zapr@v1.3.0

# Then tidy up the module references
go mod tidy
go run .


# Use microk8s kube config
export KUBECONFIG=~/.kube/microk8s-config
go run .

🧱 Deployment

# Use Dockerfile and build the image
docker build -t rammurthymalisetti/db-autoscaler:latest .
docker push rammurthymalisetti/db-autoscaler:latest


# Create RBAC from rbac.yaml
kubectl apply -f rbac.yaml

# Create Deployment from deployment.yaml
kubectl apply -f deployment.yaml

# Network Model:

flowchart TB
    subgraph Node1["Node #1"]
        subgraph Pod1["Pod: db-autoscaler-7df7f5cf7b-h72jx"]
            App1["Autoscaler Process\n:8081 (healthz, readyz)"]
            Kubelet1["Kubelet"]
            Kubelet1 -->|HTTP GET /healthz| App1
        end
    end

    subgraph Node2["Node #2"]
        subgraph Pod2["Pod: db-autoscaler-7df7f5cf7b-bm2sd"]
            App2["Autoscaler Process\n:8081 (healthz, readyz)"]
            Kubelet2["Kubelet"]
            Kubelet2 -->|HTTP GET /healthz| App2
        end
    end

    subgraph Node3["Node #3"]
        subgraph Pod3["Pod: db-autoscaler-7df7f5cf7b-vg7mf"]
            App3["Autoscaler Process\n:8081 (healthz, readyz)"]
            Kubelet3["Kubelet"]
            Kubelet3 -->|HTTP GET /readyz| App3
        end
    end

    subgraph Cluster["Kubernetes Cluster"]
        direction TB
        Service["(Optional) ClusterIP Service\nport 8081 -> pods:8081"]
    end

    ext[("Your Terminal\n(curl/port-forward)")] -->|curl localhost:8081/healthz| Service
    Service --> Pod1
    Service --> Pod2
    Service --> Pod3



