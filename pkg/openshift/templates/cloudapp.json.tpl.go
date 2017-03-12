package templates

//CloudAppTemplate defines the template for deploying a cloud app to openshift 3
var CloudAppTemplate = `
{{define "cloudapp"}}
{
  "kind": "Template",
  "apiVersion": "v1",
  "metadata": {
    "name": "cloudapp",
    "annotations": {
      "description": "cloudApp",
      "tags": "rhmap,cloudApp"
    }
  },
  "objects": [
  {
      "kind": "ImageStream",
      "apiVersion": "v1",
      "metadata": {
          "name": "{{.ServiceName}}",
          "labels": {
               "rhmap/domain": "{{.Domain}}",
               "rhmap/env": "{{.Env}}",
               "rhmap/guid": "{{.CloudAppGUID}}",
               "rhmap/project": "{{.ProjectGUID}}",
               "rhmap/title": "{{.ServiceName}}"
          },
          "annotations": {
              "description": "Keeps track of changes in the application image",
              "rhmap/description": "",
              "rhmap/title": "{{.ServiceName}}"
          }
      }
  },
    {
      "kind": "Route",
      "apiVersion": "v1",
      "metadata": {
        "name": "{{.ServiceName}}",
        "creationTimestamp": null,
        "labels": {
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/guid": "{{.CloudAppGUID}}",
          "rhmap/project": "{{.ProjectGUID}}",
          "rhmap/title": "{{.ServiceName}}"
        },
        "annotations": {
          "rhmap/description": "",
          "rhmap/title": "{{.ServiceName}}"
        }
      },
      "spec": {
        "host": "",
        "to": {
          "kind": "Service",
          "name": "{{.ServiceName}}",
          "weight": 100
        },
        "tls": {
          "termination": "edge",
          "insecureEdgeTerminationPolicy": "Allow"
        }
      }
    },
    {
      "kind": "BuildConfig",
      "apiVersion": "v1",
      "metadata": {
        "name": "{{.ServiceName}}",
        "creationTimestamp": null,
        "labels": {
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/guid": "{{.CloudAppGUID}}",
          "rhmap/project": "{{.ProjectGUID}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/type": "cloudapp",
          "rhmap/name": "cloudapp"
        },
        "annotations": {
          "description": "Defines how to build the application",
          "rhmap/description": "",
          "rhmap/title": "{{.ServiceName}}"
        }
      },
      "spec": {
        "triggers": [
          {
            "type": "ConfigChange"
          },
          {
            "type": "ImageChange",
            "imageChange": {}
          }
        ],
        "runPolicy": "SerialLatestOnly",
        "source": {
          "type": "Git",
          "git": {
            "uri": "{{.Repo.Loc}}",
            "ref": "{{.Repo.Ref}}"
          }
          {{if ne .Repo.Auth.AuthType  "" }}
          ,
          "sourceSecret": {
            "name": "{{.ServiceName}}-scmsecret"
          }
          {{end}}
        },
        "strategy": {
          "type": "Source",
          "sourceStrategy": {
            "from": {
              "kind": "ImageStreamTag",
              "namespace": "openshift",
              "name": "nodejs:4"
            },
            "env": [
              {
                "name": "NODE_ENV",
                "value": "production"
              }
            ]
          }
        },
        "output": {
          "to": {
            "kind": "ImageStreamTag",
            "name": "{{.ServiceName}}:latest"
          }
        },
        "resources": {},
        "postCommit": {}
      }
    },{{if ne .Repo.Auth.AuthType  "" }}
     {
        "apiVersion": "v1",
        "kind": "Secret",
        "type": "Opaque",
        "metadata": {
          "name": "{{.ServiceName}}-scmsecret",
          "labels" : {
            "rhmap/domain": "{{.Domain}}",
            "rhmap/env": "{{.Env}}",
            "rhmap/guid": "{{.CloudAppGUID}}",
            "rhmap/project": "{{.ProjectGUID}}",
            "rhmap/title": "{{.ServiceName}}"
          },
          "annotations" : {
            "rhmap/description" : "cloud app git secret",
            "rhmap/title" : "{{.ServiceName}}",
            "description": "git secret for cloning remote repo"
          }
        },
        "data":{
          {{if eq .Repo.Auth.AuthType "http"}}
            "username":"{{.Repo.Auth.User}}",
            "password":"{{.Repo.Auth.Key}}"
          {{end}}
          {{if eq .Repo.Auth.AuthType "ssh"}}
            "ssh-privatekey":"{{.Repo.Auth.Key}}" 
          {{end}}  
        }
    },{{end}}
    {
      "kind": "Service",
      "apiVersion": "v1",
      "metadata": {
        "name": "{{.ServiceName}}",
        "creationTimestamp": null,
        "labels": {
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/guid": "{{.CloudAppGUID}}",
          "rhmap/project": "{{.ProjectGUID}}",
          "rhmap/title": "{{.ServiceName}}"
        },
        "annotations": {
          "description": "Exposes and load balances the application pods",
          "rhmap/description": "",
          "rhmap/title": "{{.ServiceName}}"
        }
      },
      "spec": {
        "ports": [
          {
            "name": "web",
            "protocol": "TCP",
            "port": 8001,
            "targetPort": 8001
          }
        ],
        "selector": {
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/guid": "{{.CloudAppGUID}}",
          "rhmap/project": "{{.ProjectGUID}}"
        },
        "type": "ClusterIP",
        "sessionAffinity": "None"
      }
    },
    {
      "kind": "DeploymentConfig",
      "apiVersion": "v1",
      "metadata": {
        "name": "{{.ServiceName}}",
        "creationTimestamp": null,
        "labels": {
          "rhmap/domain": "{{.Domain}}",
          "rhmap/env": "{{.Env}}",
          "rhmap/guid": "{{.CloudAppGUID}}",
          "rhmap/project": "{{.ProjectGUID}}",
          "rhmap/title": "{{.ServiceName}}",
          "rhmap/type": "cloudapp",
          "rhmap/name": "cloudapp"
        },
        "annotations": {
          "description": "Defines how to deploy the application server",
          "rhmap/description": "",
          "rhmap/title": "{{.ServiceName}}"
        }
      },
      "spec": {
        "strategy": {
          "type": "Rolling",
          "rollingParams": {
            "updatePeriodSeconds": 1,
            "intervalSeconds": 1,
            "timeoutSeconds": 600,
            "maxUnavailable": "25%",
            "maxSurge": "25%"
          },
          "resources": {}
        },
        "triggers": [
          {
            "type": "ConfigChange"
          },
          {
            "type": "ImageChange",
            "imageChangeParams": {
              "automatic": true,
              "containerNames": [
                "{{.ServiceName}}"
              ],
              "from": {
                "kind": "ImageStreamTag",
                "name": "{{.ServiceName}}:latest"
              }
            }
          }
        ],
        "replicas": {{.Replicas}},
        "selector": {
          "name": "{{.ServiceName}}"
        },
        "template": {
          "metadata": {
            "name": "{{.ServiceName}}",
            "creationTimestamp": null,
            "labels": {
              "name": "{{.ServiceName}}",
              "rhmap/domain": "{{.Domain}}",
              "rhmap/env": "{{.Env}}",
              "rhmap/guid": "{{.CloudAppGUID}}",
              "rhmap/project": "{{.ProjectGUID}}"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "{{.ServiceName}}",
                "image": "{{.ServiceName}}",
                "ports": [
                  {
                    "containerPort": 8001,
                    "protocol": "TCP"
                  }
                ],
                "env": [
                   {
                    "name":"FH_REDIS_HOST",
                    "value":""
                  },
                  {{$len := len .EnvVars}}
                  {{range $index,$envVar := .EnvVars}}
                  {
                    "name": "{{$envVar.Name}}",
                    "value": "{{$envVar.Value}}"
                  }
                    {{if not (isEnd $index $len)}}
                  ,
                    {{end}}
                  {{end}}
                ],
                "resources": {
                  "limits": {
                    "cpu": "500m",
                    "memory": "250Mi"
                  },
                  "requests": {
                    "cpu": "100m",
                    "memory": "90Mi"
                  }
                },
                "terminationMessagePath": "/dev/termination-log",
                "imagePullPolicy": "Always"
              }
            ],
            "restartPolicy": "Always",
            "terminationGracePeriodSeconds": 30,
            "dnsPolicy": "ClusterFirst",
            "securityContext": {}
          }
        }
      }
    }
  ]
}
{{end}}`
