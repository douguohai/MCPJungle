package config

import (
	"errors"
	"testing"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"gorm.io/gorm"
)

func TestNewServerConfigService(t *testing.T) {
	db := testhelpers.RequireTestDB(t)

	svc := NewServerConfigService(db)
	testhelpers.AssertNotNil(t, svc)
	if svc.db != db {
		t.Errorf("Expected db to be %v, got %v", db, svc.db)
	}
}

func TestInitWithRollsBackConfigWhenSetupFails(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	svc := NewServerConfigService(setup.DB)
	wantErr := errors.New("create administrator")
	created, err := svc.InitWith(model.ModeEnterprise, func(_ *gorm.DB) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("InitWith() error = %v, want %v", err, wantErr)
	}
	if created {
		t.Fatal("InitWith() reported a committed configuration after rollback")
	}

	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if config.Initialized {
		t.Fatal("enterprise configuration remained initialized after setup rollback")
	}
}

func TestGetConfigEmptyDatabase(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	svc := NewServerConfigService(setup.DB)

	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)

	// Should return default uninitialized config
	if config.Initialized {
		t.Error("Expected config to be uninitialized when database is empty")
	}
}

func TestGetConfigWithExistingConfig(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	// Create a test config using the helper
	setup.CreateTestServerConfig(model.ModeDev, true)

	svc := NewServerConfigService(setup.DB)

	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)

	// Should return the existing config
	if !config.Initialized {
		t.Error("Expected config to be initialized")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to be %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitFirstTime(t *testing.T) {
	setup := testhelpers.SetupServerConfigTest(t)
	defer setup.Cleanup()

	svc := NewServerConfigService(setup.DB)

	// Initially no config should exist
	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if config.Initialized {
		t.Error("Expected config to be uninitialized initially")
	}

	// Initialize the config
	created, err := svc.Init(model.ModeDev)
	testhelpers.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created")
	}

	// Verify config was created
	config, err = svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if !config.Initialized {
		t.Error("Expected config to be initialized after Init")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to be %v, got %v", model.ModeDev, config.Mode)
	}
}

func TestInitIdempotent(t *testing.T) {
	db := testhelpers.RequireTestDB(t)

	// Auto-migrate the ServerConfig model
	err := db.AutoMigrate(&model.ServerConfig{})
	testhelpers.AssertNoError(t, err)

	svc := NewServerConfigService(db)

	// Initialize the config first time
	created, err := svc.Init(model.ModeDev)
	testhelpers.AssertNoError(t, err)
	if !created {
		t.Error("Expected config to be created first time")
	}

	// Try to initialize again
	created, err = svc.Init(model.ModeDev)
	testhelpers.AssertNoError(t, err)
	if created {
		t.Error("Expected config not to be created second time")
	}

	// Verify config is still valid
	config, err := svc.GetConfig()
	testhelpers.AssertNoError(t, err)
	if !config.Initialized {
		t.Error("Expected config to remain initialized")
	}
	if config.Mode != model.ModeDev {
		t.Errorf("Expected mode to remain %v, got %v", model.ModeDev, config.Mode)
	}
}
