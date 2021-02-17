package main

import (
	"testing"
)

// TestUnitProcessNetAddr probes the 'processNetAddr' function.
func TestUnitProcessNetAddr(t *testing.T) {
	testVals := struct {
		inputs          []string
		expectedOutputs []string
	}{
		inputs:          []string{"9980", ":9980", "localhost:9980", "test.com:9980", "192.168.14.92:9980"},
		expectedOutputs: []string{":9980", ":9980", "localhost:9980", "test.com:9980", "192.168.14.92:9980"},
	}
	for i, input := range testVals.inputs {
		output := processNetAddr(input)
		if output != testVals.expectedOutputs[i] {
			t.Error("unexpected result", i)
		}
	}
}

// TestUnitProcessModules tests that processModules correctly processes modules
// passed to the -M / --modules flag.
func TestUnitProcessModules(t *testing.T) {
	// Test valid modules.
	testVals := []struct {
		in  string
		out string
	}{
		{"cghmrtwe", "cghmrtwe"},
		{"CGHMRTWE", "cghmrtwe"},
		{"c", "c"},
		{"g", "g"},
		{"h", "h"},
		{"m", "m"},
		{"r", "r"},
		{"t", "t"},
		{"w", "w"},
		{"e", "e"},
		{"C", "c"},
		{"G", "g"},
		{"H", "h"},
		{"M", "m"},
		{"R", "r"},
		{"T", "t"},
		{"W", "w"},
		{"E", "e"},
	}
	for _, testVal := range testVals {
		out, err := processModules(testVal.in)
		if err != nil {
			t.Error("processModules failed with error:", err)
		}
		if out != testVal.out {
			t.Errorf("processModules returned incorrect modules: expected %s, got %s\n", testVal.out, out)
		}
	}

	// Test invalid modules.
	invalidModules := []string{"abdfijklnopqsuvxyz", "cghmrtwez", "cz", "z", "cc", "ccz", "ccm", "cmm", "ccmm"}
	for _, invalidModule := range invalidModules {
		_, err := processModules(invalidModule)
		if err == nil {
			t.Error("processModules didn't error on invalid module:", invalidModule)
		}
	}
}

// TestUnitProcessConfig probes the 'processConfig' function.
func TestUnitProcessConfig(t *testing.T) {
	// Test valid configs.
	testVals := struct {
		inputs          [][]string
		expectedOutputs [][]string
	}{
		inputs: [][]string{
			{"localhost:9980", "localhost:9981", "localhost:9982", "cghmrtwe"},
			{"localhost:9980", "localhost:9981", "localhost:9982", "CGHMRTWE"},
		},
		expectedOutputs: [][]string{
			{"localhost:9980", "localhost:9981", "localhost:9982", "cghmrtwe"},
			{"localhost:9980", "localhost:9981", "localhost:9982", "cghmrtwe"},
		},
	}
	var config Config
	for i := range testVals.inputs {
		config.TurtleDexd.APIaddr = testVals.inputs[i][0]
		config.TurtleDexd.RPCaddr = testVals.inputs[i][1]
		config.TurtleDexd.HostAddr = testVals.inputs[i][2]
		config, err := processConfig(config)
		if err != nil {
			t.Error("processConfig failed with error:", err)
		}
		if config.TurtleDexd.APIaddr != testVals.expectedOutputs[i][0] {
			t.Error("processing failure at check", i, 0)
		}
		if config.TurtleDexd.RPCaddr != testVals.expectedOutputs[i][1] {
			t.Error("processing failure at check", i, 1)
		}
		if config.TurtleDexd.HostAddr != testVals.expectedOutputs[i][2] {
			t.Error("processing failure at check", i, 2)
		}
	}

	// Test invalid configs.
	invalidModule := "z"
	config.TurtleDexd.Modules = invalidModule
	_, err := processConfig(config)
	if err == nil {
		t.Error("processModules didn't error on invalid module:", invalidModule)
	}
}

// TestLoadAPIPassword tests the 'loadAPIPassword' function.
func TestLoadAPIPassword(t *testing.T) {
	// If config.TurtleDexd.AuthenticateAPI is false, no password should be set
	var config Config

	config, err := loadAPIPassword(config)
	if err != nil {
		t.Fatal(err)
	} else if config.APIPassword != "" {
		t.Fatal("loadAPIPassword should not set a password if config.TurtleDexd.AuthenticateAPI is false")
	}
	config.TurtleDexd.AuthenticateAPI = true
	// On first invocation, loadAPIPassword should generate a new random
	// password
	config2, err := loadAPIPassword(config)
	if err != nil {
		t.Fatal(err)
	} else if config2.APIPassword == "" {
		t.Fatal("loadAPIPassword should have generated a random password")
	}
	// On subsequent invocations, loadAPIPassword should use the
	// previously-generated password
	config3, err := loadAPIPassword(config)
	if err != nil {
		t.Fatal(err)
	} else if config3.APIPassword != config2.APIPassword {
		t.Fatal("loadAPIPassword should have used previously-generated password")
	}
}

// TestVerifyAPISecurity checks that the verifyAPISecurity function is
// correctly banning the use of a non-loopback address without the
// --disable-security flag, and that the --disable-security flag cannot be used
// without an api password.
func TestVerifyAPISecurity(t *testing.T) {
	// Check that the loopback address is accepted when security is enabled.
	var securityOnLoopback Config
	securityOnLoopback.TurtleDexd.APIaddr = "127.0.0.1:9980"
	err := verifyAPISecurity(securityOnLoopback)
	if err != nil {
		t.Error("loopback + securityOn was rejected")
	}

	// Check that the blank address is rejected when security is enabled.
	var securityOnBlank Config
	securityOnBlank.TurtleDexd.APIaddr = ":9980"
	err = verifyAPISecurity(securityOnBlank)
	if err == nil {
		t.Error("blank + securityOn was accepted")
	}

	// Check that a public hostname is rejected when security is enabled.
	var securityOnPublic Config
	securityOnPublic.TurtleDexd.APIaddr = "turtledex.io:9980"
	err = verifyAPISecurity(securityOnPublic)
	if err == nil {
		t.Error("public + securityOn was accepted")
	}

	// Check that a public hostname is rejected when security is disabled and
	// there is no api password.
	var securityOffPublic Config
	securityOffPublic.TurtleDexd.APIaddr = "turtledex.io:9980"
	securityOffPublic.TurtleDexd.AllowAPIBind = true
	err = verifyAPISecurity(securityOffPublic)
	if err == nil {
		t.Error("public + securityOff was accepted without authentication")
	}

	// Check that a public hostname is accepted when security is disabled and
	// there is an api password.
	var securityOffPublicAuthenticated Config
	securityOffPublicAuthenticated.TurtleDexd.APIaddr = "turtledex.io:9980"
	securityOffPublicAuthenticated.TurtleDexd.AllowAPIBind = true
	securityOffPublicAuthenticated.TurtleDexd.AuthenticateAPI = true
	err = verifyAPISecurity(securityOffPublicAuthenticated)
	if err != nil {
		t.Error("public + securityOff with authentication was rejected:", err)
	}
}