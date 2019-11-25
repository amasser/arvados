// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"

	"git.curoverse.com/arvados.git/lib/controller/rpc"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UserSuite{})

type UserSuite struct {
	FederationSuite
}

func (s *UserSuite) TestLoginClusterUserList(c *check.C) {
	s.cluster.ClusterID = "local"
	s.cluster.Login.LoginCluster = "zzzzz"
	s.fed = New(s.cluster)
	s.addDirectRemote(c, "zzzzz", rpc.NewConn("zzzzz", &url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_API_HOST")}, true, rpc.PassthroughTokenProvider))

	for _, updateFail := range []bool{false, true} {
		for _, opts := range []arvados.ListOptions{
			{Offset: 0, Limit: -1, Select: nil},
			{Offset: 1, Limit: 1, Select: nil},
			{Offset: 0, Limit: 2, Select: []string{"uuid"}},
			{Offset: 0, Limit: 2, Select: []string{"uuid", "email"}},
		} {
			c.Logf("updateFail %v, opts %#v", updateFail, opts)
			spy := arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
			stub := &arvadostest.APIStub{Error: errors.New("local cluster failure")}
			if updateFail {
				s.fed.local = stub
			} else {
				s.fed.local = rpc.NewConn(s.cluster.ClusterID, spy.URL, true, rpc.PassthroughTokenProvider)
			}
			userlist, err := s.fed.UserList(s.ctx, opts)
			if updateFail && err == nil {
				// All local updates fail, so the only
				// cases expected to succeed are the
				// ones with 0 results.
				c.Check(userlist.Items, check.HasLen, 0)
				c.Check(stub.Calls(nil), check.HasLen, 0)
			} else if updateFail {
				c.Logf("... err %#v", err)
				calls := stub.Calls(stub.UserBatchUpdate)
				if c.Check(calls, check.HasLen, 1) {
					c.Logf("... stub.UserUpdate called with options: %#v", calls[0].Options)
					shouldUpdate := map[string]bool{
						"uuid":       false,
						"email":      true,
						"first_name": true,
						"last_name":  true,
						"is_admin":   true,
						"is_active":  true,
						"prefs":      true,
						// can't safely update locally
						"owner_uuid":   false,
						"identity_url": false,
						// virtual attrs
						"full_name":  false,
						"is_invited": false,
					}
					if opts.Select != nil {
						// Only the selected
						// fields (minus uuid)
						// should be updated.
						for k := range shouldUpdate {
							shouldUpdate[k] = false
						}
						for _, k := range opts.Select {
							if k != "uuid" {
								shouldUpdate[k] = true
							}
						}
					}
					var uuid string
					for uuid = range calls[0].Options.(arvados.UserBatchUpdateOptions).Updates {
					}
					for k, shouldFind := range shouldUpdate {
						_, found := calls[0].Options.(arvados.UserBatchUpdateOptions).Updates[uuid][k]
						c.Check(found, check.Equals, shouldFind, check.Commentf("offending attr: %s", k))
					}
				}
			} else {
				updates := 0
				for _, d := range spy.RequestDumps {
					d := string(d)
					if strings.Contains(d, "PATCH /arvados/v1/users/batch") {
						c.Check(d, check.Matches, `(?ms).*Authorization: Bearer `+arvadostest.SystemRootToken+`.*`)
						updates++
					}
				}
				c.Check(err, check.IsNil)
				c.Check(updates, check.Equals, 1)
				c.Logf("... response items %#v", userlist.Items)
			}
		}
	}
}

// userAttrsCachedFromLoginCluster must have an entry for every field
// in the User struct.
func (s *UserSuite) TestUserAttrsUpdateWhitelist(c *check.C) {
	buf, err := json.Marshal(&arvados.User{})
	c.Assert(err, check.IsNil)
	var allFields map[string]interface{}
	err = json.Unmarshal(buf, &allFields)
	c.Assert(err, check.IsNil)
	for k := range allFields {
		_, ok := userAttrsCachedFromLoginCluster[k]
		c.Check(ok, check.Equals, true, check.Commentf("field name %q missing from userAttrsCachedFromLoginCluster", k))
	}
}
