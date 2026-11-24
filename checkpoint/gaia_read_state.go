package checkpoint

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// gaiaChangedValue keeps observed hashes and clock values out of unrelated
// updates. GetOk also rejects unknown computed values and empty string inputs.
func gaiaChangedValue(d *schema.ResourceData, name string) (interface{}, bool) {
	if !d.HasChange(name) {
		return nil, false
	}
	return d.GetOk(name)
}

// gaiaReadResource retains legacy write-action blocks and exposes their readable
// collections separately. An API's current collection is not an "add" request.
func gaiaReadResource(name string, r *schema.Resource) *schema.Resource {
	current := make(map[string]*schema.Schema)
	for key, spec := range r.Schema {
		if item := gaiaReadActionItem(spec); item != nil {
			var elem interface{} = &schema.Schema{Type: item.Type}
			if item.Type == schema.TypeList || item.Type == schema.TypeSet {
				elem = item.Elem
			}
			if object, ok := elem.(*schema.Resource); ok {
				fields := gaiaComputedFields(object.Schema)
				if name == "static-route" && key == "next_hop" && fields["priority"] != nil {
					fields["priority"].Type = schema.TypeString
				}
				elem = &schema.Resource{Schema: fields}
			}
			current["current_"+key] = &schema.Schema{
				Type: schema.TypeList, Computed: true,
				Description: "Current " + strings.ReplaceAll(key, "_", " ") + " returned by Gaia. The legacy block describes write actions.",
				Elem:        elem,
			}
		}
	}
	for key, spec := range current {
		r.Schema[key] = spec
	}
	if name == "static-route" && r.Schema["current_next_hop"] == nil {
		if spec := r.Schema["next_hop"]; spec != nil {
			if item, ok := spec.Elem.(*schema.Resource); ok && item.Schema["priority"] != nil && item.Schema["priority"].Type == schema.TypeInt {
				fields := gaiaComputedFields(item.Schema)
				fields["priority"].Type = schema.TypeString
				r.Schema["current_next_hop"] = &schema.Schema{Type: schema.TypeList, Computed: true, Elem: &schema.Resource{Schema: fields}, Description: "Current gateways and priorities, including Gaia's default priority sentinel."}
			}
		}
	}
	if name == "password-policy" {
		if spec := r.Schema["lock_settings"]; spec != nil {
			if block, ok := spec.Elem.(*schema.Resource); ok {
				for _, key := range []string{"password_expiration_days", "password_expiration_maximum_days_before_lock"} {
					if spec := block.Schema[key]; spec != nil && spec.Type == schema.TypeInt {
						block.Schema["current_"+key] = &schema.Schema{Type: schema.TypeString, Computed: true, Sensitive: spec.Sensitive, Description: "Current value, including Gaia's never sentinel."}
					}
				}
			}
		}
	}
	// Returned hashes/settings must not be planned for removal when omitted from
	// an import configuration. Explicit configuration continues to manage them.
	if name == "grub-password" {
		r.Schema["password_hash"].Computed = true
	}
	if name == "time-and-date" {
		r.Schema["time"].Computed = true
		r.Schema["date"].Computed = true
	}
	return r
}

func gaiaReadActionItem(spec *schema.Schema) *schema.Schema {
	if spec.Type != schema.TypeList {
		return nil
	}
	block, ok := spec.Elem.(*schema.Resource)
	if !ok {
		return nil
	}
	add, ok := block.Schema["add"]
	if !ok {
		return nil
	}
	for name := range block.Schema {
		if name != "add" && name != "remove" && name != "delete" && name != "set" {
			return nil
		}
	}
	return add
}

func gaiaComputedFields(fields map[string]*schema.Schema) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(fields))
	for name, spec := range fields {
		copy := &schema.Schema{Type: spec.Type, Computed: true, Sensitive: spec.Sensitive, Description: spec.Description}
		if nested, ok := spec.Elem.(*schema.Resource); ok {
			copy.Elem = &schema.Resource{Schema: gaiaComputedFields(nested.Schema)}
		} else {
			copy.Elem = spec.Elem
		}
		out[name] = copy
	}
	return out
}

// gaiaReadState converts complete API values before setting state. In particular
// it works on an empty import, checks d.Set errors, and never asserts API types.
func gaiaReadState(d *schema.ResourceData, fields map[string]*schema.Schema, response map[string]interface{}) error {
	data := response
	if raw, exists := response["bgp"]; exists && fields["policy_id"] != nil {
		policies, ok := raw.([]interface{})
		if !ok {
			return fmt.Errorf("invalid Gaia BGP policy collection type %T", raw)
		}
		data = nil
		for _, raw := range policies {
			policy, ok := raw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid Gaia BGP policy item type %T", raw)
			}
			id, exists := gaiaReadLookup(policy, "policy_id")
			if exists && fmt.Sprint(id) == fmt.Sprint(d.Get("policy_id")) {
				data = policy
				break
			}
		}
		if data == nil {
			d.SetId("")
			return nil
		}
	}
	for _, wrapper := range []string{"NFS-Version-4-Settings", "lightshot-partition"} {
		if nested, ok := response[wrapper].(map[string]interface{}); ok {
			data = nested
			break
		}
	}
	values, err := gaiaReadObject(data, fields, "", 0)
	if err != nil {
		return err
	}
	for name, value := range values {
		if err := d.Set(name, value); err != nil {
			return fmt.Errorf("cannot set Gaia state field %q: %w", name, err)
		}
	}
	return nil
}

