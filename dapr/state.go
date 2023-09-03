package dapr

import (
	"github.com/dapr/dapr/pkg/components"
	host_state "github.com/taction/wit-dapr/pkg/imports/host-state"

	"log"
)

func LoadComponents(paths ...string) ([]host_state.Component, error) {
	loader := components.NewLocalComponents(paths...)
	log.Println("Loading components…")
	comps, err := loader.LoadComponents()
	if err != nil {
		panic(err)
	}
	var compList []host_state.Component
	for _, comp := range comps {
		// copy comp to host_state.Component
		md := make([]host_state.MetadataItem, len(comp.Spec.Metadata))
		for i, m := range comp.Spec.Metadata {
			md[i] = host_state.MetadataItem{
				Name:  m.Name,
				Value: m.Value.String(),
				SecretKeyRef: host_state.SecretKeyRef{
					Name: m.SecretKeyRef.Name,
					Key:  m.SecretKeyRef.Key,
				},
			}
		}
		compList = append(compList, host_state.Component{
			TypeMeta:   comp.TypeMeta,
			ObjectMeta: comp.ObjectMeta,
			Spec: host_state.ComponentSpec{
				Type:     comp.Spec.Type,
				Version:  comp.Spec.Version,
				Metadata: md,
			},
		})
	}
	return compList, nil
}
