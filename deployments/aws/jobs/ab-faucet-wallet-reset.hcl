job "[[ .gt_environment ]]-ab-faucet-wallet-reset" {
    datacenters = ["[[ .nomad_datacenter ]]"]
    type = "batch"

    constraint {
        attribute = "${meta.group}"
        value = "apps_faucet"
    }

    meta {
        version = "[[ .alphabill_version ]]"
    }

    group "app" {
        constraint {
            operator  = "distinct_hosts"
            value     = "true"
        }

        volume "faucet" {
            type      = "host"
            source    = "faucet"
            read_only = false
        }

        task "ab-faucet-wallet-reset" {
            driver = "exec"

            artifact {
                source = "http://nexus.[[ .gt_domain ]]:8081/repository/binary-raw/alphabill/${NOMAD_META_version}.tar.gz"
                destination = "local/alphabill"
                mode = "file"
            }

            artifact {
                source = "http://nexus.[[ .gt_domain ]]:8081/repository/binary-raw/alphabill-spend-initial-bill/${NOMAD_META_version}.tar.gz"
                destination = "local/alphabill-spend-initial-bill"
                mode = "file"
            }

            config {
                command = "sh"
                args = ["/local/faucet-wallet-reset.sh"]
            }

            volume_mount {
                volume      = "faucet"
                destination = "/faucet"
            }

            template {
                data = <<EOH
[[ fileContents "jobs/scripts/faucet-wallet-reset.sh" ]]
EOH
                destination = "local/faucet-wallet-reset.sh"
                perms = "755"
            }

            [[ indent 12 (fileContents "jobs/common/logger-config.hcl") ]]

            resources {
                cpu     = 100
                memory  = 100
            }
        }

[[ indent 8 (fileContents "jobs/common/restart.hcl") ]]
    }
}
