job "mongodb" {
  datacenters = ["dc1"]
  type        = "service"

  group "mongodb" {
    count = 1

    restart {
      attempts = 10
      interval = "15m"
      delay    = "30s"
      mode     = "fail"
    }

    network {
      mode = "host"
    }

    task "mongodb" {
      driver = "docker"

      config {
        image        = "mongo:7.0"
        network_mode = "host"
        args         = ["--bind_ip_all"]
      }

      env {
        MONGO_INITDB_ROOT_USERNAME = "admin"
        MONGO_INITDB_ROOT_PASSWORD = "password"
      }

      resources {
        cpu    = 300
        memory = 256
      }

      kill_signal  = "SIGINT"
      kill_timeout = "30s"
    }
  }
}
