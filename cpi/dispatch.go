package cpi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type Request struct {
	Method  string        `json:"method"`
	Args    []interface{} `json:"arguments"`
	Context Context       `json:"context"`
}

type Response struct {
	Result interface{}    `json:"result"`
	Error  *ResponseError `json:"error"`
	Log    string         `json:"log"`
}

type ResponseError struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	CanRetry bool   `json:"ok_to_retry"`
}

func Dispatch(req *Request, actionFunc interface{}) (*Response, error) {
	actionValue := reflect.ValueOf(actionFunc)
	actionType := actionValue.Type()

	argCount := len(req.Args)
	requiredArgCount := actionType.NumIn()

	if actionType.IsVariadic() {
		requiredArgCount--
	}

	if argCount < requiredArgCount {
		return nil, fmt.Errorf("Not enough arguments: have %d, want %d", argCount, requiredArgCount)
	}

	if argCount > requiredArgCount && !actionType.IsVariadic() {
		return nil, fmt.Errorf("Too many arguments: have %d, want %d", argCount, requiredArgCount)
	}

	var args []reflect.Value
	for i, arg := range req.Args {
		bytes, err := json.Marshal(arg)
		if err != nil {
			return nil, err
		}

		argValue := newArgValue(actionType, i)
		err = json.Unmarshal(bytes, argValue.Interface())
		if err != nil {
			return nil, err
		}

		args = append(args, reflect.Indirect(argValue))
	}

	actionResult := actionValue.Call(args)

	return newResponse(actionResult)
}

func newArgValue(actionType reflect.Type, index int) reflect.Value {
	argCount := actionType.NumIn()

	var argType reflect.Type
	if actionType.IsVariadic() && index >= argCount-1 {
		argType = actionType.In(argCount - 1).Elem()
	} else {
		argType = actionType.In(index)
	}

	return reflect.New(argType)
}

func newResponse(result []reflect.Value) (*Response, error) {
	if len(result) < 1 || len(result) > 2 {
		return nil, errors.New("Too many action results")
	}

	var resultValue, errValue reflect.Value
	switch len(result) {
	case 1:
		if isErrorValue(result[0]) {
			errValue = result[0]
		} else {
			resultValue = result[0]
		}

	case 2:
		resultValue, errValue = result[0], result[1]
		if !isErrorValue(errValue) {
			return nil, errors.New("Action error is not an error")
		}

	default:
		return nil, errors.New("Invalid action results")
	}

	resp := &Response{}
	if resultValue.IsValid() {
		resp.Result = resultValue.Interface()
	}

	if errValue.IsValid() && !errValue.IsNil() {
		err := errValue.Interface().(error)
		resp.Error = &ResponseError{Message: err.Error()}
	}

	return resp, nil
}

func isErrorValue(value reflect.Value) bool {
	return value.Type().Implements(reflect.TypeOf((*error)(nil)).Elem())
}
