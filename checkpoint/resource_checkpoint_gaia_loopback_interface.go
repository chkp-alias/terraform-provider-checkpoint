package checkpoint

import (
    "fmt"
    checkpoint "github.com/CheckPointSW/cp-mgmt-api-go-sdk/APIFiles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
    "log"
    "strings"

)
func resourceGaiaLoopbackInterface() *schema.Resource {   
    return gaiaReadResource("loopback-interface", &schema.Resource{
        Create: createGaiaLoopbackInterface,
        Read:   readGaiaLoopbackInterface,
        Update: updateGaiaLoopbackInterface,
        Delete: deleteGaiaLoopbackInterface,
        Importer: gaiaResourceImporter("loopback-interface", map[string]schema.ValueType{
            "virtual_system_id": schema.TypeInt,
            "name": schema.TypeString,
            "member_id": schema.TypeString,
        }, "name"),
        Schema: map[string]*schema.Schema{
            "debug": {
                Type:        schema.TypeBool,
                Optional:    true,
                Description: "Enable debug logging for this resource.",
            },
            "ipv4_address": {
                Type:        schema.TypeString,
                Optional:    true,
                Description: `Either this, 'ipv6-address' or both must be specified`,
            },
            "ipv4_mask_length": {
                Type:        schema.TypeInt,
                Optional:    true,
                Description: `N/A`,
            },
            "enabled": {
                Type:        schema.TypeBool,
                Optional:    true,
                Computed:    true,
                Description: `N/A`,
            },
            "ipv6_autoconfig": {
                Type:        schema.TypeBool,
                Optional:    true,
                Description: `N/A`,
            },
            "comments": {
                Type:        schema.TypeString,
                Optional:    true,
                Description: `N/A`,
            },
            "ipv6_address": {
                Type:        schema.TypeString,
                Optional:    true,
                Computed:    true,
                Description: `Either this, 'ipv4-address' or both must be specified`,
            },
            "ipv6_mask_length": {
                Type:        schema.TypeInt,
                Optional:    true,
                Description: `N/A`,
            },
            "virtual_system_id": {
                Type:        schema.TypeInt,
                Optional:    true,
                Description: `Virtual System ID. Relevant for VSNext setups`,
            },
            "name": {
                Type:        schema.TypeString,
                Optional:    true,
                Computed:    true,
                Description: `N/A`,
            },
            "member_id": {
                Type:        schema.TypeString,
                Optional:    true,
                Description: `Relevant for commands on Scalable and ElasticXL platforms only.<br>When member-id is provided in the login request,<br>show commands during the session will be executed on the specified member,<br>unless a different member-id is provided in a successive requests<br>Set operations will be performed on all members`,
            },
            "link_state": {
                Type:        schema.TypeBool,
                Computed:    true,
                Description: `N/A`,
            },
            "speed": {
                Type:        schema.TypeString,
                Computed:    true,
                Description: `N/A`,
            },
            "duplex": {
                Type:        schema.TypeString,
                Computed:    true,
                Description: `N/A`,
            },
            "tx_bytes": {
                Type:        schema.TypeInt,
                Computed:    true,
                Description: `N/A`,
            },
            "tx_packets": {
                Type:        schema.TypeInt,
                Computed:    true,
                Description: `N/A`,
            },
            "rx_bytes": {
                Type:        schema.TypeInt,
                Computed:    true,
                Description: `N/A`,
            },
            "rx_packets": {
                Type:        schema.TypeInt,
                Computed:    true,
                Description: `N/A`,
            },
            "mtu": {
                Type:        schema.TypeString,
                Computed:    true,
                Description: `N/A`,
            },
            "ipv6_local_link_address": {
                Type:        schema.TypeString,
                Computed:    true,
                Description: `N/A`,
            },
        },
    })
}

