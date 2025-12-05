# Cloud Run Worker Deployment with Pub/Sub Push
#
# This module deploys the worker service to Cloud Run with:
# - Autoscaling (0 to 1000 instances)
# - Pub/Sub push subscription for serverless scaling
# - IAM bindings for Pub/Sub invoker

variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

variable "worker_image" {
  description = "Docker image for the worker service"
  type        = string
}

variable "db_connection_name" {
  description = "Cloud SQL connection name"
  type        = string
}

variable "redis_host" {
  description = "Memorystore Redis host"
  type        = string
}

variable "pubsub_topic" {
  description = "Pub/Sub topic name"
  type        = string
  default     = "crawlify-tasks"
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 1000
}

# Cloud Run service for worker
resource "google_cloud_run_v2_service" "worker" {
  name     = "crawlify-worker"
  location = var.region
  
  template {
    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }
    
    containers {
      image = var.worker_image
      
      resources {
        limits = {
          cpu    = "2"
          memory = "4Gi"
        }
      }
      
      # Environment variables
      env {
        name  = "GCP_PUBSUB_MODE"
        value = "push"
      }
      
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
      
      env {
        name  = "DATABASE_HOST"
        value = "/cloudsql/${var.db_connection_name}"
      }
      
      env {
        name  = "REDIS_HOST"
        value = var.redis_host
      }
      
      # Liveness probe
      liveness_probe {
        http_get {
          path = "/health"
        }
        initial_delay_seconds = 10
        period_seconds        = 30
      }
      
      # Startup probe
      startup_probe {
        http_get {
          path = "/health"
        }
        initial_delay_seconds = 5
        period_seconds        = 5
        failure_threshold     = 30
      }
    }
    
    # Cloud SQL connection
    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [var.db_connection_name]
      }
    }
    
    # Increase timeout for browser automation tasks
    timeout = "300s"
    
    # Service account
    service_account = google_service_account.worker.email
  }
  
  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

# Service account for worker
resource "google_service_account" "worker" {
  account_id   = "crawlify-worker"
  display_name = "Crawlify Worker Service Account"
}

# IAM: Allow Pub/Sub to invoke Cloud Run
resource "google_cloud_run_service_iam_member" "pubsub_invoker" {
  location = google_cloud_run_v2_service.worker.location
  service  = google_cloud_run_v2_service.worker.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.pubsub_invoker.email}"
}

# Service account for Pub/Sub push
resource "google_service_account" "pubsub_invoker" {
  account_id   = "pubsub-invoker"
  display_name = "Pub/Sub Cloud Run Invoker"
}

# Pub/Sub subscription with push to Cloud Run
resource "google_pubsub_subscription" "worker_push" {
  name  = "crawlify-tasks-push-sub"
  topic = "projects/${var.project_id}/topics/${var.pubsub_topic}"
  
  push_config {
    push_endpoint = "${google_cloud_run_v2_service.worker.uri}/tasks/push"
    
    oidc_token {
      service_account_email = google_service_account.pubsub_invoker.email
      audience              = google_cloud_run_v2_service.worker.uri
    }
    
    # Batching for efficiency
    no_wrapper {
      write_metadata = true
    }
  }
  
  # Retry policy
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }
  
  # Acknowledgement deadline matches Cloud Run timeout
  ack_deadline_seconds = 300
  
  # Expiration policy (never expire)
  expiration_policy {
    ttl = ""
  }
  
  # Dead letter policy for failed messages
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 5
  }
}

# Dead letter topic for failed tasks
resource "google_pubsub_topic" "dead_letter" {
  name = "crawlify-tasks-dead-letter"
}

# IAM: Worker can access Cloud SQL
resource "google_project_iam_member" "worker_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# IAM: Worker can access Cloud Storage
resource "google_project_iam_member" "worker_storage" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# IAM: Worker can publish to Pub/Sub (for discovered URLs)
resource "google_project_iam_member" "worker_pubsub" {
  project = var.project_id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Outputs
output "worker_url" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.worker.uri
}

output "push_subscription" {
  description = "Pub/Sub push subscription name"
  value       = google_pubsub_subscription.worker_push.name
}
