/*
Copyright 2021 Gravitational, Inc.

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

package auth

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/services"

	"github.com/gravitational/trace"
)

// httpfallback.go holds endpoints that have been converted to gRPC
// but still need http fallback logic in the old client.

func (c *Client) GetTrustedCluster(ctx context.Context, name string) (types.TrustedCluster, error) {
	if resp, err := c.APIClient.GetTrustedCluster(ctx, name); err != nil {
		if !trace.IsNotImplemented(err) {
			return nil, trace.Wrap(err)
		}
	} else {
		return resp, nil
	}

	out, err := c.Get(ctx, c.Endpoint("trustedclusters", name), url.Values{})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	trustedCluster, err := services.UnmarshalTrustedCluster(out.Bytes())
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return trustedCluster, nil
}

func (c *Client) GetTrustedClusters(ctx context.Context) ([]types.TrustedCluster, error) {
	if resp, err := c.APIClient.GetTrustedClusters(ctx); err != nil {
		if !trace.IsNotImplemented(err) {
			return nil, trace.Wrap(err)
		}
	} else {
		return resp, nil
	}

	out, err := c.Get(ctx, c.Endpoint("trustedclusters"), url.Values{})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var items []json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &items); err != nil {
		return nil, trace.Wrap(err)
	}
	trustedClusters := make([]types.TrustedCluster, len(items))
	for i, bytes := range items {
		trustedCluster, err := services.UnmarshalTrustedCluster(bytes)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		trustedClusters[i] = trustedCluster
	}

	return trustedClusters, nil
}

// UpsertTrustedCluster creates or updates a trusted cluster.
func (c *Client) UpsertTrustedCluster(ctx context.Context, trustedCluster types.TrustedCluster) (types.TrustedCluster, error) {
	if resp, err := c.APIClient.UpsertTrustedCluster(ctx, trustedCluster); err != nil {
		if !trace.IsNotImplemented(err) {
			return nil, trace.Wrap(err)
		}
	} else {
		return resp, nil
	}

	trustedClusterBytes, err := services.MarshalTrustedCluster(trustedCluster)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	out, err := c.PostJSON(ctx, c.Endpoint("trustedclusters"), &upsertTrustedClusterReq{
		TrustedCluster: trustedClusterBytes,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return services.UnmarshalTrustedCluster(out.Bytes())
}

// DeleteTrustedCluster deletes a trusted cluster by name.
func (c *Client) DeleteTrustedCluster(ctx context.Context, name string) error {
	if err := c.APIClient.DeleteTrustedCluster(ctx, name); err != nil {
		if !trace.IsNotImplemented(err) {
			return trace.Wrap(err)
		}
	} else {
		return nil
	}

	_, err := c.Delete(ctx, c.Endpoint("trustedclusters", name))
	return trace.Wrap(err)
}
