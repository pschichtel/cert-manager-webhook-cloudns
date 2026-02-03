/*
 Copyright 2019 IXON B.V.

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
	"fmt"
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/go-acme/lego/v4/platform/config/env"
	"github.com/pschichtel/cert-manager-webhook-cloudns/cloudns"
	restclient "k8s.io/client-go/rest"
)

const ProviderName = "cloudns"

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("Please set the GROUP_NAME env variable.")
	}

	client, err := cloudns.NewCloudnsClient(cloudns.DefaultBaseUrl)
	if err != nil {
		panic("Failed to setup cloudns client: " + err.Error())
	}
	ttl := env.GetOrDefaultInt("CLOUDNS_TTL", 60)

	// Start webhook server
	cmd.RunWebhookServer(GroupName,
		&clouDNSProviderSolver{
			*client,
			ttl,
		},
	)
}

// clouDNSProviderSolver implements webhook.Solver
// and will allow cert-manager to create & delete
// DNS TXT records for the DNS01 Challenge
type clouDNSProviderSolver struct {
	client cloudns.CloudnsClient
	ttl    int
}

func resolveCredentials() (*cloudns.CloudnsCredentials, error) {
	credentials := cloudns.CloudnsCredentials{}
	credentials.IdType = env.GetOrDefaultString("CLOUDNS_AUTH_ID_TYPE", "auth-id")
	if credentials.IdType != "auth-id" && credentials.IdType != "sub-auth-id" {
		return nil, fmt.Errorf("ClouDNS auth id type is not valid. Expected one of 'auth-id' or 'sub-auth-id' but was: '%s'", credentials.IdType)
	}

	credentials.Id = env.GetOrFile("CLOUDNS_AUTH_ID")
	if credentials.Id == "" {
		return nil, fmt.Errorf("CLOUDNS_AUTH_ID(_FILE) is required")
	}
	credentials.Password = env.GetOrFile("CLOUDNS_AUTH_PASSWORD")
	if credentials.Password == "" {
		return nil, fmt.Errorf("CLOUDNS_AUTH_ID(_FILE) is required")
	}

	return &credentials, nil
}

func (c clouDNSProviderSolver) Name() string {
	return ProviderName
}

// Create TXT DNS record for DNS01
func (c clouDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	credentials, err := resolveCredentials()
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	zone, err := c.client.GetZone(credentials, ch.ResolvedFQDN)
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	err = c.client.AddTxtRecord(credentials, zone.Name, ch.ResolvedFQDN, ch.Key, c.ttl)
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	return nil
}

// Delete TXT DNS record for DNS01
func (c clouDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	credentials, err := resolveCredentials()
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	zone, err := c.client.GetZone(credentials, ch.ResolvedFQDN)
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	record, err := c.client.FindTxtRecord(credentials, zone.Name, ch.ResolvedFQDN)
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	if record == nil {
		return nil
	}

	err = c.client.RemoveTxtRecord(credentials, record.ID, zone.Name)
	if err != nil {
		return fmt.Errorf("ClouDNS: %v", err)
	}

	return nil
}

// Could be used to initialise connections or warm up caches, not needed in this case
func (c clouDNSProviderSolver) Initialize(kubeClientConfig *restclient.Config, stopCh <-chan struct{}) error {
	// NOOP
	return nil
}
