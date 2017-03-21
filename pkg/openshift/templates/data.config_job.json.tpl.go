package templates

var DataConfigJob = `
{{define "data-job"}}
{
  "apiVersion": "batch/v1",
  "kind": "Job",
  "metadata": {
    "name": "{{index . "name"}}-dataconfig-job",
	"labels": {
		"rhmap/name":"dataconfig"
	}
  },
  "spec": {
	"activeDeadlineSeconds": 20,  
    "template": {
      "metadata": {
        "name": "{{index . "name"}}-dataconfig-job"
      },
      "spec": {
        "containers": [
          {
            "name": "dataconfig",
            "image": "feedhenry/negotiator:0.0.5",
            "command": ["jobs",	
              "dataconfig",
			  "--admin-user={{if isset . "admin-user"}}{{ index . "admin-user"}}{{end}}",
			  "--admin-pass={{if isset . "admin-pass"}}{{ index . "admin-pass"}}{{end}}",
			  "--database={{if isset . "database"}}{{ index . "database"}}{{end}}",
			  "--database-user={{if isset . "database-user"}}{{ index . "database-user"}}{{end}}",
			  "--database-pass={{if isset . "database-pass"}}{{ index . "database-pass"}}{{end}}",
			  "--dbhost={{if isset . "dbhost"}}{{ index . "dbhost"}}{{end}}",
			  "--replicaSet={{if isset . "replicaSet"}}{{ index . "replicaSet"}}{{end}}"
            ]
          }
        ],
        "restartPolicy": "Never"
      }
    }
  }
}
{{end}}`
