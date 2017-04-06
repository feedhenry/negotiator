package templates

//DataMysqlConfigJob runs a kubernetes job to create a mysql user/pass/db
var DataMysqlConfigJob = `
{{define "data-mysql-job"}}
{
  "apiVersion": "batch/v1",
  "kind": "Job",
  "metadata": {
    "name": "{{if isset . "name"}}{{ index . "name"}}{{end}-mysql-dataconfig-job",
	  "labels": {
		  "rhmap/name":"datamysqlconfig",
      "rhmap/type":"job"
	  }
  },
  "spec": {
	"activeDeadlineSeconds": 120,
    "template": {
      "metadata": {
        "name": "{{if isset . "name"}}{{ index . "name"}}{{end}-dataconfig-job"
      },
      "spec": {
        "containers": [
          {
            "name": "datamysqlconfig",
            "image": "feedhenry/negotiator:0.0.14",
            "command": ["jobs",
              "datamysqlconfig",
              "--host={{if isset . "dbhost"}}{{ index . "dbhost"}}{{end}}",
              "--admin-username={{if isset . "admin-username"}}{{ index . "admin-username"}}{{end}}",
              "--admin-password={{if isset . "admin-password"}}{{ index . "admin-password"}}{{end}}",
              "--admin-database={{if isset . "admin-database"}}{{ index . "admin-database"}}{{end}}",
              "--user-username={{if isset . "user-username"}}{{ index . "user-username"}}{{end}}",
              "--user-password={{if isset . "user-password"}}{{ index . "user-password"}}{{end}}",
              "--user-database={{if isset . "user-database"}}{{ index . "user-database"}}{{end}}"
            ]
          }
        ],
        "restartPolicy": "Never"
      }
    }
  }
}
{{end}}`
