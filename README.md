# vcenter-connector

OpenFaaS event connector for VMware vCenter

## Status

This project connects VMware vCenter events to OpenFaaS functions via topics applied in the "topic" annotation. 

It is build using the [OpenFaaS Connector SDK](https://github.com/openfaas-incubator/connector-sdk)

The code is under active development and only suitable for testing at this point.

## Example:

* Run the vCenter Simulator

```bash
 ./vcsim -tls=false
```

* Run the connector

```bash
export OPENFAAS_URL=http://127.0.0.1:31112
go run main.go -vcenter-url="http://user:pass@127.0.0.1:8989/sdk" -insecure
```

Deploy an echo function that subscribes to the event of "vm.powered.on"

```bash
export OPENFAAS_URL=http://127.0.0.1:31112

git clone https://github.com/alexellis/echo-fn
cd echo-fn
faas-cli deploy
```

The `stack.yml` contains an annotation of `topic=vm.powered.on`, to change this edit the file and run `faas-cli deploy`. To edit the code in the handler change the code and `image` field then run `faas-cli up`
```

* Generate some events:

```
GOVC_INSECURE=true GOVC_URL=http://user:pass@127.0.0.1:8989/sdk govc vm.power -off '*'
```

* Check the logs of the `echo-fn` function

```bash
# on Kubernetes

kubectl logs -n openfaas-fn deploy/echo-fn

# or on Swarm

docker service logs echo-fn
```

## License

MIT

## Acknowledgements

Thanks to VMware's Doug MacEachern for the awesome [govmomi](https://github.com/vmware/govmomi) project providing Golang bindings for vCenter and the [vcsim simulator tool](https://github.com/vmware/govmomi/blob/master/vcsim/README.md).

Thanks to Karol Stępniewski for showing me a demo of events being consumed in OpenFaaS via vCenter over a year ago at KubeCon in Austin. Parts of his "event-driver" originally developed in the Dispatch project have been adapted for this OpenFaaS event-connector including a method to convert camel case event names into names separated by a dot. I wanted to include this for compatibility between the two systems.
