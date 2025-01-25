# (WIP) tf-proxy

Experimental project that wraps terraform binary with purpose to proxy only specific providers.

## Usage

Set `TERRAFORM_HTTPS_PROXY` environment value and simply run `tf-proxy` binary instead of terrafom i.e. `tf-proxy init`.

:warning: Configuration is not yet exposed and at the moment it's embeded only internaly set for AWS providers.

```
		TargetProviders:  []string{"backend/s3", "provider/aws"},
```

## How it works?

```mermaid
sequenceDiagram
    participant user
    participant tf-proxy
    participant terraform
    participant filesystem

    user ->> tf-proxy: Runs "tf-proxy"
    filesystem  ->> tf-proxy: Read all provider resources
    tf-proxy ->> filesystem: Create override for aws providers
    filesystem -->> tf-proxy: Override file created
    tf-proxy ->> terraform: Call Terraform binary with given  args
    terraform -->> tf-proxy: Terraform command completes
    tf-proxy ->> filesystem: Remove _override file
    filesystem -->> tf-proxy: Override file removed
    tf-proxy -->> user: Workflow completed
```

