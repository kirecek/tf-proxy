package proxy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type Terraform struct {
	Config           *Config
	TerraformBinary  string
	OverrideFilename string
	KeepOverrideFile bool
}

func (t *Terraform) cleanup() {
	if _, err := os.Stat(t.OverrideFilename); errors.Is(err, os.ErrNotExist) {
		return
	}
	err := os.Remove(t.OverrideFilename)
	if err != nil {
		fmt.Printf("Could not remove proxy override file: %v\n", err)
	}
}

func (t *Terraform) createOverrideFile() error {
	override := hclwrite.NewEmptyFile()
	body := override.Body()

	files, err := t.determineTerraformFiles()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.New("could not find any terraform files matchin '*.tf' pattern")
	}

	for _, target := range t.Config.GetProviders() {
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid target format: %s", target)
		}

		providerType := parts[0]
		providerName := parts[1]
		proxyAddr := t.Config.GetProxyForProvider(providerName)

		for _, block := range t.generateProviderBlocks(files, providerType, providerName) {
			block.Body().AppendNewline()
			block.Body().SetAttributeValue("https_proxy", cty.StringVal(proxyAddr))
			body.AppendBlock(block)
		}
	}

	return os.WriteFile(t.OverrideFilename, override.Bytes(), 0644)
}

func (t *Terraform) generateProviderBlocks(config []*hclwrite.File, providerType, providerName string) []*hclwrite.Block {
	res := make([]*hclwrite.Block, 0)

	for _, f := range config {
		for _, block := range f.Body().Blocks() {
			labels := block.Labels()
			if block.Type() == providerType && len(labels) > 0 && labels[0] == providerName {
				res = append(res, block)
			}

			if block.Type() == "terraform" {
				for _, providerBlocks := range block.Body().Blocks() {
					pl := providerBlocks.Labels()
					if providerBlocks.Type() == providerType && len(pl) > 0 && pl[0] == providerName {
						block.Body().AppendNewline()
						res = append(res, block)
					}
				}
			}
		}
	}

	return res
}

func (t *Terraform) Run(args []string) error {
	_, err := exec.LookPath(t.TerraformBinary)
	if err != nil {
		return fmt.Errorf("Terraform binary not found: %v", err)
	}

	// make sure to delete override file
	if !t.KeepOverrideFile {
		defer t.cleanup()
	}

	err = t.createOverrideFile()
	if err != nil {
		return err
	}

	cmd := exec.Command(t.TerraformBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Run the command and check for errors
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (t *Terraform) determineTerraformFiles() ([]*hclwrite.File, error) {
	res := make([]*hclwrite.File, 0)

	files, err := filepath.Glob("*.tf")
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_override.tf") {
			continue
		}

		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}

		file, diags := hclwrite.ParseConfig(raw, f, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, errors.New("could not parse hcl file")
		}
		res = append(res, file)
	}

	return res, nil
}
