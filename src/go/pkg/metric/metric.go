/*
** Zabbix
** Copyright (C) 2001-2020 Zabbix SIA
**
** This program is free software; you can redistribute it and/or modify
** it under the terms of the GNU General Public License as published by
** the Free Software Foundation; either version 2 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful,
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
** GNU General Public License for more details.
**
** You should have received a copy of the GNU General Public License
** along with this program; if not, write to the Free Software
** Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
**/

// Package metric provides an interface for describing a schema of metric's parameters.
package metric

import (
	"fmt"
	"reflect"
	"strconv"
	"zabbix.com/pkg/zbxerr"
)

type paramKind int

const (
	kindSession paramKind = iota
	kindConn
	kindGeneral
)

const (
	required = true
	optional = false
)

// Param stores parameters' metadata.
type Param struct {
	name         string
	kind         paramKind
	required     bool
	validator    Validator
	defaultValue *string
}

func newParam(name string, kind paramKind, required bool, validator Validator) *Param {
	if name == "" {
		panic("name cannot be empty")
	}

	return &Param{
		name:         name,
		kind:         kind,
		required:     required,
		validator:    validator,
		defaultValue: nil,
	}
}

// NewParam creates a new parameter with given name and validator.
// Returns a pointer.
func NewParam(name string) *Param {
	return newParam(name, kindGeneral, optional, nil)
}

// NewConnParam creates a new connection parameter with given name and validator.
// Returns a pointer.
func NewConnParam(name string) *Param {
	return newParam(name, kindConn, optional, nil)
}

// WithSession transforms a connection typed parameter to a dual purpose parameter which can be either
// a connection parameter or session name.
// Returns a pointer.
func (p *Param) WithSession() *Param {
	if p.kind != kindConn {
		panic("only connection typed parameter can be transformed to session")
	}

	p.kind = kindSession

	return p
}

// WithDefault sets the default value for a parameter.
// It panics if a default value is specified for a required parameter.
func (p *Param) WithDefault(value string) *Param {
	if p.required {
		panic("default value cannot be applied to a required parameter")
	}

	p.defaultValue = &value

	return p
}

// WithValidator sets a validator for a parameter
func (p *Param) WithValidator(validator Validator) *Param {
	p.validator = validator

	return p
}

// SetRequired makes the parameter mandatory.
// It panics if default value is specified for required parameter.
func (p *Param) SetRequired() *Param {
	if p.defaultValue != nil {
		panic("required parameter cannot have a default value")
	}

	p.required = required

	return p
}

// Metric stores a description of a metric and its parameters.
type Metric struct {
	description string
	params      []*Param
	varParam    bool
}

func ordinalize(num int) string {
	var ordinalDictionary = map[int]string{
		0: "th",
		1: "st",
		2: "nd",
		3: "rd",
		4: "th",
		5: "th",
		6: "th",
		7: "th",
		8: "th",
		9: "th",
	}

	if ((num % 100) >= 11) && ((num % 100) <= 13) {
		return strconv.Itoa(num) + "th"
	}

	return strconv.Itoa(num) + ordinalDictionary[num]
}

// New creates an instance of a Metric and returns a pointer to it.
// It panics if a metric is not satisfied to one of the following rules:
// 1. Parameters must be named (and names must be unique).
// 2. It's forbidden to duplicate parameters' names.
// 3. Session must be placed first.
// 4. Connection parameters must be placed in a row.
func New(description string, params []*Param, varParam bool) *Metric {
	connParamIdx := -1

	if len(params) > 0 {
		if params[0].kind != kindGeneral {
			connParamIdx = 0
		}
	}

	paramsMap := make(map[string]bool)

	for i, p := range params {
		if _, exists := paramsMap[p.name]; exists {
			panic(fmt.Sprintf("name of parameter %q must be unique", p.name))
		}

		paramsMap[p.name] = true

		if i > 0 && p.kind == kindSession {
			panic("session must be placed first")
		}

		if p.kind == kindConn {
			if i-connParamIdx > 1 {
				panic("parameters describing a connection must be placed in a row")
			}

			connParamIdx = i
		}

		if p.validator != nil && p.defaultValue != nil {
			if err := p.validator.Validate(p.defaultValue); err != nil {
				panic(fmt.Sprintf("invalid default value %q for %s parameter %q: %s",
					*p.defaultValue, ordinalize(i+1), p.name, err.Error()))
			}
		}
	}

	return &Metric{
		description: description,
		params:      params,
		varParam:    varParam,
	}
}

