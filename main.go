package main

import (

	// "context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/template"

	// "github.com/docker/docker/api/types"
	// "github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	// "github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-plugins-helpers/secrets"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

const (
	// Can be set to "vault_token" to return a vault token
	typeLabel      = "docker.plugin.secretprovider.vault.type"
	vaultTokenType = "vault_token"
	// Can be set to "true" to wrap the contents of the secret
	wrapLabel = "docker.plugin.secretprovider.vault.wrap"

	// Read secret from this path
	pathLabel = "docker.plugin.secretprovider.vault.path"
	// Read this field from the secret (defaults to "value")
	fieldLabel = "docker.plugin.secretprovider.vault.field"
	// If using v2 key/value backend, use this version of the secret
	versionLabel = "docker.plugin.secretprovider.vault.version"
	// Return JSON encoded map of secret if set to "true"
	formatLabel = "docker.plugin.secretprovider.vault.format"

	// socket address
	socketAddress = "/run/docker/plugins/vault.sock"
)

var (
	log                      = logrus.New()
	policyTemplateExpression string
	policyTemplate           *template.Template
	// secretZero               string
)

type vaultSecretsDriver struct {
	dockerClient *client.Client
	vaultClient  *vaultapi.Client
}

func (d vaultSecretsDriver) Get(req secrets.Request) secrets.Response {
	errorResponse := func(s string, err error) secrets.Response {
		log.Errorf("Error getting secret %q: %s: %v", req.SecretName, s, err)
		return secrets.Response{
			Value: []byte("-"),
			Err:   fmt.Sprintf("%s: %v", s, err),
		}
	}
	valueResponse := func(s string) secrets.Response {
		return secrets.Response{
			Value:      []byte(s),
			DoNotReuse: true,
		}
	}

	// First use secret zero client to create a service token
	// var policies []string
	// if policyTemplate == nil {
	// 	policies = []string{req.ServiceName}
	// } else {
	// 	policiesBuffer := bytes.NewBuffer(nil)
	// 	tmpl, err := policyTemplate.Clone()
	// 	if err != nil {
	// 		log.Fatalf("Error cloning template: %v", err)
	// 	}
	// 	if err := tmpl.Funcs(template.FuncMap{
	// 		"ServiceLabel": func(name string) (string, error) {
	// 			value, exists := req.ServiceLabels[name]
	// 			if !exists {
	// 				return "", fmt.Errorf("No such service label: %q", name)
	// 			}
	// 			return value, nil
	// 		},
	// 	}).Execute(policiesBuffer, req); err != nil {
	// 		log.Fatalf("Error executing policy template: %v", err)
	// 	}
	// 	policies = strings.Split(policiesBuffer.String(), ",")
	// 	if len(policies) == 0 {
	// 		log.Fatalf("Empty policies list after executing template %q", policyTemplateExpression)
	// 	}
	// }
	// serviceToken, err := d.vaultClient.Auth().Token().Create(&vaultapi.TokenCreateRequest{
	// 	Policies: policies,
	// })
	// if err != nil {
	// 	policiesString := strings.Join(policies, ",")
	// 	return errorResponse(fmt.Sprintf("Error creating service token with policies like %q", policiesString), err)
	// }

	// Create a Vault client limited to the service token
	// var serviceVaultClient *vaultapi.Client
	// vaultConfig := vaultapi.DefaultConfig()
	// if c, err := vaultapi.NewClient(vaultConfig); err != nil {
	// 	log.Fatalf("Error creating Vault client: %v", err)
	// } else {
	// 	c.SetToken(serviceToken.Auth.ClientToken)
	// 	serviceVaultClient = c
	// 	defer serviceVaultClient.Auth().Token().RevokeSelf(serviceToken.Auth.ClientToken)
	// }
	// serviceVaultClient.SetToken(serviceToken.Auth.ClientToken)

	// Tips: as we use global token, so no need to create policy and token for each services.
	serviceVaultClient := d.vaultClient

	// Inspect the secret to read its labels
	var vaultWrapValue bool
	if v, exists := req.SecretLabels[wrapLabel]; exists {
		if v, err := strconv.ParseBool(v); err == nil {
			vaultWrapValue = v
		} else {
			return errorResponse(fmt.Sprintf("Error parsing boolean value of label %q", wrapLabel), err)
		}
	}

	format := "plain"
	if value, exists := req.SecretLabels[formatLabel]; exists {
		format = value
	}

	switch req.SecretLabels[typeLabel] {
	case vaultTokenType:
		// Create a token
		// TODO: Set reasonable default values, and allow configuring them through secret labels
		secret, err := serviceVaultClient.Auth().Token().Create(&vaultapi.TokenCreateRequest{
			Lease: "1h",
			// Policies: policies,
			Metadata: map[string]string{
				"created_by": os.Args[0],
				// TODO: Add any other interesting metadata
			},
		})
		if err != nil {
			return errorResponse("Error creating token in Vault", err)
		}

		switch format {
		case "meta+json":
			resultBytes, err := json.Marshal(secret)
			if err != nil {
				return errorResponse("Error marshalling secret", err)
			}
			return valueResponse(string(resultBytes))
		case "json":
			resultBytes, err := json.Marshal(secret.Auth.ClientToken)
			if err != nil {
				return errorResponse("Error marshalling secret", err)
			}
			return valueResponse(string(resultBytes))
		case "plain":
			return valueResponse(secret.Auth.ClientToken)
		default:
			return errorResponse("Unexpected format", errors.New(format))
		}
	default:
		var secret *vaultapi.Secret
		// Read from KV secrets mount
		field := ""
		if fieldName, exists := req.SecretLabels[fieldLabel]; exists {
			field = fieldName
		}
		path := fmt.Sprintf("secret/data/%s", req.SecretName)
		if v, exists := req.SecretLabels[pathLabel]; exists {
			path = v
		}
		params := make(url.Values)
		if v, exists := req.SecretLabels[versionLabel]; exists {
			params.Set("version", v)
		}
		secret, err := serviceVaultClient.Logical().ReadWithData(path, params)
		if err != nil {
			return errorResponse(fmt.Sprintf("Error getting kv secret from Vault at path %q", path), err)
		}
		if secret == nil || secret.Data == nil {
			return errorResponse(fmt.Sprintf("Data is nil at path %q (secret: %#v)", path, secret), err)
		}

		data := secret.Data["data"]
		if dataMap, ok := data.(map[string]interface{}); ok {
			if !vaultWrapValue {
				switch format {
				case "json", "meta+json":
					var resultBytes []byte
					if format == "meta+json" {
						resultBytes, err = json.Marshal(secret)
						if err != nil {
							return errorResponse("Error marshalling secret", err)
						}
					} else if len(field) == 0 {
						resultBytes, err = json.Marshal(dataMap)
						if err != nil {
							return errorResponse("Error marshalling secret data map", err)
						}
					} else {
						resultBytes, err = json.Marshal(dataMap[field])
						if err != nil {
							return errorResponse(fmt.Sprintf("Error marshalling secret data field %q", field), err)
						}
					}
					return valueResponse(fmt.Sprintf("%v", string(resultBytes)))
				case "plain":
					if len(field) == 0 {
						field = "value"
					}
					return valueResponse(fmt.Sprintf("%v", dataMap[field]))
				default:
					return errorResponse("Unexpected format", errors.New(format))
				}
			}
			// Wrap data map
			wrappedSecret, err := serviceVaultClient.Logical().Write("sys/wrapping/wrap", dataMap)
			if err != nil {
				return errorResponse("Error wrapping secret data", err)
			}
			return valueResponse(wrappedSecret.WrapInfo.Token)
		}
		return errorResponse("Invalid data map", err)
	}
}

// Read "secret zero" from the file system of a helper service task container, then serve the plugin.
func main() {
	// Create Docker client
	var httpClient *http.Client
	dockerAPIVersion := os.Getenv("DOCKER_API_VERSION")
	cli, err := client.NewClient("unix:///var/run/docker.sock", dockerAPIVersion, httpClient, nil)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// // Read plugin configuration from environment
	// vaultHelperServiceName := os.Getenv("vault-token-service")
	// secretZeroName := os.Getenv("vault-token-secret")

	// // Inspect the helper service
	// service, _, err := cli.ServiceInspectWithRaw(context.Background(), vaultHelperServiceName, types.ServiceInspectOptions{})
	// if err != nil {
	//     log.Fatalf("Error inspecting helper service %q: %v", vaultHelperServiceName, err)
	// }

	// // Look up hostname to filter tasks
	// hostname, err := os.Hostname()
	// if err != nil {
	//     log.Fatalf("Error getting hostname: %v", err)
	// }

	// // Find a task on this node, as otherwise we will not be able to exec inside its container
	// args := filters.NewArgs(filters.Arg("name", vaultHelperServiceName), filters.Arg("node", hostname))
	// tasks, err := cli.TaskList(context.Background(), types.TaskListOptions{
	// Filters: args,
	// })
	// if err != nil {
	// log.Fatalf("Error listing tasks for service %q: %v", vaultHelperServiceName, err)
	// }

	// Look for a task from the helper service
	// var secretZero string
	// for _, task := range tasks {
	// 	// avoid services with the name as a shared prefix but different ID
	// 	if task.ServiceID != service.ID {
	// 		continue
	// 	}
	// 	// Use a task that has a container
	// 	containerStatus := task.Status.ContainerStatus
	// 	if containerStatus != nil {
	// 		// Create an exec to later read its output
	// 		response, err := cli.ContainerExecCreate(context.Background(), containerStatus.ContainerID, types.ExecConfig{
	// 			AttachStdout: true,
	// 			Detach:       false,
	// 			Tty:          false,
	// 			Cmd:          []string{"cat", fmt.Sprintf("/run/secrets/%s", secretZeroName)},
	// 		})
	// 		if err != nil {
	// 			log.Fatalf("Error creating exec: %v", err)
	// 		}
	// 		// Start and attach to exec to read its output
	// 		resp, err := cli.ContainerExecAttach(context.Background(), response.ID, types.ExecStartCheck{
	// 			Detach: false,
	// 			Tty:    false,
	// 		})
	// 		if err != nil {
	// 			log.Fatalf("Error attaching to exec: %v", err)
	// 		}
	// 		defer resp.Close()
	// 		// Read the output into a buffer and convert to a string
	// 		buf := new(bytes.Buffer)
	// 		if _, err := stdcopy.StdCopy(buf, buf, resp.Reader); err != nil {
	// 			if err != nil {
	// 				log.Fatalf("Error reading secret zero: %v", err)
	// 			}
	// 		}
	// 		secretZero = buf.String()
	// 		break
	// 	}
	// }
	// if len(secretZero) == 0 {
	// 	log.Fatalf("Failed to read a Vault token from the helper service %q", vaultHelperServiceName)
	// }

	// Create a Vault client
	vaultToken := os.Getenv("VAULT_TOKEN")
	var vaultClient *vaultapi.Client
	vaultConfig := vaultapi.DefaultConfig()
	if c, err := vaultapi.NewClient(vaultConfig); err != nil {
		log.Fatalf("Error creating Vault client: %v", err)
	} else {
		c.SetToken(vaultToken)
		vaultClient = c
	}

	// Create the driver
	d := vaultSecretsDriver{
		dockerClient: cli,
		vaultClient:  vaultClient,
	}
	h := secrets.NewHandler(d)

	// Parse policy template
	// policyTemplateExpression = os.Getenv("policy-template")
	// if len(policyTemplateExpression) > 0 {
	// 	tmpl, err := template.New("policies").Parse(policyTemplateExpression)
	// 	if err != nil {
	// 		log.Fatalf("Error parsing policy template %q: %v", policyTemplateExpression, err)
	// 	}
	// 	policyTemplate = tmpl
	// }

	// Serve plugin
	log.Infof("Listening on %s", socketAddress)
	log.Error(h.ServeUnix(socketAddress, 0))
}
