### Negotiator 

Negotiator negotiates between an RHMAP core and a OpenShift instance. It understands both.
Negotiator plays the role of being the middleman between rhmap and OpenShift. It uses the OpenShift client and Kubernetes client to directly interact with the Kubernetes API and OpenShift API. For example when a new cloud app from RHMAP needs to be deployed or a new environment needs to be created, a request will be sent to negotiator with the details required such as a giturl, an auth token, env vars etc. Negotiator will take care of turning this into the required OpenShift Objects and sending them on to OpenShift / Kubernetes directly.

Try it out locally.

Create a new OpenShift Project ``` oc new-project mine ```

- ensure go 1.7 or greater installed
- clone this repo into $GOPATH/src/github.com/feedhenry/negotiator
- ``` cd $GOPATH/src/github.com/feedhenry/negotiator ```
- ``` go build``` (will be slow the first time but faster afterwards)
- ``` ./negotiator ```

In a separate terminal run the following 

```
oc new-project se
curl http://localhost:3000/deploy/se/cloudapp -H Content-type:application/json -d '{"repo": {"loc": "https://github.com/feedhenry/testing-cloud-app.git","ref": "master"}, "target":{"host":"AN OPENSHIFT MASTER","token":"AN OPENSHIFT TOKEN"}, "serviceName": "cloudapp4","replicas": 1,  "projectGuid":"test","envVars":[{"name":"test","value":"test"}]}'

curl http://localhost:3000/deploy/se/cache -H Content-type:application/json -d '{"serviceName": "cache","replicas": 1,  "projectGuid":"test", "target":{"host":"AN OPENSHIFT MASTER","token":"AN OPENSHIFT TOKEN"}}'
```

## Developing

- install glide package manager ``` curl https://glide.sh/get | sh ``` 

All dependencies are vendored so you shouldn't need to update or install. 

Layout:
```bash
.
├── config
│
├── cmd # where the main.go for the server and cli are located
│
├── deploy #domain specfic logic for deployment
│   ├── template.go # deploys templates to OpenShift
│   
└── pkg
│    └── openshift # pkg for making the openshift and kubernetes client more simple to work with. Our domain logic does not go here
│ ##handlers go in the root dir and deal with http specific logic 
│  
└── web 
     └── deploy.go 
     └── sys.go      

``` 

## Test 

```bash
make test-unit 
```

## build and publish

```
env GOOS=linux go build .

docker build -t rhmap/negotiator:0.0.1 . ##change build number

docker tag rhmap/negotiator:0.0.1 rhmap/negotiator:latest

docker push rhmap/negotiator:0.0.1

```

## Run in OpenShift

```
oc new-app -f os_template.json --param=API_TOKEN=<YOUR_TOKEN>,API_HOST=<OpenShift_HOST>,DEPLOY_NAMESPACE=<SOME_NS>

```
