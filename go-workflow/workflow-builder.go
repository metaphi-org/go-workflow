package goworkflow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Status string

const PENDING Status = "PENDING"
const DONE Status = "DONE"
const ERROR Status = "ERROR"

type DataTracker[C any, T any] struct {
	lock   sync.Mutex
	Config C
	data   *T
}

func (d *DataTracker[C, T]) GetData() T {
	return *d.data
}

func (d *DataTracker[C, T]) Update(cb func(*T)) {
	d.lock.Lock()
	defer d.lock.Unlock()
	cb(d.data)
}

type ComponentFunction[CT context.Context, C any, T any] func(CT, *DataTracker[C, T]) error

type componentStatus struct {
	Status       Status
	ErrorMessage string
}
type component[CT context.Context, C any, T any] struct {
	id            string
	Name          string
	addDependency func(d *component[CT, C, T])
	executor      ComponentFunction[CT, C, T]
	status        componentStatus
}

/* AddDependencies: current component required all d as dependency, so they will be executed before it*/
func (c *component[CT, C, T]) AddDependencies(d ...*component[CT, C, T]) {
	for _, dep := range d {
		c.addDependency(dep)
	}
}

func (c *component[CT, C, T]) Status() componentStatus {
	return c.status
}

type dependencyChannel struct {
	DependentCount int
	Dependency     chan Status
}

type dependencyManager struct {
	lk sync.Mutex
	// Directed edge: dependencyId -> componentId -> bool
	dependencyGraph    map[string]map[string]bool
	dependencyChannels map[string]dependencyChannel
	componentIdToName  map[string]string
}

func (d *dependencyManager) AddLink(componentId string, dependencyId string) {
	d.lk.Lock()
	defer d.lk.Unlock()

	if _, ok := d.dependencyGraph[dependencyId]; !ok {
		d.dependencyGraph[dependencyId] = make(map[string]bool)
	}
	d.dependencyGraph[dependencyId][componentId] = true
}

func (d *dependencyManager) hasCircularDependency() (bool, string) {
	visited := make(map[string]bool)
	currentPath := make(map[string]bool)
	circularDependencyPath := make([]string, 0)

	var dfs func(string) bool
	dfs = func(component string) bool {
		if currentPath[component] {
			circularDependencyPath = append(circularDependencyPath, component)
			return true
		}
		if visited[component] {
			return false
		}
		visited[component] = true
		currentPath[component] = true

		for child := range d.dependencyGraph[component] {
			if dfs(child) {
				circularDependencyPath = append(circularDependencyPath, component)
				return true
			}
		}

		currentPath[component] = false
		return false
	}

	for component := range d.dependencyGraph {
		if dfs(component) {
			break
		}
	}

	circularDependencyNames := []string{}
	for _, componentId := range circularDependencyPath {
		circularDependencyNames = append(circularDependencyNames, d.componentIdToName[componentId])
	}
	slices.Reverse(circularDependencyNames)

	if len(circularDependencyPath) > 0 {
		return true, fmt.Sprintf("Circular dependency detected: %v", strings.Join(circularDependencyNames, " -> "))
	}

	return false, ""
}

func (d *dependencyManager) BuildChannels() error {
	d.lk.Lock()
	defer d.lk.Unlock()

	if yes, msg := d.hasCircularDependency(); yes {
		return errors.New(msg)
	}

	for dependencyId := range d.dependencyGraph {
		for _, dep := range d.dependencyGraph[dependencyId] {
			if dep {
				if _, ok := d.dependencyChannels[dependencyId]; !ok {
					d.dependencyChannels[dependencyId] = dependencyChannel{DependentCount: 0}
				}
				t := d.dependencyChannels[dependencyId]
				t.DependentCount++
				d.dependencyChannels[dependencyId] = t
			}
		}
	}

	for key := range d.dependencyChannels {
		t := d.dependencyChannels[key]
		t.Dependency = make(chan Status, t.DependentCount)
		d.dependencyChannels[key] = t
	}
	return nil
}

func (d *dependencyManager) UpdateStatus(componentId string, status Status) {
	d.lk.Lock()
	defer d.lk.Unlock()

	for i := 0; i < d.dependencyChannels[componentId].DependentCount; i++ {
		d.dependencyChannels[componentId].Dependency <- status
	}
}

