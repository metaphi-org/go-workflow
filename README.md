# Go Workflow

A lightweight workflow management library for Go

### Overview

Go Workflow is a library that allows you to define and execute workflows with complex dependencies in a concurrent and efficient manner. It provides a simple and intuitive API for defining components, dependencies, and execution logic.

### Key Features

- **Components**: Define independent components with their own input and execution logic.
- **Generic**: Generic workflow config, data, context and component input are supported.
- **Dependencies**: Declare dependencies between components to control execution order, and prevent circular dependencies.
- **Concurrent Execution**: Components are executed concurrently for improved performance
- **Data**: Common data object is available across workflow components with concurrency safe write mechanism
- **Error Handling**: Robust error handling and propagation through the workflow

### Getting Started

#### Installation

```bash
go get github.com/metaphi-org/go-workflow
```

To use Go Workflow, simply import the library and create a new workflow instance:

```Go
import "github.com/metaphi-org/go-workflow/go-workflow"

func main() {
ctx := context.Background()
wf := goworkflow.NewWorkflow[context.Context, Config, Data](ctx)
// ...
}
```

#### Defining Components

Components are the building blocks of your workflow. Define a component by calling wf.AddComponent and use the `goworkflow.MakeComponent` to provide name and logic.

```Go
// component with no custom input
var cmp1 = wf.AddComponent(
	goworkflow.MakeComponent(
		"cmp1", // name of component
		nil,    // optional input of the component
		func(ctx context.Context, input any, dt *goworkflow.DataTracker[Config, Data]) error {
			log.Println(dt.GetData().A) // read data
			// logic here.
			dt.Update(func(d *Data) {
				// update data
				d.A = "A"
			})
			return nil
		},
	),
)

// component with custom input
var cmp2 = wf.AddComponent(
	goworkflow.MakeComponent(
		"cmp2",        // name of component
		CustomInput{}, // custom input
		func(ctx context.Context, input CustomInput, dt *goworkflow.DataTracker[Config, Data]) error {
			log.Println(input.Name)     // read custom input
			log.Println(dt.GetData().A) // read data
			// logic here.
			dt.Update(func(d *Data) {
				// update data
				d.A = "A"
			})
			return nil
		},
	),
)

// support for concurrency limiter using optional goworkflow.AddComponentConfig config
var concurrencyLimiter = limiter.NewConcurrencyLimiter(10)
var cmp3 = wf.AddComponent(
	goworkflow.MakeComponent(
		"cmp3", // name of component
		nil,    // no input
		func(ctx context.Context, input any, dt *goworkflow.DataTracker[Config, Data]) error {
			// logic here
			return nil
		},
	),
	&goworkflow.AddComponentConfig{
		ConcurrencyLimiter: concurrencyLimiter,
	},
)
```

#### Declaring Dependencies

Declare dependencies between components using the AddDependencies method:

```Go
cmp3.AddDependencies(cmp1, cmp2) // cmp3 will only execute once cmp1 and cmp2 are successfully executed
```

#### Executing the Workflow

Execute the workflow by calling wf.Execute and providing the required context, config, and data:

```Go
data, status, err := wf.Execute(ctx, myConfig, myData)
```

### Example

Refer to [this](./go-workflow/examples_test.go) file for design of document processing workflow using `goworkflow`.

### License

Go Workflow is released under the MIT License.

### Contributing

Contributions are welcome! Please open an issue or submit a pull request.
