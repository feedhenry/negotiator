package templates

var DataMongoTemplate = `
{{define "data-mongo"}}
{
  "kind": "Template",
  "apiVersion": "v1",
  "metadata": {
    "name": "data-mongo",
    "annotations": {
      "description": "Mongodb",
      "tags": "rhmap,mongodb"
    }
  },
  "objects": [
    {
      "kind": "Service",
      "apiVersion": "v1",
      "metadata": {
        "name": "data-mongo",
        "labels": {
          "name": "data-mongo",
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo§",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "ports": [
          {
            "protocol": "TCP",
            "port": 27017,
            "targetPort": 27017,
            "nodePort": 0
          }
        ],
        "selector": {
          "name": "mongodb-replica"
        },
        "portalIP": "None",
        "clusterIP": "None",
        "type": "ClusterIP",
        "sessionAffinity": "None"
      }
    },
    {
      "kind": "PersistentVolumeClaim",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-claim-1",
		"labels": {
          "name": "data-mongo",
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "accessModes": [
          "ReadWriteOnce"
        ],
        "resources": {
          "requests": {
			{{if isset .Options "storage"}}  
            "storage": "{{ index .Options "storage"}}"
			{{else}}
			"storage": "1Gi"
			{{end}}
          }
        }
      }
    },
    {
      "kind": "DeploymentConfig",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-1",
        "labels": {
          "name": "mongodb",
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "strategy": {
          "type": "Recreate"
        },
        "triggers": [
          {
            "type": "ConfigChange"
          }
        ],
        "replicas": 1,
        "selector": {
          "name": "mongodb-replica"
        },
        "template": {
          "metadata": {
            "labels": {
              "name": "mongodb-replica"
            }
          },
          "spec": {
            "volumes": [
              {
                "name": "mongodb-data-volume",
                "persistentVolumeClaim": {
                  "claimName": "mongodb-claim-1"
                }
              }
            ],
            "containers": [
              {
                "name": "mongodb",
                "image": "docker.io/rhmap/mongodb:centos-3.2-40",
                "ports": [
                  {
                    "containerPort": 27017,
                    "protocol": "TCP"
                  }
                ],
                "env": [
                  {
                    "name": "MONGODB_REPLICA_NAME",
                    "value": ""
                  },
                  {
                    "name": "MONGODB_SERVICE_NAME",
                    "value": "data-mongo"
                  },
                  {
                    "name": "MONGODB_KEYFILE_VALUE",
                    "value": "{{genPass}}"
                  },
                  {
                    "name": "MONGODB_ADMIN_PASSWORD",
                    "value":"{{genPass}}"
                  }
                ],
                "resources": {
                  "limits": {
                    "cpu": "1000m",
                    "memory": "1000Mi"
                  },
                  "requests": {
                    "cpu": "200m",
                    "memory": "200Mi"
                  }
                },
                "volumeMounts": [
                  {
                    "name": "mongodb-data-volume",
                    "mountPath": "/var/lib/mongodb/data"
                  }
                ],
                "terminationMessagePath": "/dev/termination-log",
                "imagePullPolicy": "IfNotPresent"
              }
            ],
            "restartPolicy": "Always",
            "dnsPolicy": "ClusterFirst"
          }
        }
      }
    }
  ],
  "labels": {
    "template": "mongodb-core-single-template"
  },
  "params":[
    {
      "name": "storage",
      "value": "1Gi",
      "description": "The size of the volume for MongoDB Data",
      "required": true
    }
  ]
}
{{end}}
`
