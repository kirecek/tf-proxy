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
	ProxyAddr        string
	OverrideFilename string
	TerraformBinary  string
	KeepOverrideFile bool
	TargetProviders  []string
}

type overrideTarget struct {
	resourceType string
	name         string
	alias        string
	hasAlias     bool
}

func parseTarget(input string) (*overrideTarget, error) {
	parts := strings.Split(input, "/")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, errors.New("invalid format: expected '<type>/<name>/<alias>' or '<type>/<name>'")
	}

	typeValue := parts[0]
	if typeValue != "backend" && typeValue != "provider" {
		return nil, errors.New("invalid type: must be 'backend' or 'provider'")
	}

	res := &overrideTarget{
		resourceType: typeValue,
		name:         parts[1],
	}

	if len(parts) == 3 {
		res.alias = parts[2]
		res.hasAlias = true
	}

	return res, nil
}

func (t *Terraform) cleanup() {
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
		fmt.Printf("Error reading terraform files: %v\n", err)
		return err
	}

	for _, unparsedTarget := range t.TargetProviders {
		target, err := parseTarget(unparsedTarget)
		if err != nil {
			return fmt.Errorf("invalid format for target %s", unparsedTarget)
		}

		for _, block := range t.generateProviderBlocks(files, target) {
			body.AppendBlock(block)
		}
	}

	return os.WriteFile(t.OverrideFilename, override.Bytes(), 0644)
}

func (t *Terraform) generateProviderBlocks(config []*hclwrite.File, target *overrideTarget) []*hclwrite.Block {
	res := make([]*hclwrite.Block, 0)

	for _, f := range config {
		for _, block := range f.Body().Blocks() {
			labels := block.Labels()
			if block.Type() == target.resourceType && len(labels) > 0 && labels[0] == target.name {

				if target.hasAlias {
					resAlias := string(block.Body().GetAttribute("alias").Expr().BuildTokens(nil).Bytes())
					resAlias = strings.Trim(strings.TrimSpace(resAlias), "\"")

					if resAlias != target.alias {
						continue
					}

				}

				block.Body().AppendNewline()
				block.Body().SetAttributeValue("https_proxy", cty.StringVal(t.ProxyAddr))

				res = append(res, block)
			}

			if block.Type() == "terraform" {
				for _, providerBlocks := range block.Body().Blocks() {
					pl := providerBlocks.Labels()
					if providerBlocks.Type() == target.resourceType && len(pl) > 0 && pl[0] == target.name {
						providerBlocks.Body().AppendNewline()
						providerBlocks.Body().SetAttributeValue("https_proxy", cty.StringVal(t.ProxyAddr))
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
		fmt.Printf("Terraform binary not found: %v\n", err)
		return err
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
