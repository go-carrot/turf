<a href="https://engineering.carrot.is/"><p align="center"><img src="https://cloud.githubusercontent.com/assets/2105067/24525319/d3d26516-1567-11e7-9506-7611b3287d53.png" alt="Go Carrot" width="350px" align="center;" /></p></a>
# Turf

Turf is a library that works with [Surf](https://github.com/go-carrot/surf) to generate controllers.

## Rest Controllers

The subpackage `rest` inside of this repository defines the types of controllers to handle various [model types](https://github.com/carrot/restful-api-spec#determine-interface-model-types), as defined in [Carrot's Restful API Spec](https://github.com/carrot/restful-api-spec).

### Base Models

> Base Models are models that can be accessed directly, and are not dependent on the relation of any models.
>
> [Full Definition](https://github.com/carrot/restful-api-spec#base-models)

```go
type TasksController struct {
	turf.Controller
}

func NewTasksController() *TasksController {
	return &TasksController{
		Controller: rest.BaseController{
			GetModel: func() surf.Model {
				return models.NewTask()
			},
		},
	}
}
```

### One-to-One Models

> One to one models are models(a) who exist only to be associated to another model(b), and the model(b) can only reference a single model(a).
>
> [Full Definition](https://github.com/carrot/restful-api-spec#one-to-one-models)

### One-to-Many Models

> One to many models are models(a) that exist to be associated to another model(b), but model(b) can reference multiple models(a).
> 
> [Full Definition](https://github.com/carrot/restful-api-spec#one-to-many-models)

### Many-to-Many Models

> Many to many models are models who are responsible for associating two other models (model(a) to model(b)). These models can contain additional information about the association, but that is optional. 
> 
> [Full Definition](https://github.com/carrot/restful-api-spec#many-to-many-models)

## License

[MIT](LICENSE.md)
