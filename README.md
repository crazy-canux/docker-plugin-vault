# docker plugin vault

docker secret plugin for vault provider.

<https://docs.docker.com/engine/extend/#debugging-plugins>

## how to build docker plugin

    make clean
    make create
    make push


## how to install plugin
 
    docker plugin install --grant-all-permissions kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:1.0.0 VAULT_TOKEN=your-token

    docker plugin set kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:1.0.0 VAULT_TOKEN=your-token
