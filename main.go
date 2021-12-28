package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"text/template"

	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/apigateway"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudfunctions"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Filebase64OrPanicArgs struct {
	// Path to the directory containing the source code. e.g. "./pkg/helloworld"
	Path         string
	Project      string
	Region       string
	FunctionName string
}

func filebase64OrPanic(args *Filebase64OrPanicArgs) pulumi.StringInput {
	tmpl, err := template.ParseFiles(args.Path)
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, args)
	if err != nil {
		panic(err)
	}

	return pulumi.String(base64.StdEncoding.EncodeToString(buf.Bytes()))
}

type Service struct {
	pulumi.ResourceState
}

type ServiceArgs struct {
	Project string
	Region  string
	Path    string
}

func NewService(ctx *pulumi.Context, name string, args *ServiceArgs, opts ...pulumi.ResourceOption) (*Service, error) {
	if args == nil {
		return nil, errors.New("missing one or more required arguments")
	}
	if args.Project == "" {
		return nil, errors.New("missing project argument")
	}
	if args.Region == "" {
		return nil, errors.New("missing region argument")
	}
	if args.Path == "" {
		return nil, errors.New("missing path argument")
	}

	service := &Service{}
	err := ctx.RegisterComponentResource("bluprince13:gcp:Service", name, service, opts...)
	if err != nil {
		return nil, err
	}

	bucket, err := storage.NewBucket(ctx, "bucket", &storage.BucketArgs{
		Location: pulumi.String("EU"),
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	bucketObject, err := storage.NewBucketObject(ctx, "go-zip", &storage.BucketObjectArgs{
		Bucket: bucket.Name,
		Source: pulumi.NewFileArchive(args.Path),
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	function, err := cloudfunctions.NewFunction(ctx, "function", &cloudfunctions.FunctionArgs{
		Name:                pulumi.String("function"),
		SourceArchiveBucket: bucket.Name,
		Runtime:             pulumi.String("go116"),
		SourceArchiveObject: bucketObject.Name,
		EntryPoint:          pulumi.String("Handler"),
		TriggerHttp:         pulumi.Bool(true),
		AvailableMemoryMb:   pulumi.Int(128),
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	_, err = cloudfunctions.NewFunctionIamMember(ctx, "invoker", &cloudfunctions.FunctionIamMemberArgs{
		Project:       function.Project,
		CloudFunction: function.Name,
		Role:          pulumi.String("roles/cloudfunctions.invoker"),
		Member:        pulumi.String("allUsers"),
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	api, err := apigateway.NewApi(ctx, "api", &apigateway.ApiArgs{
		ApiId: pulumi.String("api"),
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	apiConfig, err := apigateway.NewApiConfig(ctx, "apiConfig", &apigateway.ApiConfigArgs{
		Api:         api.ApiId,
		ApiConfigId: pulumi.String("cfg"),
		OpenapiDocuments: apigateway.ApiConfigOpenapiDocumentArray{
			&apigateway.ApiConfigOpenapiDocumentArgs{
				Document: &apigateway.ApiConfigOpenapiDocumentDocumentArgs{
					Path: pulumi.String("openapi.yaml"),
					Contents: filebase64OrPanic(&Filebase64OrPanicArgs{
						Path:         "openapi.yaml",
						Project:      args.Project,
						Region:       args.Region,
						FunctionName: "function",
					}),
				},
			},
		},
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	gateway, err := apigateway.NewGateway(ctx, "gateway", &apigateway.GatewayArgs{
		ApiConfig: apiConfig.ID(),
		GatewayId: pulumi.String("gateway"),
	}, pulumi.Parent(service))
	if err != nil {
		panic(err)
	}

	ctx.Export("url", gateway.DefaultHostname)

	return service, nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		NewService(ctx, "myService", &ServiceArgs{
			Project: "project",
			Region:  "europe-west2",
			Path:    "pkg/helloworld",
		})
		return nil
	})
}
