# docker plugin vault

docker secret plugin for vault provider.

<https://docs.docker.com/engine/extend/#debugging-plugins>

## how to build docker plugin

    make clean
    make create
    make push

## how to install plugin
 
install and setup token, enabled by default:

    docker plugin install --grant-all-permissions kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:1.0.0 VAULT_TOKEN=your-token

setup token if renew:

    docker plugin disable kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:1.0.0
    docker plugin set kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:1.0.0 VAULT_TOKEN=your-token
    docker plugin enable kaizen--dev.artifactory.geo.arm.com/docker-plugin-vault:1.0.0

## design

原来设计：

1. 通过一个service来传递token给plugin，这样token是安全的。但是要额外创建一个service，并且创建一个token的secret。
2. plugin自己创建token，并给每个secret创建对应的policy。

目前实现：

1. 通过环境变量传递token，docker plugin inspect能看到token,不安全。
2. 只能通过field,path,version在docker stack deploy的时候创建token。
3. 不需要创建token和policy。
