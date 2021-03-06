// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"reflect"
	"runtime"

	check "gopkg.in/check.v1"
)

// a Gocheck checker for testing the name of a function. Used with
// (*arvadostest.APIStub).Calls() to check that an HTTP request has
// been routed to the correct arvados.API method.
//
//	c.Check(bytes.NewBuffer().Read, isMethodNamed, "Read")
var isMethodNamed check.Checker = &chkIsMethodNamed{
	CheckerInfo: &check.CheckerInfo{
		Name:   "isMethodNamed",
		Params: []string{"obtained", "expected"},
	},
}

type chkIsMethodNamed struct{ *check.CheckerInfo }

func (*chkIsMethodNamed) Check(params []interface{}, names []string) (bool, string) {
	methodName := runtime.FuncForPC(reflect.ValueOf(params[0]).Pointer()).Name()
	regex := `.*\)\.` + params[1].(string) + `(-.*)?`
	return check.Matches.Check([]interface{}{methodName, regex}, names)
}
