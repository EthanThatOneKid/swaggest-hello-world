package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/openapi"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui/v3cdn"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func main() {
	// Init API documentation schema.
	apiSchema := &openapi.Collector{}
	apiSchema.Reflector().SpecEns().Info.Title = "Basic Example"
	apiSchema.Reflector().SpecEns().Info.WithDescription("This app showcases a trivial REST API.")
	apiSchema.Reflector().SpecEns().Info.Version = "v1.2.3"

	// Setup request decoder and validator.
	validatorFactory := jsonschema.NewFactory(apiSchema, apiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.ApplyDefaults = true
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)

	// Create router.
	r := chirouter.NewWrapper(chi.NewRouter())

	// Setup middlewares.
	r.Use(
		middleware.Recoverer,                          // Panic recovery.
		nethttp.OpenAPIMiddleware(apiSchema),          // Documentation collector.
		request.DecoderMiddleware(decoderFactory),     // Request decoder setup.
		request.ValidatorMiddleware(validatorFactory), // Request validator setup.
		response.EncoderMiddleware,                    // Response encoder setup.
		gzip.Middleware,                               // Response compression with support for direct gzip pass through.
	)

	// Add use case handler to router.
	r.Method(http.MethodPost, "/doubler/{param1}", nethttp.NewHandler(helloWorld()))

	// Swagger UI endpoint at /docs.
	r.Method(http.MethodGet, "/docs/openapi.json", apiSchema)
	r.Mount("/docs", v3cdn.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	// Start server.
	// TODO: Get port from CLI flags.
	log.Println("http://localhost:8000/docs")
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal(err)
	}
}

// Configure use case interactor in application layer.
type myInput struct {
	Param1 int    `path:"param1" description:"Parameter in resource path." multipleOf:"2"`
	Param2 string `json:"param2" description:"Parameter in resource body."`
}

type myOutput struct {
	Value1 int    `json:"value1"`
	Value2 string `json:"value2"`
}

func helloWorld() usecase.IOInteractorOf[myInput, myOutput] {
	u := usecase.NewInteractor(func(ctx context.Context, input myInput, output *myOutput) error {
		if input.Param1%2 != 0 {
			return status.InvalidArgument
		}

		// Do something to set output based on input.
		output.Value1 = input.Param1 + input.Param1
		output.Value2 = input.Param2 + input.Param2

		return nil
	})

	// Additional properties can be configured for purposes of automated documentation.
	u.SetTitle("Doubler")
	u.SetDescription("Doubler doubles parameter values.")
	u.SetTags("transformation")
	u.SetExpectedErrors(status.InvalidArgument)
	// u.SetIsDeprecated(true)
	// TODO: Reference latest example.
	// https://github.com/swaggest/rest/blob/v0.2.61/_examples/basic/main.go

	return u
}
