package design

import (
	. "github.com/goadesign/goa/design" // Use . imports to enable the DSL
	. "github.com/goadesign/goa/design/apidsl"
)

var _ = API("negotiator", func() { // API defines the microservice endpoint and
	Title("Negotiator")                                // other global properties. There should be one
	Description("A negotiator between RHMAP and OSCP") // and exactly one API definition appearing in
	Scheme("http")                                     // the design.
	Origin("*", func() {
		Methods("GET", "POST", "DELETE", "PUT")
	})
	Host("localhost:3000")
	Produces("application/json")
	Consumes("application/json")
})

var _ = Resource("template", func() {
	BasePath("/templates")
	Action("list", func() {
		Description("list available templates") // with its path, parameters (both path
		Routing(GET("/"))                       // parameters and querystring values) and payload
		Response(OK, func() {
			Media(CollectionOf(TemplateType))
		}) // Responses define the shape and status code
	})
})

var _ = Resource("service", func() { // Resources group related API endpoints
	BasePath("/service")              // together. They map to REST resources for REST
	Action("deployTemplate", func() { // Actions define a single API endpoint together
		Description("deploy a service template to an environment namespace") // with its path, parameters (both path
		Routing(POST("/deploy/:template/:namespace"))                        // parameters and querystring values) and payload
		Params(func() {                                                      // (shape of the request body).
			Param("namespace", String, "the name of the environment")
			Param("template", String, "the name of the template you want to deploy")
		})
		Payload(Deploy)
		Response(OK, DeployResponse, func() {
			Status(201)
		}) // Responses define the shape and status code
		Response(NotFound) // of HTTP responses.
		Response(BadRequest)
	})
	Action("delete", func() { // Actions define a single API endpoint together
		Description("delete a service from an environment") // with its path, parameters (both path
		Routing(DELETE("/:environment/:serviceName"))       // parameters and querystring values) and payload
		Params(func() {
			Param("environment", String, "the name of the environment")
			Param("serviceName", String, "the identifier for the service")
		})
		Response(NoContent)  // Responses define the shape and status code
		Response(BadRequest) // of HTTP responses.
		Response(NotFound)
	})
	Action("update", func() { // Actions define a single API endpoint together
		Description("update a service in an environment") // with its path, parameters (both path
		Routing(PUT("/:environment/:serviceName"))        // parameters and querystring values) and payload
		Payload(DeployUpdate)
		Response(OK, DeployResponse, func() {
			Status(200)
		}) // Responses define the shape and status code
		Response(BadRequest) // of HTTP responses.
	})
})

var Deploy = Type("DeployPayload", func() {
	Attribute("target", Target)
	Attribute("myattribute", String)
	Attribute("route", String)
	Attribute("projectGuid", String)
	Attribute("cloudAppGuid", String)
	Attribute("domain", String)
	Attribute("replicas", Integer)
	Attribute("repo", Repo)
	Attribute("serviceName", String)
	Attribute("envVars", ArrayOf(EnvVar))
	Required("target")
})

var EnvVar = Type("EnvVar", func() {
	Attribute("name", String)
	Attribute("value", String)
})

var Target = Type("Target", func() {
	Attribute("host", String)
	Attribute("token", String)
})

var Repo = Type("Repo", func() {
	Attribute("loc", String, "the location of the git repo")
	Attribute("ref", String, "the git ref to use. Example master")
})

var DeployUpdate = Type("UpdatePayload", func() {
	Attribute("id", String)
	Attribute("href", String)
	Attribute("name", String)
	Required("id")
})

var DeployResponse = MediaType("DeployResponse", func() {
	Attribute("serviceName", String)
	Attribute("watchURL", String)
	Attribute("route", String)
	View("default", func() {
		Attribute("serviceName", String)
		Attribute("watchURL", String)
		Attribute("route", String)
	})
})

var TemplateType = MediaType("TemplateSummary", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("labels", HashOf(String, String))
	Attribute("dependsOn", ArrayOf(String))
	View("default", func() {
		Attribute("name", String)
		Attribute("description", String)
		Attribute("labels", HashOf(String, String))
		Attribute("dependsOn", ArrayOf(String))
	})
})
