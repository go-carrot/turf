<a href="https://engineering.carrot.is/"><p align="center"><img src="https://cloud.githubusercontent.com/assets/2105067/24525319/d3d26516-1567-11e7-9506-7611b3287d53.png" alt="Go Carrot" width="350px" align="center;" /></p></a>
# Turf

Turf is a library that works with [Surf](https://github.com/go-carrot/surf) to generate controllers.

# Rest Controllers

The subpackage `rest` inside of this repository defines the types of controllers to handle various [model types](https://github.com/carrot/restful-api-spec#determine-interface-model-types), as defined in [Carrot's Restful API Spec](https://github.com/carrot/restful-api-spec).

#### Base Models

> Base Models are models that can be accessed directly, and are not dependent on the relation of any models.
>
> [Full Definition](https://github.com/carrot/restful-api-spec#base-models)

```go
func NewPostsController() *rest.BaseController {
	return &rest.BaseController{
		GetModel: func() surf.Model {
			return models.NewPost()
		},
	}
}
```

#### One-to-One Models

> One to one models are models(a) who exist only to be associated to another model(b), and the model(b) can only reference a single model(a).
>
> [Full Definition](https://github.com/carrot/restful-api-spec#one-to-one-models)

```go
func NewPostsVideoController() *rest.OneToOneController {
	return &rest.OneToOneController{
		NestedModelNameSingular: "video",
		ForeignReference:        "video_id", // The value in the BaseModel that references the NestedModel
		GetBaseModel: func() surf.Model {
			return models.NewPost()
		},
		GetNestedModel: func() surf.Model {
			return models.NewVideo()
		},
	}
}
```

#### One-to-Many Models

> One to many models are models(a) that exist to be associated to another model(b), but model(b) can reference multiple models(a).
> 
> [Full Definition](https://github.com/carrot/restful-api-spec#one-to-many-models)

```go
func NewAuthorPostsController() *rest.OneToManyController {
	return &rest.OneToManyController{
		NestedForeignReference: "author_id", // The value in the NestedModel that references the BaseModel
		GetBaseModel: func() surf.Model {
			return models.NewAuthor()
		},
		GetNestedModel: func() surf.Model {
			return models.NewPost()
		},
		BelongsTo: func(baseModel, nestedModel surf.Model) bool {
			return nestedModel.(*models.Post).AuthorId == baseModel.(*models.Author).Id
		},
	}
}
```

#### Many-to-Many Models

> Many to many models are models who are responsible for associating two other models (model(a) to model(b)). These models can contain additional information about the association, but that is optional. 
> 
> [Full Definition](https://github.com/carrot/restful-api-spec#many-to-many-models)

```go
func NewPostTagsController() *rest.ManyToManyController {
	return &rest.ManyToManyController{
		BaseModelForeignReference:   "post_id", // The BaseModel reference in the RelationModel
		NestedModelForeignReference: "tag_id",  // The NestedModel reference in the RelationModel
		GetBaseModel: func() surf.Model {
			return models.NewPost()
		},
		GetNestedModel: func() surf.Model {
			return models.NewTag()
		},
		GetRelationModel: func() surf.Model {
			return models.NewPostTag()
		},
	}
}
```

## Lifecycle Hooks

All Rest models have a field named `LifecycleHooks` that can be set to give control at a certain point in the lifecycle of a method.

Usage is detailed in [this file](https://github.com/go-carrot/turf/blob/br.readme/rest/lifecycle_hooks.go).

## Method Whitelists

All Rest models have a field named `MethodWhiteList` that can be set with a slice of strings.

```go
rest.BaseController{
    GetModel: func() surf.Model {
        return models.NewPost()
    },
    MethodWhiteList: []string{turf.INDEX, turf.SHOW},
}
```

If `MethodWhiteList` is not set, all supported methods get registered upon calling `controller.Register`.

# Controller Registration

All Controllers have a `Register` method that will automatically register the controller to a [httprouter.Router](https://github.com/julienschmidt/httprouter).

This also allows middleware to be passed in.

```go
router := httprouter.New()
controllers.NewPostsController().Register(router, middleware.Global)
http.ListenAndServe(":8080", router)
```

# License

[MIT](LICENSE.md)
