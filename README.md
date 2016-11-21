### Negotiator 

Negotiator negotiates between an RHMAP core and a OpenShift instance. It understands both.

Try it out locally.

Create a new OpenShift Project ``` oc new-project mine ```

- ensure go 1.7 or greater installed
- install glide package manager ``` curl https://glide.sh/get | sh ``` 
- clone this repo into $GOPATH/src/github.com/feedhenry/negotiator
- ``` cd $GOPATH/src/github.com/feedhenry/negotiator ```
- ``` glide i --strip-vendor ``` 
- ``` go build . ``` (will be slow the first time but faster afterwards)
- export some envars for config 
    - ``` export API_HOST="http://anopenshifthost.com:8443" ```
    - ``` export API_TOKEN="some-os-token" ```
    - ``` export DEPLOY_NAMESPACE="mine" ```



Start the server 

```
./negotiator

```

Curl request 

```
curl -X POST  http://localhost:3000/deploy/cloudapp -d '{"guid":"testguid","namespace":"mine","domain":"testing"}'

```

Note currently it just sets up a service, a route and an imagestream. 