/* await all dependencies of component with componentId as Id */
func (d *dependencyManager) WaitDependencies(componentId string) Status {
	wg := sync.WaitGroup{}
	overallStatus := DONE
	for dependencyId := range d.dependencyGraph {
		if _, ok := d.dependencyGraph[dependencyId][componentId]; ok {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				st := <-d.dependencyChannels[k].Dependency
				if st == ERROR {
					overallStatus = ERROR
				}
			}(dependencyId)
		}
	}
	wg.Wait()
	return overallStatus
}

type Workflow[CT context.Context, C any, T any] struct {
	executed          bool
	componentsMap     map[string]*component[CT, C, T]
	dependencyManager *dependencyManager
}

/* for testing purposes: reset workflow and dependencies*/
func (wf *Workflow[CT, C, T]) ResetWorkflow() {
	wf.executed = false
	wf.dependencyManager.dependencyGraph = map[string]map[string]bool{}
	wf.dependencyManager.dependencyChannels = map[string]dependencyChannel{}
}

func (wf *Workflow[CT, C, T]) AddComponent(name string, executor ComponentFunction[CT, C, T]) *component[CT, C, T] {
	if len(name) == 0 {
		panic("name cannot be empty")
	}
	id := uuid.New().String()
	var addDependencyWrapper = func(d *component[CT, C, T]) {
		wf.dependencyManager.AddLink(id, d.id)
	}
	component := component[CT, C, T]{id: id, Name: name, executor: executor, status: componentStatus{Status: PENDING}, addDependency: addDependencyWrapper}
	wf.componentsMap[id] = &component
	wf.dependencyManager.componentIdToName[id] = name
	return &component
}

func (wf *Workflow[CT, C, T]) Execute(ctx CT, config C, data *T) (*T, Status, error) {
	if wf.executed {
		return nil, ERROR, errors.New("workflow already executed")
	}
	if data == nil {
		return nil, ERROR, errors.New("data cannot be nil")
	}
	dataTracker := DataTracker[C, T]{Config: config, data: data}
	err := wf.dependencyManager.BuildChannels()
	if err != nil {
		log.Println("Workflow.Execute:Error:", err)
		return data, ERROR, err
	}

	wg := sync.WaitGroup{}

	for _, cmp := range wf.componentsMap {
		wg.Add(1)
		go func(c *component[CT, C, T]) {
			defer wg.Done()
			executionStatus := DONE
			errMsg := ""
			// check if all dependencies are done
			overallStatus := wf.dependencyManager.WaitDependencies(c.id)
			if overallStatus == ERROR {
				log.Println("Workflow.Execute:Error:Dependency failed for component:", c.id)
				executionStatus = ERROR
				errMsg = fmt.Sprintf("component dependency failed: %s", c.id)
			}

			// execute the component if dependencies are resolved
			if executionStatus == DONE {
				err := c.executor(ctx, &dataTracker)
				if err != nil {
					log.Println("Workflow.Execute:Error:Component execution failed for component:", c.id, err)
					executionStatus = ERROR
					errMsg = err.Error()
				}
			}
			// update the status of the component
			c.status = componentStatus{
				Status:       executionStatus,
				ErrorMessage: errMsg,
			}
			wf.dependencyManager.UpdateStatus(c.id, executionStatus)
		}(cmp)
	}
	wg.Wait()

	wf.executed = true

	finalStatus := DONE
	for _, cmp := range wf.componentsMap {
		if cmp.status.Status == ERROR {
			finalStatus = ERROR
			break
		}
	}
	return data, finalStatus, nil
}

func NewWorkflow[CT context.Context, C any, T any](ctx CT) *Workflow[CT, C, T] {
	return &Workflow[CT, C, T]{
		executed:      false,
		componentsMap: map[string]*component[CT, C, T]{},
		dependencyManager: &dependencyManager{
			dependencyGraph:    map[string]map[string]bool{},
			dependencyChannels: map[string]dependencyChannel{},
			componentIdToName:  map[string]string{},
		},
	}
}
