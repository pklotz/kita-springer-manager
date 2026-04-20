package seeds

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/pak/kita-springer-manager/internal/models"
)

//go:embed kitas_stadt_bern.json
var kitasStadtBern []byte

//go:embed kitas_stiftung_bern.json
var kitasStiftungBern []byte

var registry = map[string][]byte{
	"stadt_bern":    kitasStadtBern,
	"stiftung_bern": kitasStiftungBern,
}

type kitaSeed struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Phone    string   `json:"phone"`
	Email    string   `json:"email"`
	StopName string   `json:"stop_name"`
	Groups   []string `json:"groups"`
}

func Load(key string) ([]models.Kita, error) {
	data, ok := registry[key]
	if !ok {
		return nil, fmt.Errorf("unknown seed %q — valid: stadt_bern, stiftung_bern", key)
	}
	var seeds []kitaSeed
	if err := json.Unmarshal(data, &seeds); err != nil {
		return nil, err
	}
	kitas := make([]models.Kita, len(seeds))
	for i, s := range seeds {
		kitas[i] = models.Kita{
			Name:     s.Name,
			Address:  s.Address,
			Phone:    s.Phone,
			Email:    s.Email,
			StopName: s.StopName,
			Groups:   s.Groups,
		}
		if kitas[i].Groups == nil {
			kitas[i].Groups = []string{}
		}
	}
	return kitas, nil
}
