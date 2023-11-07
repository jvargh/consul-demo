###################
## AKS resources ##
###################
data "terraform_remote_state" "aks" {
  backend = "local"
  config = {
    path = "../cloud-aks/terraform.tfstate"
  }
}

provider "azurerm" {
  features {}
}

data "azurerm_kubernetes_cluster" "cluster" {
  name                = data.terraform_remote_state.aks.outputs.kubernetes_cluster_name
  resource_group_name = data.terraform_remote_state.aks.outputs.resource_group_name
}
provider "kubernetes" {
  alias                  = "aks"
  host                   = data.azurerm_kubernetes_cluster.cluster.kube_config.0.host
  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config.0.client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config.0.client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config.0.cluster_ca_certificate)
}

## AKS - Counting App ##
resource "kubernetes_deployment" "aks_counting" {
  provider = kubernetes.aks
  metadata {
    name = "counting"
    labels = {
      "app" = "counting"
    }
  }
  spec {
    replicas = 1
    selector {
      match_labels = {
        "app" = "counting"
      }
    }
    template {
      metadata {
        labels = {
          "app" = "counting"
        }
      }
      spec {
        container {
          image = "hashicorp/counting-service:0.0.2"
          name  = "counting"
          port {
            container_port = 9001
            name           = "http"
          }
        }
      }
    }
  }
}
resource "kubernetes_service" "aks_counting" {
  provider = kubernetes.aks
  metadata {
    name      = "counting"
    namespace = "default"
    labels = {
      "app" = "counting"
    }
  }
  spec {
    selector = {
      "app" = "counting"
    }
    port {
      name        = "http"
      port        = 9001
      target_port = 9001
      protocol    = "TCP"
    }
    type = "ClusterIP"
  }
}

## AKS - Dashboard App ##
resource "kubernetes_pod" "aks_dashboard" {
  provider = kubernetes.aks

  metadata {
    name = "dashboard"
    annotations = {
      "consul.hashicorp.com/connect-service-upstreams" = "counting:9001:dc2"
    }
    labels = {
      "app" = "dashboard"
    }
  }

  spec {
    container {
      image = "hashicorp/dashboard-service:0.0.4"
      name  = "dashboard"

      env {
        name  = "COUNTING_SERVICE_URL"
        value = "http://localhost:9001"
      }

      port {
        container_port = 9002
      }
    }
  }
}

resource "kubernetes_service" "aks_dashboard" {
  provider = kubernetes.aks

  metadata {
    name      = "dashboard"
    namespace = "default"
    labels = {
      "app" = "dashboard"
    }
  }

  spec {
    selector = {
      "app" = "dashboard"
    }
    port {
      port        = 9002
      target_port = 9002      
    }
    # type = "LoadBalancer"
  }
}


##################
# EKS resources ##
##################
data "terraform_remote_state" "eks" {
  backend = "local"
  config = {
    path = "../cloud-eks/terraform.tfstate"
  }
}

provider "aws" {
  region = data.terraform_remote_state.eks.outputs.region
}

data "aws_eks_cluster" "cluster" {
  name = data.terraform_remote_state.eks.outputs.cluster_name
}

provider "kubernetes" {
  alias                  = "eks"
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    args        = ["eks", "get-token", "--cluster-name", data.aws_eks_cluster.cluster.name]
    command     = "aws"
  }
}

## EKS - Counting App ##
resource "kubernetes_deployment" "eks_counting" {
  provider = kubernetes.eks
  metadata {
    name = "counting"
    labels = {
      "app" = "counting"
    }
  }
  spec {
    replicas = 1
    selector {
      match_labels = {
        "app" = "counting"
      }
    }
    template {
      metadata {
        labels = {
          "app" = "counting"
        }
      }
      spec {
        container {
          image = "hashicorp/counting-service:0.0.2"
          name  = "counting"
          port {
            container_port = 9001
            name           = "http"
          }
        }
      }
    }
  }
}
resource "kubernetes_service" "eks_counting" {
  provider = kubernetes.eks
  metadata {
    name      = "counting"
    namespace = "default"
    labels = {
      "app" = "counting"
    }
  }
  spec {
    selector = {
      "app" = "counting"
    }
    port {
      name        = "http"
      port        = 9001
      target_port = 9001
      protocol    = "TCP"
    }
    type = "ClusterIP"
  }
}

## EKS - Dashboard App ##
resource "kubernetes_pod" "eks_dashboard" {
  provider = kubernetes.eks

  metadata {
    name = "dashboard"
    annotations = {
      "consul.hashicorp.com/connect-service-upstreams" = "counting:9001:dc1"
    }
    labels = {
      "app" = "dashboard"
    }
  }

  spec {
    container {
      image = "hashicorp/dashboard-service:0.0.4"
      name  = "dashboard"

      env {
        name  = "COUNTING_SERVICE_URL"
        value = "http://localhost:9001"
      }

      port {
        container_port = 9002
      }
    }
  }
}

resource "kubernetes_service" "eks_dashboard" {
  provider = kubernetes.eks

  metadata {
    name      = "dashboard"
    namespace = "default"
    labels = {
      "app" = "dashboard"
    }
  }

  spec {
    selector = {
      "app" = "dashboard"
    }
    port {
      port        = 80
      target_port = 9002      
    }
    # type = "LoadBalancer"
  }
}

