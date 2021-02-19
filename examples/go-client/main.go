/*
Copyright 2018-2021 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"log"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/types"

	"github.com/gravitational/trace"
	"github.com/pborman/uuid"
)

var (
	crtPath    = "certs/access-admin.crt"
	keyPath    = "certs/access-admin.key"
	casPath    = "certs/access-admin.cas"
	idFilePath = "certs/access-admin-identity"
	// Create valid tlsConfig here to use TLS Provider
	tlsConfig *tls.Config
)

func main() {
	ctx := context.Background()
	log.Printf("Starting Teleport client...")

	clt, err := client.New(ctx, client.Config{
		// Addrs can be auth, proxy, or webproxy addresses. Each will be dialed until one
		// provides a successful connection.
		Addrs: []string{"localhost:3080", "localhost:3024", "localhost:3025"},
		// Multiple credentials can be tried by providing credentialProviders. The first
		// provider to provide valid credentials will be used to authenticate the client.
		Credentials: []client.Credentials{
			client.LoadIdentityFile(idFilePath),
			// client.LoadKeyPair(crtPath, keyPath, casPath),
			// client.LoadTLS(tlsConfig),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer clt.Close()

	if err := demoClient(ctx, clt); err != nil {
		log.Printf("error(s) in demoClient: %v", err)
	}
}

func demoClient(ctx context.Context, clt *client.Client) (err error) {
	// Create a new access request for the `access-admin` user to use the `admin` role.
	accessReq, err := types.NewAccessRequest(uuid.New(), "access-admin", "admin")
	if err != nil {
		return trace.Wrap(err, "failed to make new access request")
	}
	if err = clt.CreateAccessRequest(ctx, accessReq); err != nil {
		return trace.Wrap(err, "failed to create access request")
	}
	log.Printf("Created access request: %v", accessReq)

	defer func() {
		if err2 := clt.DeleteAccessRequest(ctx, accessReq.GetName()); err2 != nil {
			err = trace.NewAggregate([]error{err, err2}...)
			log.Println("Failed to delete access request")
			return
		}
		log.Println("Deleted access request")
	}()

	// Approve the access request as if this was another party.
	if err = clt.SetAccessRequestState(ctx, types.AccessRequestUpdate{
		RequestID: accessReq.GetName(),
		State:     types.RequestState_APPROVED,
	}); err != nil {
		return trace.Wrap(err, "failed to accept request")
	}
	log.Printf("Approved access request")

	return nil
}