func findSession(name string, sessions interface{}) (session interface{}) {
	v := reflect.ValueOf(sessions)
	if v.Kind() != reflect.Map {
		panic("sessions must be map of strings")
	}

	for _, key := range v.MapKeys() {
		if name == key.String() {
			session = v.MapIndex(key).Interface()
			break
		}
	}

	return
}

func mergeWithSessionData(out map[string]string, metricParams []*Param, session interface{}) error {
	v := reflect.ValueOf(session)
	for i := 0; i < v.NumField(); i++ {
		var p *Param = nil

		val := v.Field(i).String()

		j := 0
		for j = range metricParams {
			if metricParams[j].name == v.Type().Field(i).Name {
				p = metricParams[j]
				break
			}
		}

		ordNum := ordinalize(j + 1)

		if p == nil {
			panic(fmt.Sprintf("cannot find parameter %q in schema", v.Type().Field(i).Name))
		}

		if val == "" {
			if p.required {
				return zbxerr.ErrorTooFewParameters.Wrap(
					fmt.Errorf("%s parameter %q is required", ordNum, p.name))
			}

			if p.defaultValue != nil {
				val = *p.defaultValue
			}
		}

		if p.validator != nil {
			if err := p.validator.Validate(&val); err != nil {
				return zbxerr.New(fmt.Sprintf("invalid %s parameter %q", ordNum, p.name)).Wrap(err)
			}
		}

		out[p.name] = val
	}

	return nil
}

// EvalParams returns a mapping of parameters' names to their values passed to a plugin and/or
// sessions specified in the configuration file.
// If a session is configured, then an other connection parameters must not be accepted and an error will be returned.
// Also it returns error in following cases:
// * incorrect number of parameters are passed;
// * missing required parameter;
// * value validation is failed.
func (m *Metric) EvalParams(rawParams []string, sessions interface{}) (params map[string]string, err error) {
	var (
		session interface{}
		val     *string
	)

	if !m.varParam && len(rawParams) > len(m.params) {
		return nil, zbxerr.ErrorTooManyParameters
	}

	if len(rawParams) > 0 && m.params[0].kind == kindSession {
		session = findSession(rawParams[0], sessions)
	}

	params = make(map[string]string)

	for i, p := range m.params {
		kind := p.kind
		if kind == kindSession {
			if session != nil {
				continue
			}

			kind = kindConn
		}

		val = nil
		skipConnIfSessionIsSet := !(session != nil && kind == kindConn)
		ordNum := ordinalize(i + 1)

		if i >= len(rawParams) || rawParams[i] == "" {
			if p.required && skipConnIfSessionIsSet {
				return nil, zbxerr.ErrorTooFewParameters.Wrap(
					fmt.Errorf("%s parameter %q is required", ordNum, p.name))
			}

			if p.defaultValue != nil && skipConnIfSessionIsSet {
				val = p.defaultValue
			}
		} else {
			val = &rawParams[i]
		}

		if val == nil {
			continue
		}

		if p.validator != nil && skipConnIfSessionIsSet {
			if err = p.validator.Validate(val); err != nil {
				return nil, zbxerr.New(fmt.Sprintf("invalid %s parameter %q", ordNum, p.name)).Wrap(err)
			}
		}

		if kind == kindConn {
			if session == nil {
				params[p.name] = *val
			} else {
				return nil, zbxerr.ErrorInvalidParams.Wrap(
					fmt.Errorf("%s parameter %q cannot be passed along with session", ordNum, p.name))
			}
		}

		if kind == kindGeneral {
			params[p.name] = *val
		}
	}

	// Fill connection parameters with data from a session
	if session != nil {
		if err = mergeWithSessionData(params, m.params, session); err != nil {
			return nil, err
		}
	}

	return params, nil
}

// MetricSet stores a mapping of keys to metrics.
type MetricSet map[string]*Metric

// List returns an array of metrics' keys and their descriptions suitable to pass to plugin.RegisterMetrics.
func (ml MetricSet) List() (list []string) {
	for key, metric := range ml {
		list = append(list, key, metric.description)
	}

	return
}
