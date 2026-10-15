package service

import (
	_ "embed"
	"encoding/json"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/common"
	"github.com/superaddmin/SuperXray-gui/v2/util/xraycompat"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

// XraySettingService provides business logic for Xray configuration management.
// It handles validation and storage of Xray template configurations.
type XraySettingService struct {
	SettingService
	validateConfig func(*xray.Config) error
}

func (s *XraySettingService) SaveXraySetting(newXraySettings string) error {
	// The frontend round-trips the whole getXraySetting response back
	// through the textarea, so if it has ever received a wrapped
	// payload (see UnwrapXrayTemplateConfig) it sends that same wrapper
	// back here. Strip it before validation/storage, otherwise we save
	// garbage the next read can't recover from without this same call.
	newXraySettings = UnwrapXrayTemplateConfig(newXraySettings)
	normalized, _, err := xraycompat.NormalizeConfig(newXraySettings)
	if err != nil {
		return err
	}
	newXraySettings = normalized
	if err := s.CheckXrayConfig(newXraySettings); err != nil {
		return err
	}
	return s.SettingService.saveSetting("xrayTemplateConfig", newXraySettings)
}

func (s *XraySettingService) CheckXrayConfig(XrayTemplateConfig string) error {
	// Build with a read-only template overlay: routes may refer to the panel's
	// enabled inbounds, and validation must not persist a candidate or traffic.
	preview := XrayService{settingService: *NewSettingService(templateSettings{
		SettingRepository: s.settings(), template: XrayTemplateConfig,
	})}
	xrayConfig, err := preview.getXrayConfig(false)
	if err != nil {
		return common.NewError("xray template config invalid:", err)
	}
	validate := s.validateConfig
	if validate == nil {
		validate = xray.ValidateConfig
	}
	return validate(xrayConfig)
}

type templateSettings struct {
	database.SettingRepository
	template string
}

func (s templateSettings) Get(key string) (*model.Setting, error) {
	if key == "xrayTemplateConfig" {
		return &model.Setting{Key: key, Value: s.template}, nil
	}
	return s.SettingRepository.Get(key)
}

// UnwrapXrayTemplateConfig returns the raw xray config JSON from `raw`,
// peeling off any number of `{ "inboundTags": ..., "outboundTestUrl": ...,
// "xraySetting": <real config> }` response-shaped wrappers that may have
// ended up in the database.
//
// How it got there: getXraySetting used to embed the raw DB value as
// `xraySetting` in its response without checking whether the stored
// value was already that exact response shape. If the frontend then
// saved it verbatim (the textarea is a round-trip of the JSON it was
// handed), the wrapper got persisted — and each subsequent save nested
// another layer, producing the blank Xray Settings page reported in
// issue #4059.
//
// If `raw` does not look like a wrapper, it is returned unchanged.
func UnwrapXrayTemplateConfig(raw string) string {
	const maxDepth = 8 // defensive cap against pathological multi-nest values
	for i := 0; i < maxDepth; i++ {
		var top map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &top); err != nil {
			return raw
		}
		inner, ok := top["xraySetting"]
		if !ok {
			return raw
		}
		// Real xray configs never contain a top-level "xraySetting" key,
		// but they do contain things like "inbounds"/"outbounds"/"api".
		// If any of those are present, we're already at the real config
		// and the "xraySetting" field is either user data or coincidence
		// — don't touch it.
		for _, k := range []string{"inbounds", "outbounds", "routing", "api", "dns", "log", "policy", "stats"} {
			if _, hit := top[k]; hit {
				return raw
			}
		}
		// Peel off one layer.
		unwrapped := string(inner)
		// `xraySetting` may be stored either as a JSON object or as a
		// JSON-encoded string of an object. Handle both.
		var asStr string
		if err := json.Unmarshal(inner, &asStr); err == nil {
			unwrapped = asStr
		}
		raw = unwrapped
	}
	return raw
}
