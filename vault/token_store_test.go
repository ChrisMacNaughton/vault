package vault

import (
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/vault/logical"
)

func mockTokenStore(t *testing.T) (*Core, *TokenStore, string) {
	c, _, root := TestCoreUnsealed(t)
	ts, err := NewTokenStore(c)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return c, ts, root
}

func TestTokenStore_RootToken(t *testing.T) {
	_, ts, _ := mockTokenStore(t)

	te, err := ts.RootToken()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if te.ID == "" {
		t.Fatalf("missing ID")
	}

	out, err := ts.Lookup(te.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(out, te) {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_CreateLookup(t *testing.T) {
	c, ts, _ := mockTokenStore(t)

	ent := &TokenEntry{Path: "test", Policies: []string{"dev", "ops"}}
	if err := ts.Create(ent); err != nil {
		t.Fatalf("err: %v", err)
	}
	if ent.ID == "" {
		t.Fatalf("missing ID")
	}

	out, err := ts.Lookup(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(out, ent) {
		t.Fatalf("bad: %#v", out)
	}

	// New store should share the salt
	ts2, err := NewTokenStore(c)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Should still match
	out, err = ts2.Lookup(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(out, ent) {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_CreateLookup_ProvidedID(t *testing.T) {
	c, ts, _ := mockTokenStore(t)

	ent := &TokenEntry{
		ID:       "foobarbaz",
		Path:     "test",
		Policies: []string{"dev", "ops"},
	}
	if err := ts.Create(ent); err != nil {
		t.Fatalf("err: %v", err)
	}
	if ent.ID != "foobarbaz" {
		t.Fatalf("bad: %#v", ent)
	}

	out, err := ts.Lookup(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(out, ent) {
		t.Fatalf("bad: %#v", out)
	}

	// New store should share the salt
	ts2, err := NewTokenStore(c)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Should still match
	out, err = ts2.Lookup(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(out, ent) {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_Revoke(t *testing.T) {
	_, ts, _ := mockTokenStore(t)

	ent := &TokenEntry{Path: "test", Policies: []string{"dev", "ops"}}
	if err := ts.Create(ent); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := ts.Revoke("")
	if err.Error() != "cannot revoke blank token" {
		t.Fatalf("err: %v", err)
	}
	err = ts.Revoke(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := ts.Lookup(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_Revoke_Orphan(t *testing.T) {
	_, ts, _ := mockTokenStore(t)

	ent := &TokenEntry{Path: "test", Policies: []string{"dev", "ops"}}
	if err := ts.Create(ent); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent2 := &TokenEntry{Parent: ent.ID}
	if err := ts.Create(ent2); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := ts.Revoke(ent.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := ts.Lookup(ent2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(out, ent2) {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_RevokeTree(t *testing.T) {
	_, ts, _ := mockTokenStore(t)

	ent1 := &TokenEntry{}
	if err := ts.Create(ent1); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent2 := &TokenEntry{Parent: ent1.ID}
	if err := ts.Create(ent2); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent3 := &TokenEntry{Parent: ent2.ID}
	if err := ts.Create(ent3); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent4 := &TokenEntry{Parent: ent2.ID}
	if err := ts.Create(ent4); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := ts.RevokeTree("")
	if err.Error() != "cannot revoke blank token" {
		t.Fatalf("err: %v", err)
	}
	err = ts.RevokeTree(ent1.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	lookup := []string{ent1.ID, ent2.ID, ent3.ID, ent4.ID}
	for _, id := range lookup {
		out, err := ts.Lookup(id)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out != nil {
			t.Fatalf("bad: %#v", out)
		}
	}
}

func TestTokenStore_RevokeAll(t *testing.T) {
	_, ts, _ := mockTokenStore(t)

	ent1 := &TokenEntry{}
	if err := ts.Create(ent1); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent2 := &TokenEntry{Parent: ent1.ID}
	if err := ts.Create(ent2); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent3 := &TokenEntry{}
	if err := ts.Create(ent3); err != nil {
		t.Fatalf("err: %v", err)
	}

	ent4 := &TokenEntry{Parent: ent3.ID}
	if err := ts.Create(ent4); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := ts.RevokeAll()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	lookup := []string{ent1.ID, ent2.ID, ent3.ID, ent4.ID}
	for _, id := range lookup {
		out, err := ts.Lookup(id)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out != nil {
			t.Fatalf("bad: %#v", out)
		}
	}
}

func TestTokenStore_HandleRequest_CreateToken_NoPolicy(t *testing.T) {
	_, ts, root := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root

	resp, err := ts.HandleRequest(req)
	if err != logical.ErrInvalidRequest {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Data["error"] != "token must have at least one policy" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_BadParent(t *testing.T) {
	_, ts, _ := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = "random"

	resp, err := ts.HandleRequest(req)
	if err != logical.ErrInvalidRequest {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Data["error"] != "parent token lookup failed" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken(t *testing.T) {
	_, ts, root := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root
	req.Data["policies"] = []string{"foo"}

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_RootID(t *testing.T) {
	_, ts, root := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root
	req.Data["id"] = "foobar"
	req.Data["policies"] = []string{"foo"}

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken != "foobar" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_NonRootID(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	testMakeToken(t, ts, root, "client", []string{"foo"})

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = "client"
	req.Data["id"] = "foobar"
	req.Data["policies"] = []string{"foo"}

	resp, err := ts.HandleRequest(req)
	if err != logical.ErrInvalidRequest {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Data["error"] != "root required to specify token id" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_NonRoot_Subset(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	testMakeToken(t, ts, root, "client", []string{"foo", "bar"})

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = "client"
	req.Data["policies"] = []string{"foo"}

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_NonRoot_InvalidSubset(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	testMakeToken(t, ts, root, "client", []string{"foo", "bar"})

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = "client"
	req.Data["policies"] = []string{"foo", "bar", "baz"}

	resp, err := ts.HandleRequest(req)
	if err != logical.ErrInvalidRequest {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Data["error"] != "child policies must be subset of parent" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_NonRoot_NoParent(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	testMakeToken(t, ts, root, "client", []string{"foo"})

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = "client"
	req.Data["no_parent"] = true
	req.Data["policies"] = []string{"foo"}

	resp, err := ts.HandleRequest(req)
	if err != logical.ErrInvalidRequest {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Data["error"] != "root required to create orphan token" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_CreateToken_Root_NoParent(t *testing.T) {
	_, ts, root := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root
	req.Data["no_parent"] = true
	req.Data["policies"] = []string{"foo"}

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}

	out, _ := ts.Lookup(resp.Auth.ClientToken)
	if out.Parent != "" {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_HandleRequest_CreateToken_Metadata(t *testing.T) {
	_, ts, root := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root
	req.Data["policies"] = []string{"foo"}
	meta := map[string]string{
		"user":   "armon",
		"source": "github",
	}
	req.Data["meta"] = meta

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}

	out, _ := ts.Lookup(resp.Auth.ClientToken)
	if !reflect.DeepEqual(out.Meta, meta) {
		t.Fatalf("bad: %#v", out)
	}
}

func TestTokenStore_HandleRequest_CreateToken_Lease(t *testing.T) {
	_, ts, root := mockTokenStore(t)

	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root
	req.Data["policies"] = []string{"foo"}
	req.Data["lease"] = "1h"

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}
	if resp.Auth.Lease != time.Hour {
		t.Fatalf("bad: %#v", resp)
	}
	if !resp.Auth.Renewable {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestTokenStore_HandleRequest_Revoke(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	testMakeToken(t, ts, root, "child", []string{"root", "foo"})
	testMakeToken(t, ts, "child", "sub-child", []string{"foo"})

	req := logical.TestRequest(t, logical.WriteOperation, "revoke/child")
	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	out, err := ts.Lookup("child")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("bad: %v", out)
	}

	// Sub-child should not exist
	out, err = ts.Lookup("sub-child")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("bad: %v", out)
	}
}

func TestTokenStore_HandleRequest_RevokeOrphan(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	testMakeToken(t, ts, root, "child", []string{"root", "foo"})
	testMakeToken(t, ts, "child", "sub-child", []string{"foo"})

	req := logical.TestRequest(t, logical.WriteOperation, "revoke-orphan/child")
	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	out, err := ts.Lookup("child")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("bad: %v", out)
	}

	// Sub-child should exist!
	out, err = ts.Lookup("sub-child")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("bad: %v", out)
	}
}

func TestTokenStore_HandleRequest_Lookup(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	req := logical.TestRequest(t, logical.ReadOperation, "lookup/"+root)
	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp == nil {
		t.Fatalf("bad: %#v", resp)
	}

	exp := map[string]interface{}{
		"id":       root,
		"policies": []string{"root"},
		"path":     "sys/root",
		"meta":     map[string]string(nil),
	}
	if !reflect.DeepEqual(resp.Data, exp) {
		t.Fatalf("bad: %#v exp: %#v", resp.Data, exp)
	}
}

func TestTokenStore_HandleRequest_RevokePrefix(t *testing.T) {
	exp := mockExpiration(t)
	ts := exp.tokenStore

	// Create new token
	root, err := ts.RootToken()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a new token
	auth := &logical.Auth{
		ClientToken: root.ID,
		Lease:       time.Hour,
	}
	err = exp.RegisterAuth("auth/github/login", auth)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	req := logical.TestRequest(t, logical.WriteOperation, "revoke-prefix/auth/github/")
	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	out, err := ts.Lookup(root.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("bad: %v", out)
	}
}

func TestTokenStore_HandleRequest_LookupSelf(t *testing.T) {
	_, ts, root := mockTokenStore(t)
	req := logical.TestRequest(t, logical.ReadOperation, "lookup-self")
	req.ClientToken = root
	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp == nil {
		t.Fatalf("bad: %#v", resp)
	}

	exp := map[string]interface{}{
		"id":       root,
		"policies": []string{"root"},
		"path":     "sys/root",
		"meta":     map[string]string(nil),
	}
	if !reflect.DeepEqual(resp.Data, exp) {
		t.Fatalf("bad: %#v exp: %#v", resp.Data, exp)
	}
}

func TestTokenStore_HandleRequest_Renew(t *testing.T) {
	exp := mockExpiration(t)
	ts := exp.tokenStore

	// Create new token
	root, err := ts.RootToken()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a new token
	auth := &logical.Auth{
		ClientToken: root.ID,
		Lease:       time.Hour,
	}
	err = exp.RegisterAuth("sys/root", auth)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	req := logical.TestRequest(t, logical.WriteOperation, "renew/"+root.ID)
	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if !reflect.DeepEqual(resp.Auth, auth) {
		t.Fatalf("bad: %#v", resp)
	}
}

func testMakeToken(t *testing.T, ts *TokenStore, root, client string, policy []string) {
	req := logical.TestRequest(t, logical.WriteOperation, "create")
	req.ClientToken = root
	req.Data["id"] = client
	req.Data["policies"] = policy

	resp, err := ts.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken != client {
		t.Fatalf("bad: %#v", resp)
	}
}

func testCoreMakeToken(t *testing.T, c *Core, root, client string, policy []string) {
	req := logical.TestRequest(t, logical.WriteOperation, "auth/token/create")
	req.ClientToken = root
	req.Data["id"] = client
	req.Data["policies"] = policy

	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v %v", err, resp)
	}
	if resp.Auth.ClientToken != client {
		t.Fatalf("bad: %#v", resp)
	}
}
