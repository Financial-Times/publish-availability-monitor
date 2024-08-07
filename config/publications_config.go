package config

type PublicationsConfig struct {
	EnabledPublications []Publication `json:"enabledPublications"`
}

type Publication struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

func (cfg *PublicationsConfig) GetXReadPolicies() []string {
	prefixedUUIDs := make([]string, len(cfg.EnabledPublications))
	for i, publication := range cfg.EnabledPublications {
		prefixedUUIDs[i] = "PBLC_READ_" + publication.UUID
	}
	return prefixedUUIDs
}
