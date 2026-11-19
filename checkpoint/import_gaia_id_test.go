package checkpoint

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"regexp"
	"testing"
)

func TestGaiaImportUsesCreatedIDFormat(t *testing.T) {
	cases := []struct {
		name, input, prefix string
		resource            *schema.Resource
		fields              map[string]interface{}
	}{
		{"single", "alice", "user", resourceGaiaUser(), map[string]interface{}{"name": "alice"}},
		{"composite", `{"peer":"192.0.2.1","remote_as":"65001"}`, "bgp-external-peer", resourceGaiaBgpExternalPeer(), map[string]interface{}{"peer": "192.0.2.1", "remote_as": "65001"}},
		{"singleton", "settings", "hostname", resourceGaiaHostname(), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, tc.resource.Schema, nil)
			d.SetId(tc.input)
			states, err := tc.resource.Importer.State(d, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(states) != 1 || !regexp.MustCompile("^"+tc.prefix+"-[a-zA-Z0-9]{10}$").MatchString(states[0].Id()) {
				t.Fatalf("unexpected state ID: %q", d.Id())
			}
			for key, want := range tc.fields {
				if got := d.Get(key); got != want {
					t.Errorf("%s = %v, want %v", key, got, want)
				}
			}
		})
	}
}
