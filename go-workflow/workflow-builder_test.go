package goworkflow_test

import (
	"context"
	"errors"
	"log"
	"math"
	"testing"
	"time"

	goworkflow "github.com/metaphi-org/go-workflow/go-workflow"
	"github.com/stretchr/testify/assert"
)

type Config struct{}

type Data struct {
	A        string
	B        string
	C        string
	Combined string
}

type customContext struct {
	context.Context
}

func getCustomContext() customContext {
	return customContext{context.TODO()}
}

func TestWorkflowCycleDetectionAndFailureHandling(t *testing.T) {
	ctx := getCustomContext()

	compA := goworkflow.MakeComponent("A", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.A = "A"
		})
		return nil
	})

	compB := goworkflow.MakeComponent("B", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.B = "B"
		})
		return nil
	})

	compC := goworkflow.MakeComponent("C", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.C = "C"
		})
		return nil
	})

	compD := goworkflow.MakeComponent("D", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B + d.C
		})
		return nil
	})

	wf := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	componentA := wf.AddComponent(compA)
	componentB := wf.AddComponent(compB)
	componentC := wf.AddComponent(compC)
	componentD := wf.AddComponent(compD)

	componentC.AddDependencies(componentA)
	componentD.AddDependencies(componentB, componentC)
	componentA.AddDependencies(componentD)

	_, st, err := wf.Execute(ctx, Config{}, &Data{})

	assert.Error(t, err)
	assert.Equal(t, st, goworkflow.ERROR)

	wf = goworkflow.NewWorkflow[customContext, Config, Data](ctx)

	componentA = wf.AddComponent(compA)
	componentB = wf.AddComponent(compB)
	componentC = wf.AddComponent(compC)
	componentD = wf.AddComponent(compD)

	componentD.AddDependencies(componentB, componentC, componentA)

	data, st, err := wf.Execute(ctx, Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "ABC", data.Combined)

	wf = goworkflow.NewWorkflow[customContext, Config, Data](ctx)

	compE := goworkflow.MakeComponent("E", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		return errors.New("this is not implemented")
	})

	compF := goworkflow.MakeComponent("F", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = "F"
		})
		return nil
	})
	componentE := wf.AddComponent(compE)
	componentF := wf.AddComponent(compF)
	componentF.AddDependencies(componentE)
	_, st, err = wf.Execute(ctx, Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.ERROR)
	assert.Equal(t, goworkflow.ERROR, componentE.Status().Status)
	assert.NotEqual(t, "F", data.Combined)
	assert.Equal(t, goworkflow.ERROR, componentF.Status().Status)
}

type Data2 struct {
	Run1     string
	Run2     string
	Combined string
}

func TestExample(t *testing.T) {
	ctx := getCustomContext()

	wf1 := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	cA1 := wf1.AddComponent(goworkflow.MakeComponent("A1", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(5 * time.Second)
		dt.Update(func(d *Data) {
			d.A = "A1"
		})
		return nil
	}))
	cB1 := wf1.AddComponent(goworkflow.MakeComponent("B1", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.B = "B1"
		})
		return nil
	}))
	cC1 := wf1.AddComponent(goworkflow.MakeComponent("C1", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B
		})
		return nil
	}))
	cC1.AddDependencies(cA1, cB1)

	wf2 := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	cA2 := wf2.AddComponent(goworkflow.MakeComponent("A2", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.A = "A2"
		})
		return nil
	}))
	cB2 := wf2.AddComponent(goworkflow.MakeComponent("B2", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(2 * time.Second)
		dt.Update(func(d *Data) {
			d.B = "B2"
		})
		return nil
	}))
	cC2 := wf2.AddComponent(goworkflow.MakeComponent("C2", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B
		})
		return nil
	}))
	cC2.AddDependencies(cA2, cB2)

	wfCombined := goworkflow.NewWorkflow[customContext, Config, Data2](ctx)
	c1 := wfCombined.AddComponent(goworkflow.MakeComponent("WF1", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data2]) error {
		data := &Data{}
		_, _, err := wf1.Execute(ctx, Config{}, data)
		if err != nil {
			return err
		}
		dt.Update(func(d *Data2) {
			d.Run1 = data.Combined
		})
		return nil
	}))
	c2 := wfCombined.AddComponent(goworkflow.MakeComponent("WF2", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data2]) error {
		time.Sleep(4 * time.Second)
		data := &Data{}
		_, _, err := wf2.Execute(ctx, Config{}, data)
		if err != nil {
			return err
		}
		dt.Update(func(d *Data2) {
			d.Run2 = data.Combined
		})
		return nil
	}))
	c3 := wfCombined.AddComponent(goworkflow.MakeComponent("Combine", nil, func(ctx customContext, input any, dt *goworkflow.DataTracker[Config, Data2]) error {
		dt.Update(func(d *Data2) {
			d.Combined = dt.GetData().Run1 + dt.GetData().Run2
		})
		return nil
	}))
	c3.AddDependencies(c1, c2)

	startTime := time.Now().Unix()

	data, st, err := wfCombined.Execute(ctx, Config{}, &Data2{})

	timeTaken := time.Now().Unix() - startTime

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "A1B1A2B2", data.Combined)
	assert.True(t, math.Abs(float64(timeTaken-6)) <= 0.01)
}

func TestCustomInput(t *testing.T) {
	wf := goworkflow.NewWorkflow[context.Context, Config, Data](context.TODO())

	type CustomInput1 struct {
		Name string
	}

	wf.AddComponent(
		goworkflow.MakeComponent(
			"A",
			CustomInput1{
				Name: "Narender",
			},
			func(ctx context.Context, input CustomInput1, dt *goworkflow.DataTracker[Config, Data]) error {
				log.Println("Name: ", input.Name)
				return nil
			},
		),
	)

	_, st, err := wf.Execute(context.TODO(), Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
}
