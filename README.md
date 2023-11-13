# Demo 0: Multi-cloud Kubernetes
```
Using Terraforms cloud-agnostic capabilities, the following was setup as part of multi-cloud strategy:
1. Provisioning Kubernetes clusters in both AWS and Azure was done using their respective providers for AKS in Azure and EKS in AWS.
   > Comment aws_eks_addon in main.tf till EKS build is complete and then uncomment and rerun 't apply'
2. Consul deployed and Consul federation configured with Mesh gateways across 2 cloud clusters using the Helm providers
3. Microservices will then be deployed across 2 clusters and shown to function, thereby verifying federation.
4. Use alias k=kubectl (shell) or set-alias -Name k -Value kubectl (PS)
5. Use alias t=terraform (shell) or set-alias -Name t -Value terraform (PS)
6. Install kubectx and setup EKS and AKS profiles as below:
    > aws eks update-kubeconfig --region us-east-2 --name <EKS cluster>
    > kubectx eks=<EKS ARN>
    > kubectl cluster-info # to verify

    > az aks get-credentials --resource-group <AKS Resource Group> --name <AKS Cluster Name>
    > kubectx aks=<AKS Cluster Name>
    > kubectl cluster-info
```
![image1](https://github.com/jvargh/consul-demo/assets/3197295/d2c278d6-435c-4eca-a1bf-ff0177624307)


#  

# Demo 1: Consul setup

## 1. Base Install steps
```
1. Use "t init --upgrade" to clear old tf state and possibly delete the tfstate file.
2. cd demo1-consulsetup; Run "terraform apply --auto-approve" with content in 'proxy_defaults.tf' commented.
```

## 2. Proxy Install steps
```
1. Uncomment content in 'proxy_defaults.tf' and run "terraform apply --auto-approve".
```

## 3. View Console-UI in browser
```
# cmd below uses hostname. change to ip if using hostname returns null response
kubectx eks # since EKS is Primary Consul DC (dc1)
export CONSUL_HTTP_ADDR=https://$(kubectl get services/consul-ui -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
export CONSUL_HTTP_SSL_VERIFY=false
echo $CONSUL_HTTP_ADDR 
```
Open URL \$CONSUL_HTTP_ADDR in browser


## 4. Using Consul CLI
```
# Show Console servers
> consul members -wan
Node Address Status Type Build Protocol DC Partition Segment
consul-server-0.dc1 10.0.1.98:8302 alive server 1.16.0 2 dc1 default \<all\>
consul-server-0.dc2 10.244.1.45:8302 alive server 1.16.0 2 dc2 default \<all\>

# Show registered services
> consul catalog services
consul
mesh-gateway
```

## 5. Use Kubectl CLI 
```
# Switch context
> kubectx eks

# Show nodes the pods run on
> k get nodes

> k get pods -o wide | grep consul
consul-connect-injector-556d9789cc-gxt2j 1/1 Running 10.0.1.51 ip-10-xxx.us-east-2.compute.internal
consul-mesh-gateway-c77f4bfcb-5kqkm 1/1 Running 10.0.1.36 ip-10-xxx.us-east-2.compute.internal
consul-server-0 1/1 Running 10.0.1.227 ip-10-xxx.us-east-2.compute.internal 
consul-webhook-cert-manager-656f4db796-sprhr 1/1 Running 10.0.1.209 ip-10-xxx.us-east-2.compute.internal
```

## 6. Delete on completion
```
t destroy --auto-approve
```

# Demo 2: Install Front End and Back End Application

## 0. Install steps
```
1. cd demo2-countingsvc; Run "terraform apply --auto-approve"
2. Install completes of FrontEnd and BackEnd in dc1 and install of FrontEnd and BackEnd in dc2.
```

## 1. Check EKS services
```
kubectx eks

# shows counting deployment pods and dashboard pods
k get pods # notice 2/2 indicating Consul Proxy sidecar container in addition to Application container
k get pod/dashboard -o yaml | grep -n -A 3 connect-inject-status # status=injected due to Consul dc config with connectInject enabled=true 
```
![image2](https://github.com/jvargh/consul-demo/assets/3197295/ec484091-6c4a-498d-887d-017a7391d869)
```
> kubectl get pod <counting pod name> -o=jsonpath='{.spec.containers[*].name}'
counting consul-dataplane

# Logs on counting pod that show update of Consul servers
kubectl logs <counting pod name> -c consul-dataplane | grep consul

# Shows registered services
consul catalog services
```

## 2. Check AKS services
```
kubectx aks

# shows counting deployment pods and dashboard pods
k get pods 
```
![image3](https://github.com/jvargh/consul-demo/assets/3197295/f9d47fda-0e21-4971-82dc-a926abdd61cc)

## 3. Connect to Dashboard App using UI
```
# Verify container port for Dashboard pod
k get pod/dashboard -o yaml | grep -n -A 3 containerPort

kubectl port-forward dashboard 80:9002 --context eks 
<http://localhost:80/>

kubectl port-forward dashboard 81:9002 --context aks
<http://localhost:81/>
```

# Demo 2: Failover 

## 1. Bring down backend app in Primary Consul DC
```
# Scaled down counting backend pod to 0 on EKS
kubectl scale deployment.apps/counting --replicas=0 --context eks

# Verify if counting pod is up. It shouldn't be there
k get pods --context eks
```
Note how the FE Dashboard shows **-1** on EKS dashboard view in browser

## 2. Deploy Consul Service Resolver 
```
# If old entry exists in Consul Datacenter (dc2) then delete
consul config delete -kind service-resolver -name counting -datacenter dc2
consul config list -kind service-resolver

# Apply config
cd demo2-countingsvc; kubectl apply -f service-resolver.yaml

# Verify ServiceResolver sync and UP state
k describe serviceresolver
k get serviceresolver
```

## 3. Test Consul Service Resolver functionality 

0\. Run kubectx eks

1\. Use kubectl to scale down, effectively bringing Primary backend down. 
```
kubectl scale deployment.apps/counting --replicas=0 --context eks
```
When service Counting has replica=0 or no pods, then Dashboard in Primary will connect to Counting service in Secondary DC. Result=\> both UIs for Primary and Secondary should reflect same values

Fig shows result of scale down. Result is that Dashboard UI for dc1 should reflect dc2 values 
Step 1: Counting service goes down (no pods)
Step 2: Dashboard in Primary will connect to Counting service in dc2. 

![image](https://github.com/jvargh/consul-demo/assets/3197295/61e0e60c-059e-446c-86fb-d4c0a9ab218c)

2\. Use kubectl to scale up, effectively bringing Primary backend up. 
```
kubectl scale deployment.apps/counting --replicas=1 --context eks
```
When service Counting is up (has at least 1 pod), dc1 Dashboard will start count from 1. Both UIs should show unique counts.


## 4. Delete on completion
```
t destroy --auto-approve
```

# Demo 3: Intentions for services communication control

## 0. Install steps
```
cd demo3; k apply -f .
```

## 1. Test Intentions
a. Verify static server and client are up
```
k get pods | grep static
```
![image](https://github.com/jvargh/consul-demo/assets/3197295/2210280f-a122-490f-9ac1-9d2352dcd52a)


b.  Service Intention now ALLOWS static-client to communicate with static-server.
```
> kubectl exec deploy/static-client -c static-client -- curl -s http://static-server
# returns "Hello World"
```

c.  In Service Intention file change Allow to Deny. This disables static-client to communicate with static-server.
```
> k apply -f client-to-server-intention.yaml

$ kubectl exec deploy/static-client -c static-client -- curl -s http://static-server
returns "command terminated with exit code 52"
```

## 2. Delete on completion
```
k delete -f  .
```

#  

# References

[Talk: Multi-Cloud WAN Federation with Consul and
Kubernetes](https://www.youtube.com/watch?v=acyxtR_3PXo&ab_channel=HashiCorp)

\- Traffic cloud to cloud goes through a common data plane. So you can control what app to app traffic goes through, which services can talk to each other. In Consul this can be done through Intentions though to
Layer 7.

\- Consul across clouds is done through Federation. Federation is act of
joining 2 or more Consul DCs.

\- Federated clusters allow services in all DCs to talk to each other
via Service Mesh

\- Using UI you can view Mesh GWs and LAN federations connected to it

## Annotations used for connectivity: 
```
consul.hashicorp.com/connect-inject-status: injected

# connect backend upstream svc to localhost of frontend. Used in Multi-DC. Need to mention service and dc
consul.hashicorp.com/connect-service-upstreams: counting:9001:dc2
```