func createGaiaLoopbackInterface(d *schema.ResourceData, m interface{}) error {
    client := m.(*checkpoint.ApiClient)
    ensureDebugServerFromClient(client)

    payload := make(map[string]interface{})

    if v, ok := d.GetOk("ipv4_address"); ok {
        payload["ipv4-address"] = v.(string)
    }

    if v, ok := d.GetOk("ipv4_mask_length"); ok {
        payload["ipv4-mask-length"] = v.(int)
    }

    if v, ok := d.GetOkExists("enabled"); ok {
        payload["enabled"] = v.(bool)
    }

    if v, ok := d.GetOkExists("ipv6_autoconfig"); ok {
        payload["ipv6-autoconfig"] = v.(bool)
    }

    if v, ok := d.GetOk("comments"); ok {
        payload["comments"] = v.(string)
    }

    if v, ok := d.GetOk("ipv6_address"); ok {
        payload["ipv6-address"] = v.(string)
    }

    if v, ok := d.GetOk("ipv6_mask_length"); ok {
        payload["ipv6-mask-length"] = v.(int)
    }

    if v, ok := d.GetOk("virtual_system_id"); ok {
        payload["virtual-system-id"] = v.(int)
    }

    log.Println("Create LoopbackInterface - Map = ", safeCopyMap(payload))

    addLoopbackInterfaceRes, err := client.ApiCallSimple("add-loopback-interface", payload)
    // DEBUG: generic logger
    if resourceDebugEnabled(d) {
        success := err == nil && addLoopbackInterfaceRes.Success
        errMsg := ""
        if err != nil {
            errMsg = err.Error()
        } else if !addLoopbackInterfaceRes.Success {
            errMsg = addLoopbackInterfaceRes.ErrorMsg
        }

        var respData map[string]interface{}
        if err == nil {
            respData = addLoopbackInterfaceRes.GetData()
        }

        debugLogOperation(
            "loopback-interface",        // resource type
            "create",                       // operation
            "add-loopback-interface",         // API call name
            payload,                        // request payload
            respData,                       // response data (if any)
            success,
            errMsg,
        )
    }
    if err != nil {
        return fmt.Errorf("Failed to add loopback-interface: %v", err)
    }
    if !addLoopbackInterfaceRes.Success {
        if addLoopbackInterfaceRes.ErrorMsg != "" {
            return fmt.Errorf(addLoopbackInterfaceRes.ErrorMsg)
        }
        return fmt.Errorf("Unknown error occurred")
    }

    // Extract API-assigned fields from Create response before calling Read.
    if data := addLoopbackInterfaceRes.GetData(); data != nil {
        if v, exists := data["name"]; exists {
            d.Set("name", v)
        }
    }

    // Two-phase creation: Apply update-only fields if present
    hasUpdateOnlyFields := false
    updatePayload := map[string]interface{}{
    }

    if v, ok := d.GetOk("name"); ok {
        updatePayload["name"] = v.(string)
        hasUpdateOnlyFields = true
    }

    if hasUpdateOnlyFields {
        log.Println("Two-phase creation: Applying update-only fields - Map = ", safeCopyMap(updatePayload))
        
        setLoopbackInterfaceRes, err := client.ApiCallSimple("set-loopback-interface", updatePayload)
        if err != nil {
            return fmt.Errorf("Failed to apply update-only fields for loopback-interface: %v", err)
        }
        if !setLoopbackInterfaceRes.Success {
            return fmt.Errorf("Failed to apply update-only fields: %s", setLoopbackInterfaceRes.ErrorMsg)
        }
    }

    d.SetId(fmt.Sprintf("loopback-interface-" + acctest.RandString(10)))
    return readGaiaLoopbackInterface(d, m)
}

func readGaiaLoopbackInterface(d *schema.ResourceData, m interface{}) error {

    client := m.(*checkpoint.ApiClient)
    ensureDebugServerFromClient(client)

    payload := map[string]interface{}{}

    if v, ok := d.GetOk("virtual_system_id"); ok {
        payload["virtual-system-id"] = v.(int)
    }

   payload["name"] = d.Get("name")
    if v, ok := d.GetOk("member_id"); ok {
        payload["member-id"] = v.(string)
    }

    showLoopbackInterfaceRes, err := client.ApiCallSimple("show-loopback-interface", payload)
    // DEBUG: generic logger
    if resourceDebugEnabled(d) {
        success := err == nil && showLoopbackInterfaceRes.Success
        errMsg := ""
        if err != nil {
            errMsg = err.Error()
        } else if !showLoopbackInterfaceRes.Success {
            errMsg = showLoopbackInterfaceRes.ErrorMsg
        }

        var respData map[string]interface{}
        if err == nil {
            respData = showLoopbackInterfaceRes.GetData()
        }

        debugLogOperation(
            "loopback-interface",        // resource type
            "read",                       // operation
            "show-loopback-interface",         // API call name
            payload,                        // request payload
            respData,                       // response data (if any)
            success,
            errMsg,
        )
    }
    if err != nil {
        return fmt.Errorf("Failed to show loopback-interface: %v", err)
    }
    if !showLoopbackInterfaceRes.Success {
        if data := showLoopbackInterfaceRes.GetData(); data != nil {
            if code, exists := data["code"]; exists {
                if strings.Contains(strings.ToLower(code.(string)), "not_found") || strings.Contains(strings.ToLower(code.(string)), "object_not_found") {
                    d.SetId("")
                    return nil
                }
            }
        }
        return fmt.Errorf(showLoopbackInterfaceRes.ErrorMsg)
    }

    loopbackInterface := showLoopbackInterfaceRes.GetData()

    log.Println("Read LoopbackInterface - Show JSON = ", safeCopyMap(loopbackInterface))

        return gaiaReadState(d, resourceGaiaLoopbackInterface().Schema, loopbackInterface)
}

