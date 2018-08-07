package main

import (
	// "fmt"
	"os"
	"strings"

	"github.com/cloudboss/figya/pkg/figya"
	"github.com/mitchellh/mapstructure"
)

type Command struct {
	Execute string `json:"execute"`
	Creates string `json:"creates"`
	Removes string `json:"removes"`
}

func New(params map[string]interface{}) (figya.Module, error) {
	var command Command
	err := mapstructure.Decode(params, &command)
	if err != nil {
		return nil, err
	}
	return &command, nil
}

func (c *Command) Name() string {
	return "command"
}

func (c *Command) Run() *figya.Result {
	parts := strings.Fields(c.Execute)
	command := parts[0]
	args := parts[1:]
	return figya.DoIf(
		c.Name(),
		func() (bool, error) {
			return c.done()
		},
		func() *figya.Result {
			commandOutput, err := figya.RunCommand(command, args...)
			if err != nil {
				errMsg := err.Error()
				return &figya.Result{
					Module: c.Name(),
					Error:  &errMsg,
				}
			}
			succeeded := commandOutput.ExitStatus == 0
			var errMsg *string
			if !succeeded {
				errMsg = &commandOutput.Stderr
			}
			return &figya.Result{
				Succeeded:    succeeded,
				Changed:      true,
				Error:        errMsg,
				Module:       c.Name(),
				ModuleOutput: &commandOutput,
			}
		},
	)
}

func (c *Command) done() (bool, error) {
	if c.Creates == "" && c.Removes == "" {
		return false, nil
	}

	var predicates []figya.Predicate
	if c.Creates != "" {
		predicates = append(predicates, c.created)
	}
	if c.Removes != "" {
		predicates = append(predicates, c.removed)
	}

	var results []bool
	for _, predicate := range predicates {
		result, err := predicate()
		if err != nil {
			return false, err
		}
		results = append(results, result)
	}
	return figya.All(results), nil
}

func (c *Command) created() (bool, error) {
	_, err := os.Stat(c.Creates)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (c *Command) removed() (bool, error) {
	_, err := os.Stat(c.Removes)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
