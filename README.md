# docker plugin vault

docker secret plugin for vault provider.

<https://docs.docker.com/engine/extend/#debugging-plugins>

## how to build docker plugin

    make clean
    make create
    make push

## how to install plugin
 
install and setup token, enabled by default:

    docker plugin install --grant-all-permissions canux--dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:1.0.0 VAULT_TOKEN=your-token

setup token if renew:

    docker plugin disable canux--dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:1.0.0
    docker plugin set canux--dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:1.0.0 VAULT_TOKEN=your-token
    docker plugin enable canux--dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:1.0.0

## design

原来设计：

1. 通过一个service来传递token给plugin，这样token是安全的。但是要额外创建一个service，并且创建一个token的secret。
2. plugin自己创建token，并给每个secret创建对应的policy。

1.0.0实现：

1. 通过环境变量传递token，docker plugin inspect能看到token,不安全。
2. 只能通过field,path,version在docker stack deploy的时候创建token。
3. 不需要创建token和policy。

1.1.0实现：

1. 给vault token创建一个secret
2. 通过secret获取token

## how to debug plugin

docker以debug模式启动

    "debug": true
    
查看log

    journalctl -f -u docker.service
    
    cd /run/docker/plugins/$your_plugin_id
    cat < init-stdout
    cat < init-stderr
    
## how to use 

use it in compose file

    secrets:
      kaizen_haproxy_ca:
        driver: canux--dev.eu-west-1.artifactory.canux.com/docker-plugin-vault:0.0.1
        labels:
          docker.plugin.secretprovider.vault.path: canux/data/pki
          docker.plugin.secretprovider.vault.field: "*.canuxcheng.com"