func updateGaiaLoopbackInterface(d *schema.ResourceData, m interface{}) error {

    client := m.(*checkpoint.ApiClient)
    ensureDebugServerFromClient(client)

    payload := map[string]interface{}{}

    if v, ok := d.GetOk("ipv4_address"); ok {
        payload["ipv4-address"] = v.(string)
    }

    if v, ok := d.GetOk("ipv4_mask_length"); ok {
        payload["ipv4-mask-length"] = v.(int)
    }

    if v, ok := d.GetOkExists("enabled"); ok {
        payload["enabled"] = v.(bool)
    }

    if v, ok := d.GetOkExists("ipv6_autoconfig"); ok {
        payload["ipv6-autoconfig"] = v.(bool)
    }

    if v, ok := d.GetOk("comments"); ok {
        payload["comments"] = v.(string)
    }

    if v, ok := d.GetOk("ipv6_address"); ok {
        payload["ipv6-address"] = v.(string)
    }

    if v, ok := d.GetOk("ipv6_mask_length"); ok {
        payload["ipv6-mask-length"] = v.(int)
    }

    if v, ok := d.GetOk("virtual_system_id"); ok {
        payload["virtual-system-id"] = v.(int)
    }

   payload["name"] = d.Get("name")
    setLoopbackInterfaceRes, err := client.ApiCallSimple("set-loopback-interface", payload)
    // DEBUG: generic logger
    if resourceDebugEnabled(d) {
        success := err == nil && setLoopbackInterfaceRes.Success
        errMsg := ""
        if err != nil {
            errMsg = err.Error()
        } else if !setLoopbackInterfaceRes.Success {
            errMsg = setLoopbackInterfaceRes.ErrorMsg
        }

        var respData map[string]interface{}
        if err == nil {
            respData = setLoopbackInterfaceRes.GetData()
        }

        debugLogOperation(
            "loopback-interface",        // resource type
            "update",                       // operation
            "set-loopback-interface",         // API call name
            payload,                        // request payload
            respData,                       // response data (if any)
            success,
            errMsg,
        )
    }
    if err != nil {
        return fmt.Errorf("Failed to set loopback-interface: %v", err)
    }
    if !setLoopbackInterfaceRes.Success {
        return fmt.Errorf(setLoopbackInterfaceRes.ErrorMsg)
    }

    return readGaiaLoopbackInterface(d, m)
}

func deleteGaiaLoopbackInterface(d *schema.ResourceData, m interface{}) error {

    client := m.(*checkpoint.ApiClient)
    ensureDebugServerFromClient(client)

    payload := map[string]interface{}{}

    if v, ok := d.GetOk("virtual_system_id"); ok {
        payload["virtual-system-id"] = v.(int)
    }

   payload["name"] = d.Get("name")
    deleteLoopbackInterfaceRes, err := client.ApiCallSimple("delete-loopback-interface", payload)
    // DEBUG: generic logger
    if resourceDebugEnabled(d) {
        success := err == nil && deleteLoopbackInterfaceRes.Success
        errMsg := ""
        if err != nil {
            errMsg = err.Error()
        } else if !deleteLoopbackInterfaceRes.Success {
            errMsg = deleteLoopbackInterfaceRes.ErrorMsg
        }

        var respData map[string]interface{}
        if err == nil {
            respData = deleteLoopbackInterfaceRes.GetData()
        }

        debugLogOperation(
            "loopback-interface",        // resource type
            "delete",                       // operation
            "delete-loopback-interface",         // API call name
            payload,                        // request payload
            respData,                       // response data (if any)
            success,
            errMsg,
        )
    }
    if err != nil {
        return fmt.Errorf("Failed to delete loopback-interface: %v", err)
    }
    if !deleteLoopbackInterfaceRes.Success {
        return fmt.Errorf(deleteLoopbackInterfaceRes.ErrorMsg)
    }

    d.SetId("")
    return nil
}

