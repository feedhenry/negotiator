## Setup rhmap to talk to negotiator for local development

Use the OpenShift development VM

### Supercore

checkout the environment_services branch of fh-supercore

```
git fetch --all
git checkout environment_services
```

We want to run supercore from source so we need to perform some steps in our core project

1) ``` oc edit scc restriced ``` add the following to the volumes array ```hostPath``` and change ```allowHostDirVolumePlugin``` to ```true``

2) inside the vm ```cd /mnt/src/fh-supercore && npm install . ```

3) add nodemon to the start command ``` "start": "nodemon /usr/src/app/fh-supercore.js config/conf.json --master-only" ``` in package.json

4) change the supercore  dc to use 

change the image to  ```francolaiuppa/docker-nodemon-forever:latest```

edit the environment and a new envar:  ```- name: XDG_CONFIG_HOME value: /usr/src/app/  ```      

add a new volume: ``` - hostPath: path: /mnt/src/fh-supercore name: source```

add a new volumeMount: ```- mountPath: /usr/src/app name: source```


5)  Create a file change.js in the root of supercore. When you want to the code to reload do ```touch change.js```



### Negotiator

in the core project use the template in the root of this project ```os_template.json``` to add negotiator to the core project

```
oc new-app -f os_tempalte.json

```

### Update supercore environment to point at negotiator

Add the following env vars

```
NEGOTIATOR_ENABLED: true
NEGOTIATOR_SERVICE_HOST: "http://negotiator"
NEGOTIATOR_SERVICE_PORT: "3000"
```

supercore can now talk to negotiator. 

To test it create an MBaaS target against OpenShift. But don't deploy any MBaaS just fill in the mbaas specific fields with fake values.

next create an environment that targets that MBaaS target. 

Finally create a new project and deploy it. It should deploy succesfully to your environment via negotiator