func gaiaReadObject(data map[string]interface{}, fields map[string]*schema.Schema, path string, depth int) (map[string]interface{}, error) {
	if depth > 32 {
		return nil, fmt.Errorf("Gaia response nesting exceeds 32 levels at %s", path)
	}
	out := make(map[string]interface{})
	for name, spec := range fields {
		if name == "id" || name == "debug" || gaiaReadActionItem(spec) != nil {
			continue
		}
		apiName := strings.TrimPrefix(name, "current_")
		raw, exists := gaiaReadLookup(data, apiName)
		if !exists || raw == nil {
			continue
		}
		value, err := gaiaReadValue(raw, spec, path+name, depth+1)
		if err != nil {
			return nil, err
		}
		out[name] = value
	}
	return out, nil
}

func gaiaReadLookup(data map[string]interface{}, name string) (interface{}, bool) {
	if v, ok := data[strings.ReplaceAll(name, "_", "-")]; ok {
		return v, true
	}
	if v, ok := data[name]; ok {
		return v, true
	}
	for key, value := range data {
		if strings.ReplaceAll(strings.ToLower(key), "-", "_") == name {
			return value, true
		}
	}
	if name == "resource_id" || name == "policy_id" {
		v, ok := data["id"]
		return v, ok
	}
	return nil, false
}

func gaiaReadUnset(raw interface{}) bool {
	s, ok := raw.(string)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "not-configured", "not configured":
		return true
	}
	return false
}

func gaiaReadValue(raw interface{}, spec *schema.Schema, path string, depth int) (interface{}, error) {
	fail := func() (interface{}, error) {
		return nil, fmt.Errorf("invalid Gaia response type %T for field %q", raw, path)
	}
	switch spec.Type {
	case schema.TypeInt:
		if value, ok := raw.(string); ok && strings.EqualFold(value, "n/a") {
			return nil, nil // Gaia reports inapplicable thresholds without a numeric value.
		}
		if value, ok := raw.(string); ok && strings.EqualFold(value, "default") && strings.Contains(path, "next_hop") && strings.HasSuffix(path, ".priority") {
			return nil, nil // The computed priority preserves the literal default value.
		}
		if value, ok := raw.(string); ok && strings.EqualFold(value, "never") && (strings.HasSuffix(path, ".password_expiration_days") || strings.HasSuffix(path, ".password_expiration_maximum_days_before_lock")) {
			// These legacy write fields are integers; preserve the actual union
			// value in their computed current_* companion instead of inventing a number.
			return nil, nil
		}
		if gaiaReadUnset(raw) {
			return 0, nil
		}
		text := fmt.Sprint(raw)
		if value, ok := raw.(float64); ok {
			text = strconv.FormatFloat(value, 'f', -1, 64)
		}
		value, err := strconv.Atoi(text)
		if err != nil {
			return fail()
		}
		return value, nil
	case schema.TypeFloat:
		value, err := strconv.ParseFloat(fmt.Sprint(raw), 64)
		if err != nil {
			return fail()
		}
		return value, nil
	case schema.TypeBool:
		if value, ok := raw.(bool); ok {
			return value, nil
		}
		if value, ok := raw.(string); ok {
			switch strings.ToLower(value) {
			case "true", "yes", "on", "enabled":
				return true, nil
			case "false", "no", "off", "disabled", "not-configured", "not configured", "":
				return false, nil
			}
		}
		return fail()
	case schema.TypeString:
		if (strings.Contains(path, "ipv6") || strings.Contains(path, "ipv4")) && gaiaReadUnset(raw) {
			return "", nil
		}
		if list, ok := raw.([]interface{}); ok {
			parts := make([]string, len(list))
			for i, value := range list {
				s, ok := value.(string)
				if !ok {
					return fail()
				}
				parts[i] = s
			}
			return strings.Join(parts, ","), nil
		}
		switch raw.(type) {
		case string, bool, int, float64:
		default:
			return fail()
		}
		value := fmt.Sprint(raw)
		if path == "date" && len(value) == 10 && value[2] == '-' && value[5] == '-' {
			value = value[6:] + "-" + value[3:5] + "-" + value[:2]
		}
		if path == "timezone" {
			value = strings.ReplaceAll(value, " / ", "/")
		}
		if path == "command" {
			value = strings.ReplaceAll(value, `\ `, " ")
		}
		return value, nil
	case schema.TypeList, schema.TypeSet:
		var list []interface{}
		if gaiaReadUnset(raw) {
			return []interface{}{}, nil
		}
		if values, ok := raw.([]interface{}); ok {
			list = values
		} else {
			list = []interface{}{raw}
		}
		out := make([]interface{}, 0, len(list))
		for _, item := range list {
			switch elem := spec.Elem.(type) {
			case *schema.Resource:
				object, ok := item.(map[string]interface{})
				if !ok {
					return fail()
				}
				value, err := gaiaReadObject(object, elem.Schema, path+".", depth+1)
				if err != nil {
					return nil, err
				}
				out = append(out, value)
			case *schema.Schema:
				if object, ok := item.(map[string]interface{}); ok && elem.Type == schema.TypeString {
					if name, ok := object["name"]; ok {
						item = name
					}
				}
				value, err := gaiaReadValue(item, elem, path+"[]", depth+1)
				if err != nil {
					return nil, err
				}
				out = append(out, value)
			default:
				return fail()
			}
		}
		return out, nil
	case schema.TypeMap:
		if object, ok := raw.(map[string]interface{}); ok {
			return object, nil
		}
		return fail()
	}
	return fail()
}
