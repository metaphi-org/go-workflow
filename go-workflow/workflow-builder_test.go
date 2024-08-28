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
	wf := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	componentA := wf.AddComponent("A", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.A = "A"
		})
		return nil
	})

	componentB := wf.AddComponent("B", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.B = "B"
		})
		return nil
	})

	componentC := wf.AddComponent("C", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.C = "C"
		})
		return nil
	})

	componentD := wf.AddComponent("D", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B + d.C
		})
		return nil
	})

	componentC.AddDependencies(componentA)
	componentD.AddDependencies(componentB, componentC)
	componentA.AddDependencies(componentD)

	_, st, err := wf.Execute(ctx, Config{}, &Data{})

	assert.Error(t, err)
	assert.Equal(t, st, goworkflow.ERROR)

	wf.ResetWorkflow()

	componentD.AddDependencies(componentB, componentC, componentA)

	data, st, err := wf.Execute(ctx, Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "ABC", data.Combined)

	wf.ResetWorkflow()

	componentE := wf.AddComponent("E", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		return errors.New("this is not implemented")
	})

	componentF := wf.AddComponent("F", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = "F"
		})
		return nil
	})
	componentF.AddDependencies(componentE)
	_, st, err = wf.Execute(ctx, Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.ERROR)
	assert.Equal(t, goworkflow.ERROR, componentE.Status().Status)
	assert.Equal(t, goworkflow.DONE, componentA.Status().Status)
	assert.NotEqual(t, "F", data.Combined)
	assert.Equal(t, goworkflow.ERROR, componentF.Status().Status)
}

func TestWorkflowTimes(t *testing.T) {
	ctx := getCustomContext()
	wf := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	wf.AddComponent("A", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(10 * time.Second)
		dt.Update(func(d *Data) {
			d.A = "A"
		})
		return nil
	})

	componentB := wf.AddComponent("B", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(2 * time.Second)
		dt.Update(func(d *Data) {
			d.B = "B"
		})
		return nil
	})

	componentC := wf.AddComponent("C", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(5 * time.Second)
		dt.Update(func(d *Data) {
			d.C = "C"
		})
		return nil
	})

	componentD := wf.AddComponent("D", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B + d.C
		})
		return nil
	})

	componentC.AddDependencies(componentB)
	componentD.AddDependencies(componentC)

	startTimeSeconds := time.Now().Unix()
	data, st, err := wf.Execute(ctx, Config{}, &Data{})

	timeTakenSeconds := time.Now().Unix() - startTimeSeconds
	log.Println("Time taken:", timeTakenSeconds)
	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "BC", data.Combined)
	assert.True(t, math.Abs(float64(timeTakenSeconds-10)) <= 0.01)
	assert.Equal(t, "A", data.A)
}

func TestExample(t *testing.T) {
	// ctx := getCustomContext()
}
