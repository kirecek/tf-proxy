package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kirecek/tf-proxy/internal/pkg/proxy"
)

func main() {
	for _, n := range []string{"HTTPS_PROXY", "HTTP_PROXY"} {
		if _, exists := os.LookupEnv("HTTPS_PROXY"); exists {
			fmt.Printf("'%s' is already set. Unset it before using this wrapper\n", n)
			os.Exit(1)
		}
	}

	proxyAddr := os.Getenv("TF_PROXY_HOST")
	if proxyAddr == "" {
		fmt.Println("'TF_PROXY_HOST' not set")
		os.Exit(1)
	}

	providers := []string{"backend/s3", "provider/aws"}
	if v, ok := os.LookupEnv("TF_PROXY_PROVIDERS"); !ok {
		providers = strings.Split(v, ",")
	}

	tf := proxy.Terraform{
		TerraformBinary:  "terraform",
		OverrideFilename: "terraform_proxy_providers_override.tf",
		ProxyAddr:        proxyAddr,
		KeepOverrideFile: false,
		TargetProviders:  providers,
	}

	err := tf.Run(os.Args[1:])
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(0)
	}
}
