job "[[ .gt_environment ]]-ab-root" {
    datacenters = ["[[ .nomad_datacenter ]]"]
    type = "service"

    constraint {
        attribute = "${meta.group}"
        value = "apps_root_chain"
    }

    meta {
        version = "[[ .alphabill_version ]]"
    }

    group "app" {
        count = 1

        network {
            port "grpc" { static = [[ .ab_rootchain_port ]] }
        }

        service {
            port = "grpc"
            check {
                type = "tcp"
                interval = "10s"
                timeout  = "2s"
            }
        }

        task "ab-root" {
            driver = "exec"

            artifact {
                source = "http://nexus.[[ .gt_domain ]]:8081/repository/binary-raw/alphabill/${NOMAD_META_version}.tar.gz"
                destination = "local/alphabill"
                mode = "file"
            }

            config {
                command = "local/alphabill"
                args = [
                    "root",
                    "--address", "/ip4/${NOMAD_IP_grpc}/tcp/${NOMAD_PORT_grpc}",
                    "--genesis-file", "/local/root-genesis.json",
                    "--key-file", "/secrets/keys.json",
                    "--home", "${NOMAD_TASK_DIR}",
                ]
            }

            template {
                data = "{{ key \"[[ .gt_environment ]]/rootchain/rootchain/keys.json\" }}"
                destination = "secrets/keys.json"
                change_mode = "noop"
            }

            template {
                data = "{{ key \"[[ .gt_environment ]]/rootchain/rootchain/root-genesis.json\" }}"
                destination = "local/root-genesis.json"
                change_mode = "noop"
            }

            [[ indent 12 (fileContents "jobs/common/logger-config.hcl") ]]

            resources {
                cpu     = 2000
                memory  = 1000
                memory_max = 1500
            }
        }

[[ indent 8 (fileContents "jobs/common/restart.hcl") ]]
    }
}
