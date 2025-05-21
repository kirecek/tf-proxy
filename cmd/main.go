package main

import (
	"fmt"
	"os"

	"github.com/kirecek/tf-proxy/internal/pkg/proxy"
)

func main() {
	for _, n := range []string{"HTTPS_PROXY", "HTTP_PROXY"} {
		if _, exists := os.LookupEnv("HTTPS_PROXY"); exists {
			fmt.Printf("'%s' is already set. Unset it before using this wrapper\n", n)
			os.Exit(1)
		}
	}

	// Load configuration
	config, err := proxy.LoadConfig(os.Getenv("TF_PROXY_CONFIG"))
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Get default proxy from environment if not set in config
	if config.DefaultProxy == "" {
		config.DefaultProxy = os.Getenv("TF_PROXY_HOST")
		if config.DefaultProxy == "" {
			fmt.Println("No default proxy configured. Set TF_PROXY_HOST or configure default_proxy in config file")
			os.Exit(1)
		}
	}

	tf := proxy.Terraform{
		Config:           config,
		TerraformBinary:  "terraform",
		OverrideFilename: "terraform_proxy_override.tf",
		KeepOverrideFile: false,
	}

	err = tf.Run(os.Args[1:])
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
