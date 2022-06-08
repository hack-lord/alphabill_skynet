# AWS deployment

## Folder structure

* dev/ - Holds variables and configuration related to dev environment.
* jobs/ - Common job templates.
* roles/ - Ansible helper roles.

## Preparing required variables

1. Specify Ansible vault path (or use --ask-vault-password for ansible commands each time)

    ```console
    export ANSIBLE_VAULT_PASSWORD_FILE=..
    ```

## Deployment via Gitlab

1. When an environment has not been deployed, run manual pipeline <https://gitdc.ee.guardtime.com/alphabill/alphabill/-/pipelines/new>
with the following parameters:

    * Branch - select the revision you would like to deploy
    * ACTION - reset
    * TARGET_ENVIRONMENT - environment name

2. When updating the environment, run deploy job that is available after artifacts have been published.

## Manual deployment

Use these commands to debug deployment scripts or manually provision a new environment. The commands shown below are also integrated with CICD pipeline.

### Preparing Consul

Consul K/V store needs to be prepared in advance and it's the only manual step when preparing a new environment.
These hold secrets that are used when jobs are ran via Nomad.

```console
export ENVIRONMENT=<environment name, check deployments/aws for reference>
ansible-playbook deployments/aws/site-deploy-consul.yml -e gt_environment=${ENVIRONMENT}
```

### Publishing a new artifact

```console
export ENVIRONMENT=<environment name, check deployments/aws for reference>
export ALPHABILL_VERSION=<normally sha-1 hash, can be anything>
ansible-playbook deployments/aws/site-publish-binaries.yml -e gt_environment=${ENVIRONMENT} -e alphabill_version=${ALPHABILL_VERSION}
```

### Deploying a published artifact

```console
export ENVIRONMENT=<environment name, check deployments/aws for reference>
export ALPHABILL_VERSION=<normally sha-1 hash, can be anything>
ansible-playbook deployments/aws/site-deploy-nomad.yml -e gt_environment=${ENVIRONMENT} -e alphabill_version=${ALPHABILL_VERSION}
```

### Stopping an environment

```console
export ENVIRONMENT=<environment name, check deployments/aws for reference>
ansible-playbook deployments/aws/site-stop-nomad.yml -e gt_environment=${ENVIRONMENT}
```
