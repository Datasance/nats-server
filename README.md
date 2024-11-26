# nats-server for ioFog Cluster

Nats server image for deploying nats server or leaf node on Edge devices


## Example yaml for nats-leaf without TLS

```yaml

apiVersion: datasance.com/v3
kind: Application
metadata:
  name: nats-leaf$
spec:
  microservices:
    - name: nats-leaf$
      agent:
        name: $
      images:
        registry: 1
        catalogItemId: null
        x86: ghcr.io/datasance/nats:latest
        arm: ghcr.io/datasance/nats:latest
      container:
        rootHostAccess: false
        runAsUser: null
        platform: null
        runtime: null
        cdiDevices: []
        ports:
          - internal: 4222
            external: 4222
            protocol: tcp
        volumes:
          - hostDestination: nats-data
            containerDestination: /store_leaf
            accessMode: rw
            type: volume
        env: []
        extraHosts: []
        commands: []
      config:
        accounts:
          - accountName: $
            users:
              - username: $
                password: $
            jetstream: true
        natsServer:
          serverName: $
          port: 4222
          jsDomain: $
          leafNodes:
            remotes:
              urlProtocol: nats
              url: $:7422
              user: $
              password: $
              account: $
          mqtt:
            port: $
            jsDomain: $
          mqttAuth:
            users:
              - username: $
                password: $
  routes: []

```


## Example yaml for nats-leaf with TLS

```yaml

apiVersion: datasance.com/v3
kind: Application
metadata:
  name: demo-1-nats
spec:
  microservices:
    - name: demo-1-nats
      agent:
        name: demo-1
      images:
        registry: 1
        catalogItemId: null
        x86: ghcr.io/datasance/nats:latest
        arm: ghcr.io/datasance/nats:latest
      container:
        rootHostAccess: false
        runAsUser: null
        platform: null
        runtime: null
        cdiDevices: []
        ports:
          - internal: 4222
            external: 4222
            protocol: tcp
        volumes:
          - hostDestination: nats-data
            containerDestination: /store_leaf
            accessMode: rw
            type: volume
        env: []
        extraHosts: []
        commands: []
      config:
        accounts:
          - accountName: $
            users:
              - username: $
                password: $
            jetstream: true
        natsServer:
          serverName: $
          port: 4222
          jsDomain: $
          leafNodes:
            remotes:
              urlProtocol: ws
              url: $
              user: $
              password: $
              account: $
              tls:
                CaCert: >-
                  $Base64 ca crt
                TlsCert: >-
                  $Base64 tls crt
                TlsKey: >-
                  $Base64 tls key
          mqtt:
            port: $
            jsDomain: $
            tls:
                CaCert: >-
                    $Base64 ca crt
                TlsCert: >-
                    $Base64 tls crt
                TlsKey: >-
                    $Base64 tls key
          mqttAuth:
            users:
              - username: $
                password: $
  routes: []

```


## Example yaml for nats-server without TLS

```yaml

apiVersion: datasance.com/v3
kind: Application
metadata:
  name: nats-leaf$
spec:
  microservices:
    - name: nats-leaf$
      agent:
        name: $
      images:
        registry: 1
        catalogItemId: null
        x86: ghcr.io/datasance/nats:latest
        arm: ghcr.io/datasance/nats:latest
      container:
        rootHostAccess: false
        runAsUser: null
        platform: null
        runtime: null
        cdiDevices: []
        ports:
          - internal: 4222
            external: 4222
            protocol: tcp
          - internal: 7422
            external: 7422
            protocol: tcp
        volumes:
          - hostDestination: nats-data
            containerDestination: /store_leaf
            accessMode: rw
            type: volume
        env: []
        extraHosts: []
        commands: []
      config:
        accounts:
          - accountName: $
            users:
              - username: $
                password: $
            jetstream: true
          - accountName: $
            users:
              - username: $
                password: $
            issystem: true
        natsServer:
          serverName: $
          port: 4222
          jsDomain: $
          leafNodes:
            port: 7422
          mqtt:
            port: $
            jsDomain: $
          mqttAuth:
            users:
              - username: $
                password: $

  routes: []

```


## Example yaml for nats-server with TLS

```yaml

apiVersion: datasance.com/v3
kind: Application
metadata:
  name: nats-leaf$
spec:
  microservices:
    - name: nats-leaf$
      agent:
        name: $
      images:
        registry: 1
        catalogItemId: null
        x86: ghcr.io/datasance/nats:latest
        arm: ghcr.io/datasance/nats:latest
      container:
        rootHostAccess: false
        runAsUser: null
        platform: null
        runtime: null
        cdiDevices: []
        ports:
          - internal: 4222
            external: 4222
            protocol: tcp
          - internal: 7422
            external: 7422
            protocol: tcp
        volumes:
          - hostDestination: nats-data
            containerDestination: /store_leaf
            accessMode: rw
            type: volume
        env: []
        extraHosts: []
        commands: []
      config:
        accounts:
          - accountName: $
            users:
              - username: $
                password: $
            jetstream: true
          - accountName: $
            users:
              - username: $
                password: $
            issystem: true
        natsServer:
          serverName: $
          port: 4222
          jsDomain: $
          leafNodes:
            port: 7422
          tls:
            CaCert: >-
              $Base64 ca crt
            TlsCert: >-
              $Base64 tls crt
            TlsKey: >-
              $Base64 tls key
          mqtt:
            port: $
            jsDomain: $
            tls:
                CaCert: >-
                    $Base64 ca crt
                TlsCert: >-
                    $Base64 tls crt
                TlsKey: >-
                    $Base64 tls key
          mqttAuth:
            users:
              - username: $
                password: $
  routes: []


  routes: []
```