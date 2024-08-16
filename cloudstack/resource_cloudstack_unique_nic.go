//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCloudStackUniqueNIC() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackUniqueNICCreate,
		Read:   resourceCloudStackUniqueNICRead,
		Delete: resourceCloudStackUniqueNICDelete,

		Schema: map[string]*schema.Schema{
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackUniqueNICCreate(d *schema.ResourceData, meta interface{}) error {

	cs := meta.(*cloudstack.CloudStackClient)

	listNicksParms := cs.Nic.NewListNicsParams(
		d.Get("virtual_machine_id").(string),
	)

	nics, err := cs.Nic.ListNics(listNicksParms)

	if err != nil {
		return fmt.Errorf("Error listing nics: %s", err)
	}

	networkId := d.Get("network_id").(string)
	vmId := d.Get("virtual_machine_id").(string)

	containsNic := false

	var otherNics []*cloudstack.Nic

	for _, nic := range nics.Nics {
		if string(nic.Networkid) == networkId {
			containsNic = true
			break
		} else {
			otherNics = append(otherNics, nic)
		}

	}

	if containsNic {
		parametersDefault := cs.VirtualMachine.NewUpdateDefaultNicForVirtualMachineParams(
			d.Get("network_id").(string),
			vmId,
		)

		_, err := cs.VirtualMachine.UpdateDefaultNicForVirtualMachine(parametersDefault)

		if err != nil {
			return fmt.Errorf("Error setting the nic with network id %s as default: %s", networkId, err)
		}

		for _, otherNic := range otherNics {
			parametersDelete := cs.VirtualMachine.NewRemoveNicFromVirtualMachineParams(
				otherNic.Id,
				vmId,
			)

			_, err := cs.VirtualMachine.RemoveNicFromVirtualMachine(parametersDelete)

			if err != nil {
				return fmt.Errorf("Error removing nic of network id %s: %s", networkId, err)
			}
		}
	} else {
		// Create a new parameter struct
		p := cs.VirtualMachine.NewAddNicToVirtualMachineParams(
			networkId,
			vmId,
		)

		// If there is a ipaddres supplied, add it to the parameter struct
		if ipaddress, ok := d.GetOk("ip_address"); ok {
			p.SetIpaddress(ipaddress.(string))
		}

		// Create and attach the new NIC
		r, err := Retry(10, retryableAddNicFunc(cs, p))
		if err != nil {
			return fmt.Errorf("Error creating the new NIC: %s", err)
		}

		found := false
		for _, n := range r.(*cloudstack.AddNicToVirtualMachineResponse).Nic {
			if n.Networkid == d.Get("network_id").(string) {
				d.SetId(n.Id)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Could not find NIC ID for network ID: %s", d.Get("network_id").(string))
		}
	}

	return resourceCloudStackUniqueNICRead(d, meta)
}

func resourceCloudStackUniqueNICRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(d.Get("virtual_machine_id").(string))
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Instance %s does no longer exist", d.Get("virtual_machine_id").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	// Read NIC info
	found := false
	for _, n := range vm.Nic {
		if n.Id == d.Id() {
			d.Set("ip_address", n.Ipaddress)
			d.Set("network_id", n.Networkid)
			d.Set("virtual_machine_id", vm.Id)
			found = true
			break
		}
	}

	if !found {
		log.Printf("[DEBUG] NIC for network ID %s does no longer exist", d.Get("network_id").(string))
		d.SetId("")
	}

	return nil
}

func resourceCloudStackUniqueNICDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VirtualMachine.NewRemoveNicFromVirtualMachineParams(
		d.Id(),
		d.Get("virtual_machine_id").(string),
	)

	// Remove the NIC
	_, err := cs.VirtualMachine.RemoveNicFromVirtualMachine(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting NIC: %s", err)
	}

	return nil
}
