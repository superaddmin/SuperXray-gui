package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"github.com/xtls/xray-core/infra/conf"
)

func TestSaveXrayTemplateValidatesBeforePersisting(t *testing.T) {
	setupSettingServiceTestDB(t)
	s := &XraySettingService{}
	original := `{"outbounds":[{"protocol":"freedom","tag":"original"}]}`
	if err := s.saveSetting("xrayTemplateConfig", original); err != nil {
		t.Fatal(err)
	}
	s.validateConfig = func(*xray.Config) error { return errors.New("candidate rejected") }
	if err := s.SaveXraySetting(`{"outbounds":[{"protocol":"freedom","tag":"candidate"}]}`); err == nil {
		t.Fatal("expected validation error")
	}
	stored, err := s.GetXrayConfigTemplate()
	if err != nil || stored != original {
		t.Fatal("failed validation changed the persisted template")
	}
	s.validateConfig = func(cfg *xray.Config) error {
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		if !strings.Contains(string(data), "9007199254740993") {
			t.Fatal("unknown numeric field lost precision")
		}
		var core conf.Config
		if err := json.Unmarshal(data, &core); err != nil {
			return err
		}
		_, err = core.Build()
		return err
	}
	legacy := `{"vendor":9007199254740993,"outbounds":[{"protocol":"freedom","settings":{"domainStrategy":"UseIPv4"}}]}`
	if err := s.SaveXraySetting(legacy); err != nil {
		t.Fatal(err)
	}
	stored, err = s.GetXrayConfigTemplate()
	if err != nil || !strings.Contains(stored, `"sockopt"`) || !strings.Contains(stored, "9007199254740993") {
		t.Fatal("normalized template was not persisted losslessly")
	}
}
