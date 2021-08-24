package provider

const (
	GCP   = "google"
	AWS   = "aws"
	AZURE = "azure"
)

const azureBackendTemplate = `terraform {
	backend "azurerm" {
		storage_account_name = {{ .Values.Context.StorageAccount | quote }}
		resource_group_name = {{ .Values.ResourceGroup | quote }}
		container_name = {{ .Values.Bucket | quote }}
		key = "{{ .Values.__CLUSTER__ }}/{{ .Values.Prefix }}/terraform.tfstate"
	}

	required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = "2.57.0"
    }
		kubernetes = {
			source  = "hashicorp/kubernetes"
			version = "~> 2.0.3"
		}
  }
}

provider "azurerm" {
  features {}
}

{{ if .Values.ClusterCreated }}
provider "kubernetes" {
  host                   = {{ .Values.Cluster }}.host
  client_certificate     = base64decode({{ .Values.Cluster }}.client_certificate)
  client_key             = base64decode({{ .Values.Cluster }}.client_key)
  cluster_ca_certificate = base64decode({{ .Values.Cluster }}.cluster_ca_certificate)
}
{{ else }}
data "azurerm_kubernetes_cluster" "cluster" {
  name = {{ .Values.Cluster }}
	resource_group_name = {{ .Values.ResourceGroup | quote }}
}

provider "kubernetes" {
  host                   = data.azurerm_kubernetes_cluster.cluster.kube_config[0].host
  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].cluster_ca_certificate)
}
{{ end }}
`

const gcpBackendTemplate = `terraform {
	backend "gcs" {
		bucket = {{ .Values.Bucket | quote }}
		prefix = "{{ .Values.__CLUSTER__ }}/{{ .Values.Prefix }}"
	}

	required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 3.65.0"
    }
		kubernetes = {
			source  = "hashicorp/kubernetes"
			version = "~> 2.0.3"
		}
  }
}

locals {
	gcp_location  = {{ .Values.Location | quote }}
  gcp_location_parts = split("-", local.gcp_location)
  gcp_region         = "${local.gcp_location_parts[0]}-${local.gcp_location_parts[1]}"
}

provider "google" {
  project = {{ .Values.Project | quote }}
  region  = local.gcp_region
}

data "google_client_config" "current" {}

{{ if .Values.ClusterCreated }}
provider "kubernetes" {
  host = {{ .Values.Cluster }}.endpoint
  cluster_ca_certificate = base64decode({{ .Values.Cluster }}.ca_certificate)
  token = data.google_client_config.current.access_token
}
{{ else }}
data "google_container_cluster" "cluster" {
  name = {{ .Values.Cluster }}
  location = local.gcp_region
}

provider "kubernetes" {
  host = data.google_container_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.google_container_cluster.cluster.master_auth.0.cluster_ca_certificate)
  token = data.google_client_config.current.access_token
}
{{ end }}
`

const awsBackendTemplate = `terraform {
	backend "s3" {
		bucket = {{ .Values.Bucket | quote }}
		key = "{{ .Values.__CLUSTER__ }}/{{ .Values.Prefix }}/terraform.tfstate"
		region = {{ .Values.Region | quote }}
	}

	required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.55.0"
    }
		kubernetes = {
			source  = "hashicorp/kubernetes"
			version = "~> 2.0.3"
		}
  }
}

provider "aws" {
  region = {{ .Values.Region | quote }}
}

data "aws_eks_cluster" "cluster" {
  name = {{ .Values.Cluster }}
}

data "aws_eks_cluster_auth" "cluster" {
  name = {{ .Values.Cluster }}
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}
`