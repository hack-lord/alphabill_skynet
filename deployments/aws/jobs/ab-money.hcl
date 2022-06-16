job "[[ .gt_environment ]]-ab-money" {
    datacenters = ["[[ .nomad_datacenter ]]"]
    type = "service"

    constraint {
        attribute = "${meta.group}"
        value = "apps_money_partition"
    }

    meta {
        version = "[[ .alphabill_version ]]"
    }

    group "app" {
        constraint {
            operator  = "distinct_hosts"
            value     = "true"
        }

        count = [[ len .ab_money_partition_ips ]]

        network {
            port "cluster" { static = [[ .ab_partition_port ]] }
            port "api" { static = [[ .ab_partition_api_port ]] }
        }

        service {
            port = "cluster"
            check {
                type = "tcp"
                interval = "10s"
                timeout  = "2s"
            }
        }

        service {
            port = "api"
            check {
                type = "grpc"
                interval = "10s"
                timeout  = "2s"
            }
        }

        task "ab-money" {
            driver = "exec"

            artifact {
                source = "http://nexus.[[ .gt_domain ]]:8081/repository/binary-raw/alphabill/${NOMAD_META_version}.tar.gz"
                destination = "local/alphabill"
                mode = "file"
            }

            config {
                command = "local/alphabill"
                args = [
                    "money",
                    "--address", "/ip4/${NOMAD_IP_cluster}/tcp/${NOMAD_PORT_cluster}",
                    "--genesis", "/local/partition-genesis.json",
                    "--key-file", "/secrets/keys.json",
                    "--peers", "${PEERS}",
                    "--rootchain", "/ip4/[[ .ab_rootchain_ip ]]/tcp/[[ .ab_rootchain_port ]]",
                    "--server-address", ":${NOMAD_PORT_api}",
                    "--home", "${NOMAD_TASK_DIR}",
                ]
            }

            env {
                GENESIS_PATH = "[[ .gt_environment ]]/rootchain/rootchain/partition-genesis-0.json"
                KEYS_PATH = "[[ .gt_environment ]]/money${meta.index}/money/keys.json"
            }

            template {
                data = "{{ key (env \"GENESIS_PATH\") }}"
                destination = "local/partition-genesis.json"
                change_mode = "noop"
            }

            template {
                data = "{{ key (env \"KEYS_PATH\") }}"
                destination = "secrets/keys.json"
                change_mode = "noop"
            }

            template {
                data = "PEERS={{ key \"[[ .gt_environment ]]/money_peers.txt\" }}"
                destination = "local/peers.env"
                env = "true"
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
