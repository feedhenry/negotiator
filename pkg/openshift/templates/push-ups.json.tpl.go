package templates

//PushUPSTemplate defines the template for deploying a push via UPS environment service to openshift 3
var PushUPSTemplate = `
{{define "push-ups"}}
{
  	"kind": "Template",
  	"apiVersion": "v1",
  	"metadata": {
    	"name": "RHMAP UPS",
    	"annotations": {
      		"description": "RHMAP ups",
			"dependencies": "data-mysql",
      		"tags": "rhmap,mobile,nodejs",
      		"iconClass": "icon-jboss"
    	}
  	},
  	"objects": [
	{
      "kind": "ConfigMap",
      "apiVersion": "v1",
      "metadata": {
        "name": "ups-client-config"
      },
      "data": {
      }
    },
	{
        "kind": "Route",
        "apiVersion": "v1",
        "metadata": {
            "name": "push-ups",
            "creationTimestamp": null,
            "labels": {
                "name": "push-ups"
            },
            "annotations": {
                "openshift.io/host.generated": "true"
            }
        },
        "spec": {
            "host": "",
            "to": {
                "kind": "Service",
                "name": "push-ups"
            }
        }
    },
	{
			"kind": "Service",
			"apiVersion": "v1",
			"metadata": {
				"name": "{{.ServiceName}}",
				"labels": {
					"name": "{{.ServiceName}}",
					"rhmap/domain": "{{.Domain}}",
					"rhmap/env": "{{.Env}}",
					"rhmap/guid": "{{ generatedPass }}",
					"rhmap/project": "{{.ProjectGUID}}",
					"rhmap/name": "push-ups",
					"rhmap/type": "environmentService"
				}
			},
			"spec": {
				"selector": {
					"name": "{{.ServiceName}}"
				},
				"ports": [
					{
            			"port": 8080
          			}
        		]
      		}
    	},
    	{
			"kind": "DeploymentConfig",
			"apiVersion": "v1",
			"metadata": {
				"kind":"DeploymentConfig",
				"name": "{{.ServiceName}}",
				"labels": {
					"name": "{{.ServiceName}}",
					"rhmap/domain": "{{.Domain}}",
					"rhmap/env": "{{.Env}}",
					"rhmap/guid": "{{ generatedPass }}",
					"rhmap/project": "{{.ProjectGUID}}",
					"rhmap/name": "push-ups",
					"rhmap/type":"environmentService"
				},
				"annotations": {
					"description": "Defines how to deploy the UPS push service"
				}
			},
			"spec": {
				"triggers": [
					{
						"type": "ConfigChange"
					}
				],
				"replicas": 1,
				"selector": {
					"name": "{{.ServiceName}}"
				},
				"template": {
					"metadata": {
						"name": "{{.ServiceName}}",
						"labels": {
							"name": "{{.ServiceName}}",
							"rhmap/domain": "{{.Domain}}",
							"rhmap/env": "{{.Env}}",
							"rhmap/guid": "{{ generatedPass }}",
							"rhmap/project": "{{.ProjectGUID}}",
							"rhmap/name": "push-ups",
							"rhmap/type": "environmentService"
						}
					},
          			"spec": {
						"containers": [
							{
								"name": "{{.ServiceName}}",
								"image": "docker.io/rhmap/unifiedpush-eap:1.1.4.Final-52",
								"ports": [
									{
										"containerPort": 8443
									},
									{
										"containerPort": 8080
									},
									{
										"containerPort": 9990
									}
								],
								"livenessProbe": {
									"httpGet": {
										"path": "/ag-push",
										"port": 8080
									},
									"initialDelaySeconds": 600,
									"timeoutSeconds": 5,
									"periodSeconds": 60,
									"successThreshold": 1,
									"failureThreshold": 2
								},
								"readinessProbe": {
									"httpGet": {
										"path": "/ag-push",
										"port": 8080
									},
									"initialDelaySeconds": 100,
									"timeoutSeconds": 5,
									"periodSeconds": 10,
									"successThreshold": 1,
									"failureThreshold": 1
								},
								"env": [
									{
										"name":"ADMIN_USER",
										"value":"admin"
									},
									{
										"name":"ADMIN_PASSWORD",
										"value":"123"
									},
									{
										"name": "KEYCLOAK_BASE_URL",
										"value": ""
									}
								],
								"imagePullPolicy": "IfNotPresent",
								"resources": {
									"limits": {
										"cpu": "2000m",
										"memory": "5000Mi"
									},
									"requests": {
										"cpu": "400m",
										"memory": "900Mi"
									}
								}
							}
						]
          			}
        		}
      		}
    	}
  	]
}
{{end}}`
