package templates

var DataMongoReplicaTemplate = `
{{define "data-mongo-replica"}}
{
  "kind": "Template",
  "apiVersion": "v1",
  "metadata": {
    "name": "mongodb",
    "labels": {
      "deployed": "false",
      "rhmap/name": "data-mongo"
    },
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
        "name": "mongodb-1",
        "labels": {
          "name": "mongodb-1",
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "ports": [
          {
            "port": 27017
          }
        ],
        "selector": {
          "name": "mongodb-replica-1"
        },
        "clusterIP": "None"
      }
    },
    {
      "kind": "Service",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-2",
        "labels": {
          "name": "mongodb-2",
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "ports": [
          {
            "port": 27017
          }
        ],
        "selector": {
          "name": "mongodb-replica-2"
        },
        "clusterIP": "None"
      }
    },
    {
      "kind": "Service",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-3",
        "labels": {
          "name": "mongodb-3",
		      "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "ports": [
          {
            "port": 27017
          }
        ],
        "selector": {
          "name": "mongodb-replica-3"
        },
        "clusterIP": "None"
      }
    },
    {
      "kind": "PersistentVolumeClaim",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-claim-1",
		"labels": {
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
      "kind": "PersistentVolumeClaim",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-claim-2",
		"labels": {
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
      "kind": "PersistentVolumeClaim",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-claim-3",
		"labels": {
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
          "rhmap/type":"environmentService",
          "deployed": "false"
        }
      },
      "spec": {
        "strategy": {
          "type": "Recreate",
          "resources": {
            "limits": {
              "cpu": "1000m",
              "memory": "1000Mi"
            },
            "requests": {
              "cpu": "200m",
              "memory": "200Mi"
            }
          }
        },
        "triggers": [
          {
            "type": "ConfigChange"
          }
        ],
        "replicas": 1,
        "selector": {
          "name": "mongodb-replica-1"
        },
        "template": {
          "metadata": {
            "labels": {
              "name": "mongodb-replica-1",
			  "rhmap/domain": "{{.Domain}}",
              "rhmap/env": "{{.Env}}",
              "rhmap/title": "{{.ServiceName}}",
              "rhmap/name": "data-mongo",
              "rhmap/type":"environmentService"
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
                "name": "mongodb-service",
				"image": "docker.io/rhmap/mongodb:centos-3.2-40",
                "ports": [
                  {
                    "containerPort": 27017
                  }
                ],
                "volumeMounts": [
                  {
                    "name": "mongodb-data-volume",
                    "mountPath": "/var/lib/mongodb/data"
                  }
                ],
                "env": [
                  {
                    "name": "MONGODB_REPLICA_NAME",
                    "value": "rs0"
                  },
                  {
                    "name": "MONGODB_SERVICE_NAME",
                    "value": "mongodb"
                  },
                  {
                    "name": "MONGODB_KEYFILE_VALUE",
                    "value": "{{generatedPass}}"
                  },
                  {
                    "name": "MONGODB_ADMIN_PASSWORD",
                    "value": "{{generatedPass}}"
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
                "imagePullPolicy": "IfNotPresent"
              }
            ]
          }
        }
      }
    },
	{
      "kind": "DeploymentConfig",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-2",
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
          "type": "Recreate",
          "resources": {
            "limits": {
              "cpu": "1000m",
              "memory": "1000Mi"
            },
            "requests": {
              "cpu": "200m",
              "memory": "200Mi"
            }
          }
        },
        "triggers": [
          {
            "type": "ConfigChange"
          }
        ],
        "replicas": 1,
        "selector": {
          "name": "mongodb-replica-2"
        },
        "template": {
          "metadata": {
            "labels": {
              "name": "mongodb-replica-2",
			  "rhmap/domain": "{{.Domain}}",
              "rhmap/env": "{{.Env}}",
              "rhmap/title": "{{.ServiceName}}",
              "rhmap/name": "data-mongo",
              "rhmap/type":"environmentService"
            }
          },
          "spec": {
            "volumes": [
              {
                "name": "mongodb-data-volume",
                "persistentVolumeClaim": {
                  "claimName": "mongodb-claim-2"
                }
              }
            ],
            "containers": [
              {
                "name": "mongodb-service",
                "image": "docker.io/rhmap/mongodb:centos-3.2-40",
                "ports": [
                  {
                    "containerPort": 27017
                  }
                ],
                "volumeMounts": [
                  {
                    "name": "mongodb-data-volume",
                    "mountPath": "/var/lib/mongodb/data"
                  }
                ],
                "env": [
                  {
                    "name": "MONGODB_REPLICA_NAME",
                    "value": "rs0"
                  },
                  {
                    "name": "MONGODB_KEYFILE_VALUE",
                    "value": "{{generatedPass}}"
                  },
                  {
                    "name": "MONGODB_ADMIN_PASSWORD",
                    "value": "{{generatedPass}}"
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
                "imagePullPolicy": "IfNotPresent"
              }
            ]
          }
        }
      }
    },
	{
      "kind": "DeploymentConfig",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-3",
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
          "type": "Recreate",
          "resources": {
            "limits": {
              "cpu": "1000m",
              "memory": "1000Mi"
            },
            "requests": {
              "cpu": "200m",
              "memory": "200Mi"
            }
          }
        },
        "triggers": [
          {
            "type": "ConfigChange"
          }
        ],
        "replicas": 1,
        "selector": {
          "name": "mongodb-replica-3"
        },
        "template": {
          "metadata": {
            "labels": {
              "name": "mongodb-replica-3",
			  "rhmap/domain": "{{.Domain}}",
              "rhmap/env": "{{.Env}}",
              "rhmap/title": "{{.ServiceName}}",
              "rhmap/name": "data-mongo",
              "rhmap/type":"environmentService"
            }
          },
          "spec": {
            "volumes": [
              {
                "name": "mongodb-data-volume",
                "persistentVolumeClaim": {
                  "claimName": "mongodb-claim-3"
                }
              }
            ],
            "containers": [
              {
                "name": "mongodb-service",
                "image": "docker.io/rhmap/mongodb:centos-3.2-40",
                "ports": [
                  {
                    "containerPort": 27017
                  }
                ],
                "volumeMounts": [
                  {
                    "name": "mongodb-data-volume",
                    "mountPath": "/var/lib/mongodb/data"
                  }
                ],
                "env": [
                  {
                    "name": "MONGODB_REPLICA_NAME",
                    "value": "rs0"
                  },
                  {
                    "name": "MONGODB_KEYFILE_VALUE",
                    "value": "{{generatedPass}}"
                  },
                  {
                    "name": "MONGODB_ADMIN_PASSWORD",
                    "value": "{{generatedPass}}"
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
                "imagePullPolicy": "IfNotPresent"
              }
            ]
          }
        }
      }
    },
	{
      "kind": "Pod",
      "apiVersion": "v1",
      "metadata": {
        "name": "mongodb-initiator",
        "labels": {
          "name": "mongodb-initiator",
		  "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/name": "data-mongo",
          "rhmap/type":"environmentService"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "mongodb",
            "image": "docker.io/rhmap/mongodb:centos-3.2-40",
            "ports": [
              {
                "containerPort": 27017
              }
            ],
            "command": [
              "run-mongod",
              "initiate"
            ],
            "env": [
              {
                "name": "METADATA_NAMESPACE",
                "valueFrom": {
                  "fieldRef": {
                    "fieldPath": "metadata.namespace"
                  }
                }
              },
              {
                "name":"MONGODB_ADMIN_PASSWORD",
                "value":"{{generatedPass}}"
              },
              {
                "name": "MONGODB_REPLICA_NAME",
                "value": "rs0"
              },              
              {
                "name": "MONGODB_KEYFILE_VALUE",
                "value": "{{generatedPass}}"
              },
              {
                "name": "ENDPOINT_COUNT",
                "value": "3"
              },
              {
                "name": "MONGODB_INITIAL_REPLICA_COUNT",
                "value": "3"
              }
            ],
            "imagePullPolicy": "IfNotPresent"
          }
        ],
        "restartPolicy": "Never"
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
