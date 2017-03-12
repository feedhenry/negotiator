package templates

//CacheTemplate defines the template for deploying a cache environment service to openshift 3
var CacheTemplate = `
{{define "cache"}}
{
    "kind": "Template",
    "apiVersion": "v1",
    "metadata": {
        "name": "cache",
        "annotations": {
            "description": "Redis",
            "tags": "rhmap,redis"
        }
    },
    "objects": [
        {
            "kind": "Service",
            "apiVersion": "v1",
            "metadata": {
                "name": "{{.ServiceName}}",
                "labels": {
                    "name": "{{.ServiceName}}",
                    "rhmap/domain": "{{.Domain}}",
                    "rhmap/env": "{{.Env}}",
                    "rhmap/guid": "{{.CloudAppGUID}}",
                    "rhmap/project": "{{.ProjectGUID}}"
                }
            },
            "spec": {
                "ports": [
                    {
                        "name": "{{.ServiceName}}",
                        "port": 6379,
                        "targetPort": 6379,
                        "protocol": "TCP"
                    }
                ],
                "selector": {
                    "name": "{{.ServiceName}}"
                }
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
                    "rhmap/guid": "{{.CloudAppGUID}}",
                    "rhmap/project": "{{.ProjectGUID}}",
                    "rhmap/name": "cache",
                    "rhmap/type":"environmentService"
                },
                "annotations": {
                    "description": "Defines how to deploy the redis caching layer"
                }
            },
            "spec": {
                "strategy": {
                    "type": "Recreate",
                    "recreateParams": {
                        "timeoutSeconds": 600
                    },
                    "resources": {}
                },
                "triggers": [
                    {
                        "type": "ConfigChange"
                    }
                ],
                "replicas": {{.Replicas}},
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
                                            "rhmap/guid": "{{.CloudAppGUID}}",
                                            "rhmap/project": "{{.ProjectGUID}}"
                                        }
                    },
                    "spec": {
                        "containers": [
                            {
                                "name": "{{.ServiceName}}",
                                "image": "docker.io/rhmap/redis:2.18.22",
                                "ports": [
                                    {
                                        "containerPort": 6379,
                                        "protocol": "TCP"
                                    }
                                ],
                                "env": [
                                    {
                                        "name": "REDIS_PORT",
                                        "value": "6379"
                                    }
                                ],
                                "resources": {
                                    "limits": {
                                        "cpu": "500m",
                                        "memory": "500Mi"
                                    },
                                    "requests": {
                                        "cpu": "100m",
                                        "memory": "100Mi"
                                    }
                                },
                                "terminationMessagePath": "/dev/termination-log",
                                "imagePullPolicy": "IfNotPresent"
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
