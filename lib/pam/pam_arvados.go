// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io/ioutil"
	"log/syslog"

	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
	"golang.org/x/sys/unix"
)

/*
#cgo LDFLAGS: -lpam -fPIC
#include <security/pam_ext.h>
char *stringindex(char** a, int i);
const char *get_user(pam_handle_t *pamh);
const char *get_authtoken(pam_handle_t *pamh);
*/
import "C"

func main() {}

func init() {
	if err := unix.Prctl(syscall.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		newLogger(false).WithError(err).Warn("unable to disable ptrace")
	}
}

//export pam_sm_authenticate
func pam_sm_authenticate(pamh *C.pam_handle_t, flags, cArgc C.int, cArgv **C.char) C.int {
	runtime.GOMAXPROCS(1)
	logger := newLogger(flags&C.PAM_SILENT == 0)
	cUsername := C.get_user(pamh)
	if cUsername == nil {
		return C.PAM_USER_UNKNOWN
	}

	cToken := C.get_authtoken(pamh)
	if cToken == nil {
		return C.PAM_AUTH_ERR
	}

	argv := make([]string, cArgc)
	for i := 0; i < int(cArgc); i++ {
		argv[i] = C.GoString(C.stringindex(cArgv, C.int(i)))
	}

	err := authenticate(logger, C.GoString(cUsername), C.GoString(cToken), argv)
	if err != nil {
		logger.WithError(err).Error("authentication failed")
		return C.PAM_AUTH_ERR
	}
	return C.PAM_SUCCESS
}

func authenticate(logger logrus.FieldLogger, username, token string, argv []string) error {
	hostname := ""
	apiHost := ""
	insecure := false
	for idx, arg := range argv {
		if idx == 0 {
			apiHost = arg
		} else if idx == 1 {
			hostname = arg
		} else if arg == "insecure" {
			insecure = true
		} else {
			logger.Warnf("unkown option: %s\n", arg)
		}
	}
	logger.Debugf("username=%q arvados_api_host=%q hostname=%q insecure=%t", username, apiHost, hostname, insecure)
	if apiHost == "" || hostname == "" {
		logger.Warnf("cannot authenticate: config error: arvados_api_host and hostname must be non-empty")
		return errors.New("config error")
	}
	arv := &arvados.Client{
		Scheme:    "https",
		APIHost:   apiHost,
		AuthToken: token,
		Insecure:  insecure,
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancel()
	var vms arvados.VirtualMachineList
	err := arv.RequestAndDecodeContext(ctx, &vms, "GET", "arvados/v1/virtual_machines", nil, arvados.ListOptions{
		Limit: 2,
		Filters: []arvados.Filter{
			{"hostname", "=", hostname},
		},
	})
	if err != nil {
		return err
	}
	if len(vms.Items) == 0 {
		return fmt.Errorf("no results for hostname %q", hostname)
	} else if len(vms.Items) > 1 {
		return fmt.Errorf("multiple results for hostname %q", hostname)
	} else if vms.Items[0].Hostname != hostname {
		return fmt.Errorf("looked up hostname %q but controller returned record with hostname %q", hostname, vms.Items[0].Hostname)
	}
	var user arvados.User
	err = arv.RequestAndDecodeContext(ctx, &user, "GET", "arvados/v1/users/current", nil, nil)
	if err != nil {
		return err
	}
	var links arvados.LinkList
	err = arv.RequestAndDecodeContext(ctx, &links, "GET", "arvados/v1/links", nil, arvados.ListOptions{
		Limit: 10000,
		Filters: []arvados.Filter{
			{"link_class", "=", "permission"},
			{"name", "=", "can_login"},
			{"tail_uuid", "=", user.UUID},
			{"head_uuid", "=", vms.Items[0].UUID},
			{"properties.username", "=", username},
		},
	})
	if err != nil {
		return err
	}
	if len(links.Items) < 1 || links.Items[0].Properties["username"] != username {
		return errors.New("permission denied")
	}
	logger.Debugf("permission granted based on link with UUID %s", links.Items[0].UUID)
	return nil
}

func newLogger(stderr bool) *logrus.Logger {
	logger := logrus.New()
	if !stderr {
		logger.Out = ioutil.Discard
	}
	if hook, err := lSyslog.NewSyslogHook("udp", "localhost:514", syslog.LOG_AUTH|syslog.LOG_INFO, "pam_arvados"); err != nil {
		logger.Hooks.Add(hook)
	}
	return logger
}
