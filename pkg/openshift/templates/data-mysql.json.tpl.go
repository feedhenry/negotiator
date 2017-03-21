package templates

// MySQLTemplate is the template for deploying the mysql service to an openshift3 target
var DataMySQLTemplate = `
{{define "data-mysql"}}
{
    "kind": "Template",
    "apiVersion": "v1",
    "metadata": {
        "name": "mysql-persistent",
        "annotations": {
            "description": "MySQL database service, with persistent storage.  Scaling to more than one replica is not supported.  You must have persistent volumes available in your cluster to use this template.",
            "iconClass": "icon-mysql-database",
            "tags": "database,mysql"
        }
    },
    "objects": [
        {
            "kind": "Service",
            "apiVersion": "v1",
            "metadata": {
                "name": "data-mysql",
                "labels": {
                    "name": "{{.ServiceName}}",
                    "rhmap/domain": "{{.Domain}}",
                    "rhmap/env": "{{.Env}}",
                    "rhmap/project": "{{.ProjectGUID}}",
                    "rhmap/name": "data-mysql",
                    "rhmap/type":"environmentService"
                }
            },
            "spec": {
                "ports": [
                    {
                        "name": "data-mysql",
                        "port": 3306
                    }
                ],
                "selector": {
                    "name": "data-mysql"
                }
            }
        },
        {
            "kind": "PersistentVolumeClaim",
            "apiVersion": "v1",
            "metadata": {
                "name": "data-mysql"
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
                "kind":"DeploymentConfig",
                "name": "data-mysql",
                "labels": {
                    "name": "{{.ServiceName}}",
                    "rhmap/domain": "{{.Domain}}",
                    "rhmap/env": "{{.Env}}",
                    "rhmap/project": "{{.ProjectGUID}}",
                    "rhmap/name": "data-mysql",
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
                    "name": "data-mysql"
                },
                "template": {
                    "metadata": {
                        "name": "data-mysql",
                        "labels": {
                            "name": "data-mysql",
                            "rhmap/domain": "{{.Domain}}",
                            "rhmap/env": "{{.Env}}",
                            "rhmap/project": "{{.ProjectGUID}}"
                        }
                    },
                    "spec": {
                        "containers": [
                            {
                                "name": "data-mysql",
                                "image": "docker.io/rhmap/mysql:5.5-17",
                                "ports": [
                                    {
                                        "containerPort": 3306
                                    }
                                ],
                                "livenessProbe": {
                                    "tcpSocket": {
                                        "port": 3306
                                    },
                                    "initialDelaySeconds": 600,
                                    "timeoutSeconds": 5,
                                    "periodSeconds": 60,
                                    "successThreshold": 1,
                                    "failureThreshold": 2
                                },
                                "readinessProbe": {
                                    "exec": {
                                        "command": [
                                            "/bin/sh",
                                            "-ic",
                                            "MYSQL_PWD=\"{{index .Options "mysql_password"}}\" mysql -h 127.0.0.1 -u \"{{index .Options "mysql_user"}}\" -D \"{{index .Options "mysql_database"}}\" -e \"SELECT 1\""
                                        ]
                                    },
                                    "initialDelaySeconds": 0,
                                    "timeoutSeconds": 5,
                                    "periodSeconds": 10,
                                    "successThreshold": 1,
                                    "failureThreshold": 1
                                },
                                "env": [
                                    {
                                        "name": "MYSQL_USER",
                                        "value": "{{index .Options "mysql_user"}}"
                                    },
                                    {
                                        "name": "MYSQL_PASSWORD",
                                        "value": "{{index .Options "mysql_password"}}"
                                    },
                                    {
                                        "name": "MYSQL_DATABASE",
                                        "value": "{{index .Options "mysql_database"}}"
                                    }
                                ],
                                "resources": {
                                    "limits": {
                                        "cpu": "3200m",
                                        "memory": "1Gi"
                                    },
                                    "requests": {
                                        "cpu": "100m",
                                        "memory": "700Mi"
                                    }
                                },
                                "volumeMounts": [
                                    {
                                        "name": "mysql-data",
                                        "mountPath": "/var/lib/mysql/data"
                                    }
                                ],
                                "imagePullPolicy": "IfNotPresent"
                            }
                        ],
                        "volumes": [
                            {
                                "name": "mysql-data",
                                "persistentVolumeClaim": {
                                    "claimName": "data-mysql"
                                }
                            }
                        ]
                    }
                }
            }
        }
    ],
    "parameters": [],
    "labels": {
        "template": "mysql-persistent-template"
    }
}
{{end}}
`
